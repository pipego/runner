package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	pb "github.com/pipego/runner/server/proto"
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

func TestInitTask(t *testing.T) {
	s := server{
		cfg: DefaultConfig(),
	}

	_, err := s.newTask(context.Background())
	assert.Equal(t, nil, err)
}

func TestInitGlance(t *testing.T) {
	s := server{
		cfg: DefaultConfig(),
	}

	_, err := s.newGlance(context.Background())
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

func TestBuildEnv(t *testing.T) {
	var params []*pb.TaskParam

	s := server{
		cfg: DefaultConfig(),
	}

	ctx := context.Background()

	buf := s.buildEnv(ctx, params)
	assert.Equal(t, 0, len(buf))

	params = append(params,
		&pb.TaskParam{
			Name:  "name1",
			Value: "value1",
		},
		&pb.TaskParam{
			Name:  "name2",
			Value: "$name1",
		},
		&pb.TaskParam{
			Name:  "name3",
			Value: "$name2",
		},
		&pb.TaskParam{
			Name:  "name4",
			Value: "$$name1",
		},
		&pb.TaskParam{
			Name:  "name5",
			Value: "#name1",
		},
	)

	buf = s.buildEnv(ctx, params)
	assert.NotEqual(t, 0, len(buf))
	assert.Equal(t, "name1=value1", buf[0])
	assert.Equal(t, "name2=$name1", buf[1])
	assert.Equal(t, "name3=$name2", buf[2])
	assert.Equal(t, "name4=$$name1", buf[3])
	assert.Equal(t, "name5=#name1", buf[4])
}
