package cmd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitConfig(t *testing.T) {
	var err error
	assert.Equal(t, nil, err)
}

func TestInitServer(t *testing.T) {
	c, err := initConfig(context.Background())
	assert.Equal(t, nil, err)

	_, err = initServer(context.Background(), c)
	assert.Equal(t, nil, err)
}
