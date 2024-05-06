package maint

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/pipego/runner/config"
)

func initMaint() maint {
	helper := func(name string) *config.Config {
		c := config.New()
		fi, _ := os.Open(name)
		defer func() {
			_ = fi.Close()
		}()
		buf, _ := io.ReadAll(fi)
		_ = yaml.Unmarshal(buf, c)
		return c
	}

	c := helper("../test/config.yml")

	return maint{
		cfg: &Config{
			Config: *c,
			Logger: hclog.New(&hclog.LoggerOptions{
				Name:  "maint",
				Level: hclog.LevelFromString("DEBUG"),
			}),
		},
	}
}

// nolint:dogsled
func TestClock(t *testing.T) {
	m := initMaint()
	ctx := context.Background()

	_, _, _, err := m.Clock(ctx, 0, false)
	assert.NotEqual(t, nil, err)

	_, _, _, err = m.Clock(ctx, 1, true)
	assert.Equal(t, nil, err)
}

func TestSyncClock(t *testing.T) {
	m := initMaint()
	ctx := context.Background()

	syncStatus := m.syncClock(ctx)
	assert.NotEqual(t, "", syncStatus)
}

func TestDiffClock(t *testing.T) {
	m := initMaint()
	ctx := context.Background()

	_, diffDangerous := m.diffClock(ctx, 1)
	assert.Equal(t, true, diffDangerous)
}
