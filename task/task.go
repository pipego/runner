package task

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"os/exec"
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
	Run(context.Context, string, []string, []string) error
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

	t._client, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err := t.pullImage(ctx, t.lang.Artifact.Image, t.lang.Artifact.User, t.lang.Artifact.Pass); err != nil {
		return errors.Wrap(err, "failed to pull image")
	}

	return nil
}

func (t *task) Deinit(ctx context.Context) error {
	if t.lang.Artifact.Cleanup {
		_ = t.removeImage(ctx, t.lang.Artifact.Image)
	}

	if t._client != nil {
		_ = t._client.Close()
	}

	return nil
}

func (t *task) Run(ctx context.Context, _ string, envs, args []string) error {
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
	// Run runLanguage

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

func (t *task) runLanguage(ctx context.Context, cmd []string, source, target string) ([]byte, error) {
	id, out, err := t.runContainer(ctx, t.lang.Artifact.Image, cmd, source, target)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run container")
	}

	_ = t.removeContainer(ctx, id)

	return out, nil
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

func (t *task) runContainer(ctx context.Context, name string, cmd []string, source, target string) (id string, out []byte, err error) {
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

	buf, err := t._client.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to log container")
	}

	out, err = io.ReadAll(buf)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to read log")
	}

	return resp.ID, out, nil
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
