package maint

import (
	"context"
	"math"
	"os/exec"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"

	"github.com/pipego/runner/config"
)

const (
	clockDiffDangerous = 5000
	clockTimeDelay     = 8 * time.Second
	clockTimeServer    = "time.nist.gov"

	clockNtpDate    = "\"sudo ntpdate -s " + clockTimeServer + "\""
	clockNtpService = "\"sudo service ntp restart\""
	clockNtpStat    = "ntpstat"

	clockStatusSynchronized    = 0
	clockStatusNotSynchronized = 1
	clockStatusIndeterminant   = 2
)

var (
	clockStatus = map[int]string{
		clockStatusSynchronized:    "synchronized",
		clockStatusNotSynchronized: "unsynchronized",
		clockStatusIndeterminant:   "indeterminant",
	}
)

type Maint interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Clock(context.Context, int64, bool) (string, int64, bool, error)
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

func (m *maint) Clock(ctx context.Context, clockTime int64, clockSync bool) (syncStatus string, diffTime int64, diffDangerous bool,
	err error) {
	if clockTime <= 0 {
		return clockStatus[clockStatusIndeterminant], 0, true, errors.New("invalid clock time")
	}

	if clockSync {
		syncStatus = m.syncClock(ctx)
	}

	diffTime, diffDangerous = m.diffClock(ctx, clockTime)

	m.cfg.Logger.Debug("syncStatus", syncStatus)
	m.cfg.Logger.Debug("diffTime", diffTime)
	m.cfg.Logger.Debug("diffDangerous", diffDangerous)

	return syncStatus, diffTime, diffDangerous, nil
}

func (m *maint) syncClock(ctx context.Context) string {
	runDate := func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "bash", "-c", clockNtpDate)
		_ = cmd.Start()
		_ = cmd.Wait()
		return nil
	}

	runService := func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "bash", "-c", clockNtpService)
		_ = cmd.Start()
		_ = cmd.Wait()
		return nil
	}

	runStat := func(ctx context.Context) string {
		var exitError *exec.ExitError
		cmd := exec.CommandContext(ctx, clockNtpStat)
		if err := cmd.Start(); err != nil {
			return clockStatus[clockStatusIndeterminant]
		}
		if err := cmd.Wait(); err != nil {
			if errors.As(err, &exitError) {
				if _, ok := clockStatus[exitError.ExitCode()]; ok {
					return clockStatus[exitError.ExitCode()]
				} else {
					return clockStatus[clockStatusIndeterminant]
				}
			}
		}
		return clockStatus[clockStatusSynchronized]
	}

	_ = runDate(ctx)
	_ = runService(ctx)

	time.Sleep(clockTimeDelay)

	return runStat(ctx)
}

func (m *maint) diffClock(_ context.Context, clockTime int64) (diffTime int64, diffDangerous bool) {
	diffTime = clockTime - time.Now().Unix()
	diffDangerous = math.Abs(float64(diffTime)) > clockDiffDangerous

	return diffTime, diffDangerous
}
