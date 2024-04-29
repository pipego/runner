package maint

import (
	"context"

	"github.com/hashicorp/go-hclog"

	"github.com/pipego/runner/config"
)

type Maint interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Clock(context.Context, int64, bool) (int64, bool, int64, error)
}

type Config struct {
	Config config.Config
	Logger hclog.Logger
}

type maint struct {
	cfg *Config
}

func New(_ context.Context, cfg *Config) Maint {
	return &maint{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (m *maint) Init(_ context.Context) error {
	return nil
}

func (m *maint) Deinit(_ context.Context) error {
	return nil
}

func (m *maint) Clock(ctx context.Context, clockTime int64, clockSync bool) (diffTime int64, diffDangerous bool, syncStatus int64,
	err error) {
	// TBD: FIXME
	return diffTime, diffDangerous, syncStatus, err
}
