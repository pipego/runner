package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitConfig(t *testing.T) {
	var err error
	assert.Equal(t, nil, err)
}

func TestInitRunner(t *testing.T) {
	c, err := initConfig()
	assert.Equal(t, nil, err)

	_, err = initRunner(c)
	assert.Equal(t, nil, err)
}

func TestInitServer(t *testing.T) {
	c, err := initConfig()
	assert.Equal(t, nil, err)

	_, err = initServer(c)
	assert.Equal(t, nil, err)
}
