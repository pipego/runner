package task

import (
	"bufio"
	"context"
	"os/exec"
	"sync"
	"time"
	"unicode/utf8"

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
	Init(context.Context, int) error
	Deinit(context.Context) error
	Run(context.Context, string, []string, []string) error
	Tail(ctx context.Context) Log
}

type Config struct {
	Config config.Config
	Logger hclog.Logger
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
