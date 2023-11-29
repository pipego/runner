package task

import (
	"bufio"
	"context"
	"os/exec"
	"time"

	"github.com/pkg/errors"

	"github.com/pipego/runner/config"
)

const (
	EOF = "EOF"
	Log = 5000
)

type Task interface {
	Init(context.Context, int) error
	Deinit(context.Context) error
	Run(context.Context, string, []string, []string) error
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

type task struct {
	cfg *Config
	log Livelog
}

func New(_ context.Context, cfg *Config) Task {
	return &task{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (t *task) Init(_ context.Context, log int) error {
	l := Log

	if log > 0 {
		l = log
	}

	t.log = Livelog{
		Error: make(chan error, l),
		Line:  make(chan *Line, l),
	}

	return nil
}

func (t *task) Deinit(_ context.Context) error {
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

	cmd := exec.CommandContext(ctx, n, a...)
	cmd.Env = append(cmd.Environ(), envs...)
	cmd.Stderr = cmd.Stdout

	reader, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(reader)

	_ = cmd.Start()
	t.routine(ctx, scanner)

	go func(cmd *exec.Cmd) {
		_ = cmd.Wait()
	}(cmd)

	return nil
}

func (t *task) Tail(_ context.Context) Livelog {
	return t.log
}

func (t *task) routine(ctx context.Context, scanner *bufio.Scanner) {
	go func(_ context.Context, scanner *bufio.Scanner, log Livelog) {
		p := 1
		for scanner.Scan() {
			select {
			case log.Line <- &Line{Pos: int64(p), Time: time.Now().UnixNano(), Message: scanner.Text()}:
				p += 1
			}
		}
		log.Line <- &Line{Pos: int64(p), Time: time.Now().UnixNano(), Message: EOF}
	}(ctx, scanner, t.log)
}
