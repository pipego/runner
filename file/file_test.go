package file

import (
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	FILE_INVALID = "file-invalid"
	FILE_TEST    = "file-test"
)

func initZip() ([]byte, error) {
	var b bytes.Buffer

	gz := gzip.NewWriter(&b)
	defer func(gz *gzip.Writer) {
		_ = gz.Close()
	}(gz)

	str := "#!/bin/bash\necho \"Hello World!\""
	if _, err := gz.Write([]byte(str)); err != nil {
		return nil, err
	}

	if err := gz.Flush(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func TestUnzip(t *testing.T) {
	f := file{
		cfg: DefaultConfig(),
	}

	ctx := context.Background()

	buf, err := initZip()
	assert.Equal(t, nil, err)

	_, err = f.Unzip(ctx, buf)
	assert.Equal(t, nil, err)

	buf = []byte{}
	_, err = f.Unzip(ctx, buf)
	assert.NotEqual(t, nil, err)
}

func TestWrite(t *testing.T) {
	f := file{
		cfg: DefaultConfig(),
	}

	ctx := context.Background()

	buf := []byte("#!/bin/bash\necho \"Hello World!\"")
	err := f.Write(ctx, FILE_TEST, buf)
	assert.Equal(t, nil, err)

	if _, err = os.Stat(FILE_TEST); errors.Is(err, os.ErrNotExist) {
		assert.Error(t, err)
	}

	_ = f.Remove(ctx, FILE_TEST)
}

func TestRemove(t *testing.T) {
	f := file{
		cfg: DefaultConfig(),
	}

	ctx := context.Background()

	buf := []byte("#!/bin/bash\necho \"Hello World!\"")
	_ = f.Write(ctx, FILE_TEST, buf)
	err := f.Remove(ctx, FILE_TEST)
	assert.Equal(t, nil, err)

	err = f.Remove(ctx, FILE_INVALID)
	assert.NotEqual(t, nil, err)
}

func TestType(t *testing.T) {
	f := file{
		cfg: DefaultConfig(),
	}

	ctx := context.Background()

	buf := []byte("#!/bin/bash\necho \"Hello World!\"")
	_ = f.Write(ctx, FILE_TEST, buf)
	ret := f.Type(ctx, FILE_TEST)
	assert.Equal(t, Bash, ret)
	_ = f.Remove(ctx, FILE_TEST)

	ret = f.Type(ctx, FILE_INVALID)
	assert.Equal(t, Invalid, ret)

	buf = []byte("")
	_ = f.Write(ctx, FILE_TEST, buf)
	ret = f.Type(ctx, FILE_TEST)
	assert.Equal(t, Invalid, ret)
	_ = f.Remove(ctx, FILE_TEST)

	buf = []byte("#!/bin/sh")
	_ = f.Write(ctx, FILE_TEST, buf)
	ret = f.Type(ctx, FILE_TEST)
	assert.Equal(t, Invalid, ret)
	_ = f.Remove(ctx, FILE_TEST)
}
