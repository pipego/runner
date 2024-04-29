package maint

import (
	"context"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"

	"github.com/pipego/runner/config"
)

const (
	clockNtpStat = "ntpstat"

	clockStatusSynchronised    = 0
	clockStatusNotSynchronised = 1
	clockStatusIndeterminant   = 2
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
	if clockTime <= 0 {
		return 0, true, clockStatusIndeterminant, errors.New("invalid clock time")
	}

	if clockSync {
		syncStatus = m.syncClock(ctx)
	}

	diffTime, diffDangerous = m.diffClock(ctx, clockTime)

	m.cfg.Logger.Debug("syncStatus", syncStatus)
	m.cfg.Logger.Debug("diffTime", diffTime)
	m.cfg.Logger.Debug("diffDangerous", diffDangerous)

	return diffTime, diffDangerous, syncStatus, nil
}

func (m *maint) syncClock(ctx context.Context) int64 {
	var exitError *exec.ExitError

	cmd := exec.CommandContext(ctx, clockNtpStat)

	if err := cmd.Start(); err != nil {
		return clockStatusIndeterminant
	}

	if err := cmd.Wait(); err != nil {
		if errors.As(err, &exitError) {
			return int64(exitError.ExitCode())
		}
	}

	return clockStatusSynchronised
}

func (m *maint) diffClock(_ context.Context, clockTime int64) (int64, bool) {
	// TBD: FIXME
	return 0, true
}
