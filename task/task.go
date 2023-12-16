package task

import (
	"bufio"
	"context"
	"os/exec"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/pipego/runner/chanx"
	"github.com/pipego/runner/config"
)

const (
	lineCount = 5000
	lineSep   = '\n'
	lineWidth = 500   // BOL appended
	tagBOL    = "BOL" // break of line
	tagEOF    = "EOF" // end of file
)

type Task interface {
	Init(context.Context, int) error
	Deinit(context.Context) error
	Run(context.Context, string, []string, []string) error
	Tail(ctx context.Context) Log
}

type Config struct {
	Config config.Config
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
	cfg *Config
	log Log
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

	t.wg.Add(1)
	go helper(ctx, stdout, t.log)

	t.wg.Add(1)
	go helper(ctx, stderr, t.log)

	go func() {
		t.wg.Wait()
		t.log.Line.In <- &Line{Pos: int64(p), Time: time.Now().UnixNano(), Message: tagEOF}
		close(t.log.Line.In)
	}()
}
