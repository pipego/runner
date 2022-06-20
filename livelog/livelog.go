package livelog

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/pipego/runner/config"
)

type Livelog interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Create(context.Context, int64) error
	Delete(context.Context, int64) error
	Write(context.Context, int64, *Line) error
	Tail(context.Context, int64) (<-chan *Line, <-chan error)
}

type Config struct {
	Config config.Config
}

type Line struct {
	Pos     int64  `json:"pos"`
	Time    int64  `json:"time"`
	Message string `json:"message"`
}

type livelog struct {
	sync.Mutex
	cfg     *Config
	streams map[int64]*stream
}

func New(_ context.Context, cfg *Config) Livelog {
	return &livelog{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (l *livelog) Init(_ context.Context) error {
	l.streams = make(map[int64]*stream)
	return nil
}

func (l *livelog) Deinit(_ context.Context) error {
	for key := range l.streams {
		delete(l.streams, key)
	}

	l.streams = nil

	return nil
}

func (l *livelog) Create(ctx context.Context, id int64) error {
	l.Lock()
	defer l.Unlock()

	l.streams[id] = newStream(ctx)

	return nil
}

func (l *livelog) Delete(ctx context.Context, id int64) error {
	l.Lock()
	defer l.Unlock()

	s, ok := l.streams[id]
	if ok {
		delete(l.streams, id)
	} else {
		return errors.New("invalid id")
	}

	return s.close(ctx)
}

func (l *livelog) Write(ctx context.Context, id int64, line *Line) error {
	l.Lock()
	defer l.Unlock()

	s, ok := l.streams[id]
	if !ok {
		return errors.New("invalid id")
	}

	return s.write(ctx, line)
}

func (l *livelog) Tail(ctx context.Context, id int64) (line <-chan *Line, err <-chan error) {
	l.Lock()
	defer l.Unlock()

	s, ok := l.streams[id]
	if !ok {
		return nil, nil
	}

	return s.subscribe(ctx)
}
