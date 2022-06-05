package livelog

import (
	"context"

	"github.com/pipego/runner/config"
)

type Livelog interface {
	Run(context.Context) error
}

type Config struct {
	Config config.Config
}

type livelog struct {
	cfg *Config
}

func New(_ context.Context, cfg *Config) Livelog {
	return &livelog{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (l *livelog) Run(ctx context.Context) error {
	return nil
}
