package glance

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/pipego/runner/config"
)

const (
	Base     = 10
	Bitwise  = 30
	Duration = 2 * time.Second
	GB       = "GB"
	Milli    = 1000

	Current = "."
	Parent  = ".."

	Root = "/"
	Dev  = "/dev/"
	Home = "/home"
)

type Glance interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Dir(context.Context, string) ([]Entry, error)
	File(context.Context, string, int64) (string, bool, error)
	Sys(context.Context) (Resource, Resource, Stats, Stats, Stats, string, string, error)
}

type Config struct {
	Config config.Config
}

type Entry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
	Time  string `json:"time"`
	User  string `json:"user"`
	Group string `json:"group"`
	Mode  string `json:"mode"`
}

type Resource struct {
	MilliCPU int64 `json:"milliCPU"`
	Memory   int64 `json:"memory"`
	Storage  int64 `json:"storage"`
}

type Stats struct {
	Total string `json:"total"`
	Used  string `json:"used"`
}

type glance struct {
	cfg *Config
}

func New(_ context.Context, cfg *Config) Glance {
	return &glance{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (g *glance) Init(_ context.Context) error {
	return nil
}

func (g *glance) Deinit(_ context.Context) error {
	return nil
}

func (g *glance) Dir(_ context.Context, path string) (entries []Entry, err error) {
	if stat, e := os.Lstat(path); e != nil {
		return entries, errors.Wrap(e, "failed to list file")
	} else if !stat.IsDir() {
		return entries, errors.New("invalid dir")
	}

	if ent, e := g.entry(path, Current); e == nil {
		ent.Name = Current
		entries = append(entries, ent)
	} else {
		return entries, errors.Wrap(e, "faied to get entry")
	}

	if ent, e := g.entry(path, Parent); e == nil {
		ent.Name = Parent
		entries = append(entries, ent)
	} else {
		return entries, errors.Wrap(e, "faied to get entry")
	}

	buf, e := os.ReadDir(path)
	if err != nil {
		return entries, errors.Wrap(e, "faied to read dir")
	}

	for _, item := range buf {
		if ent, e := g.entry(path, item.Name()); e == nil {
			entries = append(entries, ent)
		}
	}

	return entries, nil
}

func (g *glance) File(_ context.Context, path string, maxSize int64) (content string, readable bool, err error) {
	if !g.isText(path) {
		return content, false, errors.New("invalid text")
	}

	if !g.validSize(path, maxSize) {
		return content, false, errors.New("invalid size")
	}

	content, e := g.readFile(path)
	if e != nil {
		return content, false, errors.Wrap(e, "failed to read file")
	}

	return content, true, nil
}

// nolint: gocritic
func (g *glance) Sys(_ context.Context) (allocatable, requested Resource, _cpu, _memory, _storage Stats, _host, _os string, err error) {
	allocatable.MilliCPU, requested.MilliCPU = g.milliCPU()
	allocatable.Memory, requested.Memory = g.memory()
	allocatable.Storage, requested.Storage = g.storage()

	_cpu, _memory, _storage = g.stats(allocatable, requested)
	_host = g._host()
	_os = g._os()

	return allocatable, requested, _cpu, _memory, _storage, _host, _os, nil
}

func (g *glance) entry(dname, fname string) (Entry, error) {
	var uname, gname string

	s, err := os.Lstat(filepath.Join(dname, fname))
	if err != nil {
		return Entry{}, errors.Wrap(err, "failed to list file")
	}

	uid, gid := g.statId(s)

	if _user, err := user.LookupId(uid); err == nil {
		uname = _user.Name
	} else {
		uname = uid
	}

	if _group, err := user.LookupGroupId(gid); err == nil {
		gname = _group.Name
	} else {
		gname = gid
	}

	return Entry{
		Name:  fname,
		IsDir: s.IsDir(),
		Size:  s.Size(),
		Time:  s.ModTime().Format("2006-01-02 15:04:05"),
		User:  uname,
		Group: gname,
		Mode:  s.Mode().String(),
	}, nil
}

func (g *glance) isText(name string) bool {
	stat, e := os.Lstat(name)
	if e != nil {
		return false
	}

	if mode := stat.Mode(); !mode.IsRegular() {
		return false
	}

	file, err := os.Open(name)
	if err != nil {
		return false
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	scanner.Scan()

	return utf8.ValidString(scanner.Text())
}

func (g *glance) validSize(name string, size int64) bool {
	s, err := os.Lstat(name)
	if err != nil {
		return false
	}

	if s.Size() > size {
		return false
	}

	return true
}

func (g *glance) readFile(name string) (string, error) {
	buf, err := os.ReadFile(name)
	if err != nil {
		return "", errors.Wrap(err, "failed to read file")
	}

	return base64.StdEncoding.EncodeToString(buf), nil
}

func (g *glance) milliCPU() (alloc, request int64) {
	c, err := cpu.Counts(true)
	if err != nil {
		return -1, -1
	}

	if c*Milli > math.MaxInt64 {
		return -1, -1
	}

	// FIXME: Got error on MacOS 10.13.6
	p, err := cpu.Percent(Duration, false)
	if err != nil {
		return -1, -1
	}

	used := float64(c) * p[0] * 0.01
	if used > math.MaxInt64 {
		return -1, -1
	}

	return int64(c * Milli), int64(used * Milli)
}

func (g *glance) memory() (alloc, request int64) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return -1, -1
	}

	if v.Total > math.MaxInt64 || v.Used > math.MaxInt64 {
		return -1, -1
	}

	return int64(v.Total), int64(v.Used)
}

func (g *glance) storage() (alloc, request int64) {
	helper := func(path string) bool {
		found := false
		p, _ := disk.Partitions(false)
		for _, item := range p {
			if strings.HasPrefix(item.Device, Dev) && item.Mountpoint == path {
				found = true
				break
			}
		}
		return found
	}

	r, err := disk.Usage(Root)
	if err != nil {
		return -1, -1
	}

	total := r.Total
	used := r.Used

	if helper(Home) {
		h, err := disk.Usage(Home)
		if err != nil {
			return -1, -1
		}
		total = h.Total
		used = h.Used
	}

	if total > math.MaxInt64 || used > math.MaxInt64 {
		return -1, -1
	}

	return int64(total), int64(used)
}

func (g *glance) stats(alloc, req Resource) (_cpu, memory, storage Stats) {
	_cpu.Total = strconv.FormatInt(alloc.MilliCPU/Milli, Base) + " CPU"
	_cpu.Used = strconv.FormatInt(req.MilliCPU*100/alloc.MilliCPU, Base) + "%"

	memory.Total = strconv.FormatInt(alloc.Memory>>Bitwise, Base) + " " + GB
	memory.Used = strconv.FormatInt(req.Memory>>Bitwise, Base) + " " + GB

	storage.Total = strconv.FormatInt(alloc.Storage>>Bitwise, Base) + " " + GB
	storage.Used = strconv.FormatInt(req.Storage>>Bitwise, Base) + " " + GB

	return _cpu, memory, storage
}

func (g *glance) _host() string {
	conn, _ := net.Dial("udp", "8.8.8.8:8")
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)

	buf := conn.LocalAddr().(*net.UDPAddr)

	return strings.Split(buf.String(), ":")[0]
}

func (g *glance) _os() string {
	info, _ := host.Info()
	caser := cases.Title(language.BrazilianPortuguese)

	return fmt.Sprintf("%s %s", caser.String(info.Platform), info.PlatformVersion)
}
