package runner

import (
	"context"

	"github.com/pipego/runner/config"
)

type Runner interface {
	Run() error
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

func (r *runner) Run() error {
	return nil
}
