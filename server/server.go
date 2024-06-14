package server

import (
	"context"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	fl "github.com/pipego/runner/file"
	"github.com/pipego/runner/glance"
	"github.com/pipego/runner/maint"
	pb "github.com/pipego/runner/server/proto"
	"github.com/pipego/runner/task"
)

const (
	EOF    = "EOF" // end of file
	Kind   = "runner"
	Layout = "20060102150405"
)

type Server interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Run(context.Context) error
}

type Config struct {
	Addr   string
	Logger hclog.Logger
}

type server struct {
	cfg *Config
	pb.UnimplementedServerProtoServer
}

func New(_ context.Context, cfg *Config) Server {
	return &server{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (s *server) Init(ctx context.Context) error {
	return nil
}

func (s *server) Deinit(ctx context.Context) error {
	return nil
}

func (s *server) Run(_ context.Context) error {
	options := []grpc.ServerOption{grpc.MaxRecvMsgSize(math.MaxInt32), grpc.MaxSendMsgSize(math.MaxInt32)}

	g := grpc.NewServer(options...)
	pb.RegisterServerProtoServer(g, s)

	lis, _ := net.Listen("tcp", s.cfg.Addr)

	return g.Serve(lis)
}

// nolint:gocyclo
func (s *server) SendTask(srv pb.ServerProto_SendTaskServer) error {
	// Receive task
	name, file, params, commands, width, language, err := s.recvTask(srv)
	if err != nil {
		s.cfg.Logger.Error("SendTask", err.Error())
		return srv.Send(&pb.TaskReply{Error: err.Error()})
	}

	if len(file.GetContent()) != 0 && len(commands) != 0 {
		err := "file and commands not supported meanwhile"
		s.cfg.Logger.Error("SendTask", err)
		return srv.Send(&pb.TaskReply{Error: err})
	}

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	// Init file
	f, err := s.newFile(ctx)
	if err != nil {
		s.cfg.Logger.Error("SendTask", err.Error())
		return srv.Send(&pb.TaskReply{Error: err.Error()})
	}

	if err = f.Init(ctx); err != nil {
		s.cfg.Logger.Error("SendTask", err.Error())
		return srv.Send(&pb.TaskReply{Error: err.Error()})
	}

	defer func(ctx context.Context) {
		_ = f.Deinit(ctx)
	}(ctx)

	// Parse commands
	if len(commands) != 0 {
		commands = []string{"bash", "-c", strings.Join(commands, " ")}
	} else if len(file.GetContent()) != 0 {
		n, e := s.loadFile(ctx, f, file.GetContent(), file.GetGzip())
		defer func(ctx context.Context, n string) {
			_ = f.Remove(ctx, n)
		}(ctx, n)
		if e != nil {
			s.cfg.Logger.Error("SendTask", e.Error())
			return srv.Send(&pb.TaskReply{Error: e.Error()})
		}
		commands = []string{"bash", "-c", n}
	}

	// Run task
	t, err := s.newTask(ctx)
	if err != nil {
		s.cfg.Logger.Error("SendTask", err.Error())
		return srv.Send(&pb.TaskReply{Error: err.Error()})
	}

	if err := t.Init(ctx, width, s.buildLanguage(ctx, language)); err != nil {
		s.cfg.Logger.Error("SendTask", err.Error())
		return srv.Send(&pb.TaskReply{Error: err.Error()})
	}

	defer func(ctx context.Context) {
		_ = t.Deinit(ctx)
	}(ctx)

	if err := t.Run(ctx, name, s.buildEnv(ctx, params), commands); err != nil {
		s.cfg.Logger.Error("SendTask", err.Error())
		return srv.Send(&pb.TaskReply{Error: err.Error()})
	}

	log := t.Tail(ctx)

L:
	for {
		select {
		case <-ctx.Done():
			break L
		case line, ok := <-log.Line.Out:
			if ok {
				s.cfg.Logger.Debug("SendTask: line", line)
				_ = srv.Send(&pb.TaskReply{
					Output: &pb.TaskOutput{
						Pos:     line.Pos,
						Time:    line.Time,
						Message: line.Message,
					}})
				if line.Message == EOF {
					break L
				}
			}
		}
	}

	return nil
}

// nolint:funlen
func (s *server) SendGlance(srv pb.ServerProto_SendGlanceServer) error {
	var allocatable, requested glance.Resource
	var content, _host, _os string
	var entBuf []*pb.GlanceEntry
	var entries []glance.Entry
	var procBuf []*pb.GlanceProcess
	var readable bool
	var _cpu, _memory, _storage glance.Stats
	var _processes []glance.Process

	dir, file, sys, err := s.recvGlance(srv)
	if err != nil {
		s.cfg.Logger.Error("SendGlance", err.Error())
		return srv.Send(&pb.GlanceReply{Error: err.Error()})
	}

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	g, err := s.newGlance(ctx)
	if err != nil {
		s.cfg.Logger.Error("SendGlance", err.Error())
		return srv.Send(&pb.GlanceReply{Error: err.Error()})
	}

	if err = g.Init(ctx); err != nil {
		s.cfg.Logger.Error("SendGlance", err.Error())
		return srv.Send(&pb.GlanceReply{Error: err.Error()})
	}

	defer func(ctx context.Context) {
		_ = g.Deinit(ctx)
	}(ctx)

	if dir.GetPath() != "" {
		entries, err = g.Dir(ctx, dir.GetPath())
		if err != nil {
			s.cfg.Logger.Error("SendGlance", err.Error())
			return srv.Send(&pb.GlanceReply{Error: err.Error()})
		}
		for _, item := range entries {
			entBuf = append(entBuf, &pb.GlanceEntry{
				Name:  item.Name,
				IsDir: item.IsDir,
				Size:  item.Size,
				Time:  item.Time,
				User:  item.User,
				Group: item.Group,
				Mode:  item.Mode,
			})
		}
	}

	if file.GetPath() != "" {
		content, readable, err = g.File(ctx, file.GetPath(), file.GetMaxSize())
		if err != nil {
			s.cfg.Logger.Error("SendGlance", err.Error())
			return srv.Send(&pb.GlanceReply{Error: err.Error()})
		}
	}

	if sys.GetEnable() {
		allocatable, requested, _cpu, _memory, _storage, _processes, _host, _os, err = g.Sys(ctx)
		if err != nil {
			s.cfg.Logger.Error("SendGlance", err.Error())
			return srv.Send(&pb.GlanceReply{Error: err.Error()})
		}
		helper := func(data []glance.Thread) []*pb.GlanceThread {
			var threads []*pb.GlanceThread
			for _, item := range data {
				threads = append(threads, &pb.GlanceThread{
					Name:    item.Name,
					Cmdline: item.Cmdline,
					Memory:  item.Memory,
					Time:    item.Time,
					Pid:     item.Pid,
				})
			}
			return threads
		}
		for _, item := range _processes {
			procBuf = append(procBuf, &pb.GlanceProcess{
				Process: &pb.GlanceThread{
					Name:    item.Process.Name,
					Cmdline: item.Process.Cmdline,
					Memory:  item.Process.Memory,
					Time:    item.Process.Time,
					Pid:     item.Process.Pid,
				},
				Threads: helper(item.Threads),
			})
		}
	}

	s.cfg.Logger.Debug("SendGlance: entries", entBuf)
	s.cfg.Logger.Debug("SendGlance: content", content)
	s.cfg.Logger.Debug("SendGlance: readable", readable)
	s.cfg.Logger.Debug("SendGlance: allocatable", allocatable)
	s.cfg.Logger.Debug("SendGlance: requested", requested)
	s.cfg.Logger.Debug("SendGlance: cpu", _cpu)
	s.cfg.Logger.Debug("SendGlance: memory", _memory)
	s.cfg.Logger.Debug("SendGlance: storage", _storage)
	s.cfg.Logger.Debug("SendGlance: processes", _processes)

	_ = srv.Send(&pb.GlanceReply{
		Dir: &pb.GlanceDirRep{
			Entries: entBuf,
		},
		File: &pb.GlanceFileRep{
			Content:  content,
			Readable: readable,
		},
		Sys: &pb.GlanceSysRep{
			Resource: &pb.GlanceResource{
				Allocatable: &pb.GlanceAllocatable{
					MilliCPU: allocatable.MilliCPU,
					Memory:   allocatable.Memory,
					Storage:  allocatable.Storage,
				},
				Requested: &pb.GlanceRequested{
					MilliCPU: requested.MilliCPU,
					Memory:   requested.Memory,
					Storage:  requested.Storage,
				},
			},
			Stats: &pb.GlanceStats{
				Cpu: &pb.GlanceCPU{
					Total: _cpu.Total,
					Used:  _cpu.Used,
				},
				Host: _host,
				Memory: &pb.GlanceMemory{
					Total: _memory.Total,
					Used:  _memory.Used,
				},
				Os: _os,
				Storage: &pb.GlanceStorage{
					Total: _storage.Total,
					Used:  _storage.Used,
				},
				Processes: procBuf,
			},
		}})

	return nil
}

func (s *server) SendMaint(srv pb.ServerProto_SendMaintServer) error {
	clock, _ := s.recvMaint(srv)

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	m, _ := s.newMaint(ctx)
	_ = m.Init(ctx)

	defer func(ctx context.Context) {
		_ = m.Deinit(ctx)
	}(ctx)

	syncStatus, diffTime, diffDangerous, _ := m.Clock(ctx, clock.GetTime(), clock.GetSync())

	s.cfg.Logger.Debug("SendMaint: syncStatus", syncStatus)
	s.cfg.Logger.Debug("SendMaint: diffTime", diffTime)
	s.cfg.Logger.Debug("SendMaint: diffDangerous", diffDangerous)

	_ = srv.Send(&pb.MaintReply{
		Clock: &pb.MaintClockRep{
			Sync: &pb.MaintClockSync{
				Status: syncStatus,
			},
			Diff: &pb.MaintClockDiff{
				Time:      diffTime,
				Dangerous: diffDangerous,
			},
		},
	})

	return nil
}

// nolint:gocritic
func (s *server) recvTask(srv pb.ServerProto_SendTaskServer) (name string, file *pb.TaskFile, params []*pb.TaskParam,
	commands []string, width int, language *pb.TaskLanguage, err error) {
	for {
		r, err := srv.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil, nil, nil, 0, nil, errors.Wrap(err, "failed to receive")
		}

		if r.Kind != Kind {
			return "", nil, nil, nil, 0, nil, errors.New("invalid kind")
		}

		name = r.Spec.Task.GetName()
		file = r.Spec.Task.GetFile()
		params = r.Spec.Task.GetParams()
		commands = r.Spec.Task.GetCommands()
		width = int(r.Spec.Task.GetLog().GetWidth())
		language = r.Spec.Task.GetLanguage()

		break
	}

	return name, file, params, commands, width, language, nil
}

func (s *server) newFile(ctx context.Context) (fl.File, error) {
	c := fl.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	c.Logger = s.cfg.Logger

	return fl.New(ctx, c), nil
}

func (s *server) loadFile(ctx context.Context, file fl.File, data []byte, gzip bool) (string, error) {
	var buf []byte
	var err error

	if gzip {
		buf, err = file.Unzip(ctx, data)
		if err != nil {
			return "", errors.Wrap(err, "failed to unzip")
		}
	} else {
		buf = data
	}

	suffix := time.Now().Format(Layout)
	name := filepath.Join(string(os.PathSeparator), "tmp", "pipego-runner-file-"+suffix)

	if err = file.Write(ctx, name, buf); err != nil {
		_ = file.Remove(ctx, name)
		return "", errors.Wrap(err, "failed to write")
	}

	if file.Type(ctx, name) != fl.Bash {
		_ = file.Remove(ctx, name)
		return "", errors.New("invalid type")
	}

	return name, nil
}

func (s *server) newTask(ctx context.Context) (task.Task, error) {
	c := task.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	c.Logger = s.cfg.Logger

	return task.New(ctx, c), nil
}

func (s *server) buildEnv(ctx context.Context, params []*pb.TaskParam) []string {
	var buf []string

	for _, item := range params {
		buf = append(buf, item.GetName()+"="+s.evalEnv(ctx, params, item.GetValue()))
	}

	return buf
}

func (s *server) evalEnv(ctx context.Context, params []*pb.TaskParam, data string) string {
	if strings.HasPrefix(data, "$") {
		if strings.HasPrefix(data, "$$") {
			return data
		}

		for _, item := range params {
			if item.GetName() == strings.TrimPrefix(data, "$") {
				return s.evalEnv(ctx, params, item.GetValue())
			}
		}
	}

	return data
}

func (s *server) buildLanguage(ctx context.Context, language *pb.TaskLanguage) task.Language {
	return task.Language{
		Name: language.GetName(),
		Artifact: task.Artifact{
			Image:   language.GetArtifact().GetImage(),
			User:    language.GetArtifact().GetUser(),
			Pass:    language.GetArtifact().GetPass(),
			Cleanup: language.GetArtifact().GetCleanup(),
		},
	}
}

// nolint:lll
func (s *server) recvGlance(srv pb.ServerProto_SendGlanceServer) (dir *pb.GlanceDirReq, file *pb.GlanceFileReq, sys *pb.GlanceSysReq, err error) {
	for {
		r, err := srv.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, nil, errors.Wrap(err, "failed to receive")
		}

		if r.Kind != Kind {
			return nil, nil, nil, errors.Wrap(err, "invalid kind")
		}

		dir = r.Spec.Glance.GetDir()
		file = r.Spec.Glance.GetFile()
		sys = r.Spec.Glance.GetSys()

		break
	}

	return dir, file, sys, nil
}

func (s *server) newGlance(ctx context.Context) (glance.Glance, error) {
	c := glance.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	c.Logger = s.cfg.Logger

	return glance.New(ctx, c), nil
}

func (s *server) recvMaint(srv pb.ServerProto_SendMaintServer) (clock *pb.MaintClockReq, err error) {
	for {
		r, err := srv.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrap(err, "failed to receive")
		}

		if r.Kind != Kind {
			return nil, errors.Wrap(err, "invalid kind")
		}

		clock = r.Spec.Maint.GetClock()

		break
	}

	return clock, nil
}

func (s *server) newMaint(ctx context.Context) (maint.Maint, error) {
	c := maint.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	c.Logger = s.cfg.Logger

	return maint.New(ctx, c), nil
}
