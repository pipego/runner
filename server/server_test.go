package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitBuilder(t *testing.T) {
	s := server{
		cfg: DefaultConfig(),
	}

	err := s.initBuilder(context.Background())
	assert.Equal(t, nil, err)
}

func TestInitRunner(t *testing.T) {
	s := server{
		cfg: DefaultConfig(),
	}

	err := s.initRunner(context.Background())
	assert.Equal(t, nil, err)
}
