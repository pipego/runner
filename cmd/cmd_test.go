package cmd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitLogger(t *testing.T) {
	ctx := context.Background()

	logger, err := initLogger(ctx, "DEBUG")
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, logger)
}

func TestInitConfig(t *testing.T) {
	var err error
	assert.Equal(t, nil, err)
}

func TestInitServer(t *testing.T) {
	logger, _ := initLogger(context.Background(), "WARN")

	c, err := initConfig(context.Background(), logger)
	assert.Equal(t, nil, err)

	_, err = initServer(context.Background(), logger, c)
	assert.Equal(t, nil, err)
}
