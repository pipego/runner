// Package runner implements a directed acyclic graph task runner with deterministic teardown.
// it is similar to package errgroup, in that it runs multiple tasks in parallel and returns
// the first error it encounters. Users define a Runner as a set vertices (functions) and edges
// between them. During Run, the directed acyclec graph will be validated and each vertex
// will run in parallel as soon as it's dependencies have been resolved. The Runner will only
// return after all running goroutines have stopped.
package runner

import (
	"bufio"
	"context"
	"os/exec"
	"time"

	"github.com/pkg/errors"

	"github.com/pipego/runner/config"
)

const (
	Log = 5000
)

type Runner interface {
	Init(context.Context, int) error
	Deinit(context.Context) error
	Run(context.Context, string, []string, context.CancelFunc) error
	Tail(ctx context.Context) Livelog
}

type Config struct {
	Config config.Config
}

type Livelog struct {
	Error chan error
	Line  chan *Line
}

type Line struct {
	Pos     int64
	Time    int64
	Message string
}

type runner struct {
	cfg *Config
	log Livelog
}

func New(_ context.Context, cfg *Config) Runner {
	return &runner{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (r *runner) Init(_ context.Context, log int) error {
	l := Log

	if log > 0 {
		l = log
	}

	r.log = Livelog{
		Error: make(chan error, l),
		Line:  make(chan *Line, l),
	}

	return nil
}

func (r *runner) Deinit(_ context.Context) error {
	return nil
}

func (r *runner) Run(ctx context.Context, _ string, args []string, cancel context.CancelFunc) error {
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

	cmd := exec.CommandContext(ctx, n, a...)

	reader, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	_ = cmd.Start()

	scanner := bufio.NewScanner(reader)
	r.routine(ctx, scanner)

	go func(cmd *exec.Cmd, cancel context.CancelFunc) {
		_ = cmd.Wait()
		cancel()
	}(cmd, cancel)

	return nil
}

func (r *runner) Tail(_ context.Context) Livelog {
	return r.log
}

func (r *runner) routine(ctx context.Context, scanner *bufio.Scanner) {
	go func(ctx context.Context, scanner *bufio.Scanner, log Livelog) {
		p := 1
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				break
			case log.Line <- &Line{Pos: int64(p), Time: time.Now().UnixNano(), Message: scanner.Text()}:
				p += 1
			}
		}
	}(ctx, scanner, r.log)
}
