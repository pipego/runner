package server

import (
	"context"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestInitRunner(t *testing.T) {
	s := server{
		cfg: DefaultConfig(),
	}

	err := s.initRunner(context.Background())
	assert.Equal(t, nil, err)
}

func TestLoadFile(t *testing.T) {
	s := server{
		cfg: DefaultConfig(),
	}

	ctx := context.Background()

	err := s.initFile(ctx)
	assert.Equal(t, nil, err)

	buf := []byte("#!/bin/bash\necho \"Hello World!\"")
	name, err := s.loadFile(ctx, buf, false)
	assert.Equal(t, nil, err)

	if _, err = os.Stat(name); errors.Is(err, os.ErrNotExist) {
		assert.Error(t, err)
	}
	_ = s.cfg.File.Remove(ctx, name)

	buf = []byte("echo \"Hello World!\"")
	_, err = s.loadFile(ctx, buf, false)
	assert.NotEqual(t, nil, err)
}
