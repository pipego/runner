package server

import (
	"context"
	"testing"

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
	// TODO: TestLoadFile
	assert.Equal(t, nil, nil)
}
