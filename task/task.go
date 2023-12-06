package task

import (
	"bufio"
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/pipego/runner/config"
)

const (
	lineSep  = '\n'
	logLen   = 5000
	splitLen = 500   // split length of line (BOL included)
	tagBOL   = "BOL" // break of line
	tagEOF   = "EOF" // end of file
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
	Line chan *Line
}

type Line struct {
	Pos     int64
	Time    int64
	Message []byte
}

type task struct {
	cfg *Config
	log Livelog
	wg  sync.WaitGroup
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
	l := logLen

	if log > 0 {
		l = log
	}

	t.log = Livelog{
		Line: make(chan *Line, l),
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

	stdout, _ := cmd.StdoutPipe()
	readerStdout := bufio.NewReader(stdout)

	stderr, _ := cmd.StderrPipe()
	readerStderr := bufio.NewReader(stderr)

	_ = cmd.Start()
	t.routine(ctx, readerStdout, readerStderr)

	go func(cmd *exec.Cmd) {
		_ = cmd.Wait()
	}(cmd)

	return nil
}

func (t *task) Tail(_ context.Context) Livelog {
	return t.log
}

func (t *task) routine(ctx context.Context, stdout, stderr *bufio.Reader) {
	l := splitLen - len(tagBOL)
	p := 1

	helper := func(_ context.Context, reader *bufio.Reader, log Livelog) {
		for {
			line, err := reader.ReadBytes(lineSep)
			if err != nil {
				break
			}
			r := len(line) / l
			m := len(line) % l
			for i := 0; i < r; i++ {
				var b []byte
				b = append(b, line[i*l:(i+1)*l]...)
				b = append(b, []byte(tagBOL)...)
				log.Line <- &Line{Pos: int64(p), Time: time.Now().UnixNano(), Message: b}
			}
			log.Line <- &Line{Pos: int64(p), Time: time.Now().UnixNano(), Message: line[len(line)-m:]}
			p += 1
		}
		t.wg.Done()
	}

	t.wg.Add(1)
	go helper(ctx, stdout, t.log)

	t.wg.Add(1)
	go helper(ctx, stderr, t.log)

	go func() {
		t.wg.Wait()
		t.log.Line <- &Line{Pos: int64(p), Time: time.Now().UnixNano(), Message: []byte(tagEOF)}
		close(t.log.Line)
	}()
}
