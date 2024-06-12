package task

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os/exec"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"
	"github.com/smallnest/chanx"
	"golang.org/x/sync/errgroup"

	"github.com/pipego/runner/config"
)

const (
	authName = ""
	authPass = ""

	lineCount = 1000
	lineSep   = '\n'
	lineWidth = 500 // BOL appended

	routineNum = -1

	tagBOL = "BOL" // break of line
	tagEOF = "EOF" // end of file
)

var (
	languages = map[string]string{
		"go":     "craftslab/go:latest",
		"groovy": "craftslab/groovy:latest",
		"java":   "craftslab/java:latest",
		"python": "craftslab/python:latest",
		"rust":   "craftslab/rust:latest",
	}
)

type Task interface {
	Init(context.Context, int) error
	Deinit(context.Context) error
	Run(context.Context, string, []string, []string, Language) error
	Tail(ctx context.Context) Log
}

type Config struct {
	Config config.Config
	Logger hclog.Logger
}

type Language struct {
	Name  string
	Image string
}

type Log struct {
	Line  *chanx.UnboundedChan[*Line]
	Width int
}

type Line struct {
	Pos     int64
	Time    int64
	Message string
}

type task struct {
	cfg    *Config
	client *client.Client
	log    Log
	wg     sync.WaitGroup
}

type artifactAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func New(_ context.Context, cfg *Config) Task {
	_client, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	return &task{
		cfg:    cfg,
		client: _client,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (t *task) Init(ctx context.Context, width int) error {
	w := lineWidth

	if width > 0 {
		w = width
	}

	t.log = Log{
		Line:  chanx.NewUnboundedChan[*Line](ctx, lineCount),
		Width: w,
	}

	return nil
}

func (t *task) Deinit(_ context.Context) error {
	if t.client != nil {
		_ = t.client.Close()
	}

	return nil
}

func (t *task) Run(ctx context.Context, _ string, envs, args []string, lang Language) error {
	var a []string
	var n string
	var err error

	if len(args) > 1 {
		n, err = exec.LookPath(args[0])
		a = args[1:]
	} else if len(args) == 1 {
		n, err = exec.LookPath(args[0])
	} else {
		return errors.New("invalid args")
	}

	if err != nil {
		return errors.New("name not found")
	}

	// TBD: FIXME
	// Run t.runLanguage

	cmd := exec.CommandContext(ctx, n, a...)
	cmd.Env = append(cmd.Environ(), envs...)

	stdout, _ := cmd.StdoutPipe()
	readerStdout := bufio.NewReader(stdout)

	stderr, _ := cmd.StderrPipe()
	readerStderr := bufio.NewReader(stderr)

	_ = cmd.Start()
	t.routine(ctx, readerStdout, readerStderr)

	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(routineNum)

	g.Go(func() error {
		_ = cmd.Wait()
		return nil
	})

	if err = g.Wait(); err != nil {
		return errors.Wrap(err, "failed to wait")
	}

	return nil
}

func (t *task) Tail(_ context.Context) Log {
	return t.log
}

func (t *task) routine(ctx context.Context, stdout, stderr *bufio.Reader) {
	w := t.log.Width - utf8.RuneCountInString(tagBOL)
	p := 1

	helper := func(_ context.Context, reader *bufio.Reader, log Log) {
		for {
			line, err := reader.ReadBytes(lineSep)
			if err != nil {
				break
			}
			s := string(line)
			r := utf8.RuneCountInString(s) / w
			m := utf8.RuneCountInString(s) % w
			b := []rune(s)
			for i := 0; i < r; i++ {
				log.Line.In <- &Line{Pos: int64(p), Time: time.Now().UnixNano(), Message: string(b[i*w:(i+1)*w]) + tagBOL}
			}
			log.Line.In <- &Line{Pos: int64(p), Time: time.Now().UnixNano(), Message: string(b[len(b)-m:])}
			p += 1
		}
		t.wg.Done()
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(routineNum)

	t.wg.Add(1)
	g.Go(func() error {
		helper(ctx, stdout, t.log)
		return nil
	})

	t.wg.Add(1)
	g.Go(func() error {
		helper(ctx, stderr, t.log)
		return nil
	})

	g.Go(func() error {
		t.wg.Wait()
		t.cfg.Logger.Debug("routine: Message: tagEOF")
		t.log.Line.In <- &Line{Pos: int64(p), Time: time.Now().UnixNano(), Message: tagEOF}
		close(t.log.Line.In)
		return nil
	})

	_ = g.Wait()
}

func (t *task) runLanguage(ctx context.Context, name, file, source, target string) ([]byte, error) {
	if err := t.pullImage(ctx, name); err != nil {
		return nil, errors.Wrap(err, "failed to pull image")
	}

	buf, err := t.runImage(ctx, name, []string{file}, source, target)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run image")
	}

	if err := t.removeImage(ctx, name); err != nil {
		return nil, errors.Wrap(err, "failed to remove image")
	}

	return buf, nil
}

func (t *task) pullImage(ctx context.Context, name string) error {
	_config := registry.AuthConfig{
		Username: authName,
		Password: authPass,
	}

	encodedJSON, _ := json.Marshal(_config)
	auth := base64.URLEncoding.EncodeToString(encodedJSON)

	options := image.PullOptions{
		RegistryAuth: auth,
	}

	out, err := t.client.ImagePull(ctx, name, options)
	if err != nil {
		return errors.Wrap(err, "failed to pull image")
	}

	_ = out.Close()

	return nil
}

func (t *task) runImage(ctx context.Context, name string, cmd []string, source, target string) ([]byte, error) {
	var buf []byte

	_config := &container.Config{
		Image: name,
		Cmd:   cmd,
		Tty:   false,
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: source,
				Target: target,
			},
		},
	}

	resp, err := t.client.ContainerCreate(ctx, _config, hostConfig, &network.NetworkingConfig{},
		nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create container")
	}

	defer func(ctx context.Context, c *client.Client, id string) {
		opts := container.RemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   true,
			Force:         true,
		}
		_ = c.ContainerRemove(ctx, id, opts)
	}(ctx, t.client, resp.ID)

	if err = t.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, errors.Wrap(err, "failed to start container")
	}

	statusCh, errCh := t.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, errors.Wrap(err, "failed to wait container")
		}
	case <-statusCh:
	}

	out, err := t.client.ContainerLogs(ctx, resp.ID, container.LogsOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to log container")
	}

	buf, err = io.ReadAll(out)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read log")
	}

	return buf, nil
}

func (t *task) removeImage(ctx context.Context, name string) error {
	options := image.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	}

	if _, err := t.client.ImageRemove(ctx, name, options); err != nil {
		return errors.Wrap(err, "failed to remove image")
	}

	return nil
}
