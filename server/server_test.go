package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func initZip(data []byte) ([]byte, error) {
	var b bytes.Buffer

	gz := gzip.NewWriter(&b)
	defer func(gz *gzip.Writer) {
		_ = gz.Close()
	}(gz)

	if _, err := gz.Write(data); err != nil {
		return nil, err
	}

	if err := gz.Flush(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func TestInitRunner(t *testing.T) {
	s := server{
		cfg: DefaultConfig(),
	}

	_, err := s.newRunner(context.Background())
	assert.Equal(t, nil, err)
}

func TestLoadUnzipped(t *testing.T) {
	s := server{
		cfg: DefaultConfig(),
	}

	ctx := context.Background()

	f, err := s.newFile(ctx)
	assert.Equal(t, nil, err)

	buf := []byte("#!/bin/bash\necho \"Hello World!\"")
	name, err := s.loadFile(ctx, f, buf, false)
	assert.Equal(t, nil, err)

	if _, err = os.Stat(name); errors.Is(err, os.ErrNotExist) {
		assert.Error(t, err)
	}

	_ = f.Remove(ctx, name)

	buf = []byte("echo \"Hello World!\"")
	_, err = s.loadFile(ctx, f, buf, false)
	assert.NotEqual(t, nil, err)
}

func TestLoadZipped(t *testing.T) {
	s := server{
		cfg: DefaultConfig(),
	}

	ctx := context.Background()

	f, err := s.newFile(ctx)
	assert.Equal(t, nil, err)

	buf := []byte("#!/bin/bash\necho \"Hello World!\"")
	buf, err = initZip(buf)
	assert.Equal(t, nil, err)

	name, err := s.loadFile(ctx, f, buf, true)
	assert.Equal(t, nil, err)

	if _, err = os.Stat(name); errors.Is(err, os.ErrNotExist) {
		assert.Error(t, err)
	}

	_ = f.Remove(ctx, name)

	buf = []byte("echo \"Hello World!\"")
	buf, err = initZip(buf)
	assert.Equal(t, nil, err)

	name, err = s.loadFile(ctx, f, buf, true)
	assert.NotEqual(t, nil, err)

	_ = f.Remove(ctx, name)
}
