package task

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
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
	langBash   = "bash"
	langTarget = "/workspace"

	lineCount = 1000
	lineSep   = '\n'
	lineWidth = 500 // BOL appended

	routineNum = -1

	tagBOL = "BOL" // break of line
	tagEOF = "EOF" // end of file
)

type Task interface {
	Init(context.Context, int, Language) error
	Deinit(context.Context) error
	Run(context.Context, string, []string, []string, string) error
	Tail(ctx context.Context) Log
}

type Config struct {
	Config config.Config
	Logger hclog.Logger
}

type Language struct {
	Name     string
	Artifact Artifact
}

type Artifact struct {
	Image   string
	User    string
	Pass    string
	Cleanup bool
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
	cfg     *Config
	lang    Language
	log     Log
	wg      sync.WaitGroup
	_client *client.Client
}

func New(_ context.Context, cfg *Config) Task {
	return &task{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (t *task) Init(ctx context.Context, width int, lang Language) error {
	var w int

	t.lang = lang

	if width > 0 {
		w = width
	} else {
		w = lineWidth
	}

	t.log = Log{
		Line:  chanx.NewUnboundedChan[*Line](ctx, lineCount),
		Width: w,
	}

	if t.lang.Name != langBash {
		t._client, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

		if err := t.pullImage(ctx, t.lang.Artifact.Image, t.lang.Artifact.User, t.lang.Artifact.Pass); err != nil {
			return errors.Wrap(err, "failed to pull image")
		}
	}

	return nil
}

func (t *task) Deinit(ctx context.Context) error {
	if t.lang.Name != langBash {
		if t.lang.Artifact.Cleanup {
			_ = t.removeImage(ctx, t.lang.Artifact.Image)
		}

		if t._client != nil {
			_ = t._client.Close()
		}
	}

	return nil
}

func (t *task) Run(ctx context.Context, _ string, env, cmd []string, file string) error {
	var stdout, stderr *bufio.Reader
	var err error

	if t.lang.Name == langBash {
		stdout, stderr, err = t.runBash(ctx, env, cmd, file)
	} else {
		stdout, stderr, err = t.runLanguage(ctx, env, file)
	}

	if err != nil {
		return errors.Wrap(err, "failed to run task")
	}

	t.routine(ctx, stdout, stderr)

	return nil
}

func (t *task) Tail(_ context.Context) Log {
	return t.log
}

// nolint:ineffassign
func (t *task) runBash(ctx context.Context, env, cmd []string, file string) (stdoutReader, stderrReader *bufio.Reader, err error) {
	var name string
	var arg []string

	name, err = exec.LookPath(langBash)

	if len(cmd) != 0 {
		arg = []string{"-c", strings.Join(cmd, " ")}
	} else {
		if file != "" {
			arg = []string{"-c", file}
		} else {
			return nil, nil, errors.New("invalid file")
		}
	}

	c := exec.CommandContext(ctx, name, arg...)
	c.Env = append(c.Environ(), env...)

	stdoutPipe, _ := c.StdoutPipe()
	stdoutReader = bufio.NewReader(stdoutPipe)

	stderrPipe, _ := c.StderrPipe()
	stdoutReader = bufio.NewReader(stderrPipe)

	_ = c.Start()

	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(routineNum)

	g.Go(func() error {
		_ = c.Wait()
		return nil
	})

	if err = g.Wait(); err != nil {
		return nil, nil, errors.Wrap(err, "failed to wait")
	}

	return stdoutReader, stderrReader, nil
}

func (t *task) runLanguage(ctx context.Context, env []string, file string) (stdoutReader, stderrReader *bufio.Reader, err error) {
	name := []string{filepath.Join(string(os.PathSeparator), langTarget, filepath.Base(file))}
	source := filepath.Dir(file)

	id, stdoutReader, err := t.runContainer(ctx, t.lang.Artifact.Image, env, name, source, langTarget)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to run container")
	}

	_ = t.removeContainer(ctx, id)

	return stdoutReader, stderrReader, nil
}

func (t *task) pullImage(ctx context.Context, name, user, pass string) error {
	_config := registry.AuthConfig{
		Username: user,
		Password: pass,
	}

	encodedJSON, _ := json.Marshal(_config)
	auth := base64.URLEncoding.EncodeToString(encodedJSON)

	options := image.PullOptions{}

	if user != "" && pass != "" {
		options.RegistryAuth = auth
	}

	out, err := t._client.ImagePull(ctx, name, options)
	if err != nil {
		return errors.Wrap(err, "failed to pull image")
	}

	defer func(out io.ReadCloser) {
		_ = out.Close()
	}(out)

	_, _ = io.Copy(os.Stdout, out)

	return nil
}

func (t *task) runContainer(ctx context.Context, name string, env, cmd []string, source, target string) (id string,
	stdout *bufio.Reader, err error) {
	_config := &container.Config{
		Image: name,
		Env:   env,
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

	resp, err := t._client.ContainerCreate(ctx, _config, hostConfig, &network.NetworkingConfig{},
		nil, "")
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create container")
	}

	if err = t._client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", nil, errors.Wrap(err, "failed to start container")
	}

	statusCh, errCh := t._client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", nil, errors.Wrap(err, "failed to wait container")
		}
	case <-statusCh:
	}

	reader, err := t._client.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to log container")
	}

	stdout = bufio.NewReader(reader)

	return resp.ID, stdout, nil
}

func (t *task) removeContainer(ctx context.Context, id string) error {
	options := container.RemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   true,
		Force:         true,
	}

	defer func(ctx context.Context, c *client.Client) {
		_, _ = c.ContainersPrune(ctx, filters.Args{})
	}(ctx, t._client)

	_ = t._client.ContainerRemove(ctx, id, options)

	return nil
}

func (t *task) removeImage(ctx context.Context, id string) error {
	options := image.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	}

	defer func(ctx context.Context, c *client.Client) {
		_, _ = c.ImagesPrune(ctx, filters.Args{})
	}(ctx, t._client)

	_, _ = t._client.ImageRemove(ctx, id, options)

	return nil
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
