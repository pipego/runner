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
	"github.com/pipego/runner/livelog"
)

type Runner interface {
	Run(context.Context, string, []string, livelog.Livelog, context.CancelFunc) error
}

type Config struct {
	Config config.Config
}

type runner struct {
	cfg *Config
}

func New(_ context.Context, cfg *Config) Runner {
	return &runner{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (r *runner) Run(ctx context.Context, _ string, args []string, log livelog.Livelog, cancel context.CancelFunc) error {
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

	cmd := exec.Command(n, a...)

	// TODO: cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()

	_ = cmd.Start()

	scanner := bufio.NewScanner(stdout)
	r.routine(ctx, scanner, log)

	go func() {
		_ = cmd.Wait()
		cancel()
	}()

	return nil
}

func (r *runner) routine(ctx context.Context, scanner *bufio.Scanner, log livelog.Livelog) {
	go func() {
		p := 1
		for scanner.Scan() {
			if err := log.Write(ctx, livelog.ID, &livelog.Line{Pos: int64(p), Time: time.Now().Unix(), Message: scanner.Text()}); err != nil {
				return
			}
			p += 1
		}
	}()
}
