package file

import (
	"archive/zip"
	"context"
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/pipego/runner/config"
)

const (
	LEN  = 1024
	PERM = 0755
)

const (
	Bash int = 0
)

type File interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Write(context.Context, string, []byte) error
	Unzip(context.Context, string) error
	Remove(context.Context, string) error
	Type(context.Context, string) int
}

type Config struct {
	Config config.Config
}

type file struct {
	cfg *Config
}

func New(_ context.Context, cfg *Config) File {
	return &file{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (f *file) Init(_ context.Context) error {
	return nil
}

func (f *file) Deinit(_ context.Context) error {
	return nil
}

func (f *file) Write(_ context.Context, name string, data []byte) error {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, PERM)
	if err != nil {
		return errors.Wrap(err, "failed to open")
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	if _, err := file.Write(data); err != nil {
		return errors.Wrap(err, "failed to write")
	}

	return nil
}

func (f *file) Unzip(_ context.Context, name string) error {
	r, err := zip.OpenReader(name)
	if err != nil {
		return errors.Wrap(err, "failed to read")
	}

	defer func(r *zip.ReadCloser) {
		_ = r.Close()
	}(r)

	if len(r.File) != 1 {
		return errors.Wrap(err, "multiple files not supported")
	}

	src := r.File[0]
	if src.FileInfo().IsDir() {
		return errors.Wrap(err, "directory not supported")
	}

	dst, err := os.OpenFile(name+".tmp", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, PERM)
	if err != nil {
		return errors.Wrap(err, "failed to open")
	}

	defer func(dst *os.File) {
		_ = dst.Close()
	}(dst)

	buf, err := src.Open()
	if err != nil {
		return errors.Wrap(err, "failed to open")
	}

	defer func(buf io.ReadCloser) {
		_ = buf.Close()
	}(buf)

	for {
		if _, err = io.CopyN(dst, buf, LEN); err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "failed to copy")
		}
	}

	if err := os.Remove(name); err != nil {
		return errors.Wrap(err, "failed to remove")
	}

	if err := os.Rename(name+".tmp", name); err != nil {
		return errors.Wrap(err, "failed to rename")
	}

	return nil
}

func (f *file) Remove(_ context.Context, name string) error {
	return os.Remove(name)
}

func (f *file) Type(_ context.Context, name string) int {
	// TODO: Type
	return Bash
}
