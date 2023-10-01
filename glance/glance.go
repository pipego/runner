package glance

import (
	"context"
	"time"

	"github.com/pipego/runner/config"
)

const (
	Base     = 10
	Bitwise  = 30
	Duration = 2 * time.Second
	Milli    = 1000

	Dev  = "/dev/"
	Home = "/home"
	Root = "/"
)

type Glance interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Dir(context.Context) error
	File(context.Context) error
	Sys(context.Context) error
}

type Config struct {
	Config config.Config
}

type glance struct {
	cfg *Config
}

func New(_ context.Context, cfg *Config) Glance {
	return &glance{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (g *glance) Init(_ context.Context) error {
	// TODO: FIXME
	return nil
}

func (g *glance) Deinit(_ context.Context) error {
	// TODO: FIXME
	return nil
}

func (g *glance) Dir(ctx context.Context) error {
	// TODO: FIXME
	return nil
}

func (g *glance) File(ctx context.Context) error {
	// TODO: FIXME
	return nil
}

func (g *glance) Sys(ctx context.Context) error {
	// TODO: FIXME
	return nil
}
