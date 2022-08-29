package file

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/pipego/runner/config"
)

const (
	Perm = 0755
)

const (
	Bash = iota
	Invalid
)

var (
	Shebang = []string{
		"#!/bin/bash",
		"#!/usr/bin/env bash",
	}
)

type File interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Unzip(context.Context, []byte) ([]byte, error)
	Write(context.Context, string, []byte) error
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

func (f *file) Unzip(_ context.Context, data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "failed to new")
	}

	defer func(r *gzip.Reader) {
		_ = r.Close()
	}(r)

	buf, err := io.ReadAll(r)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, errors.Wrap(err, "failed to read")
	}

	return buf, nil
}

func (f *file) Write(_ context.Context, name string, data []byte) error {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, Perm)
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

func (f *file) Remove(_ context.Context, name string) error {
	return os.Remove(name)
}

func (f *file) Type(_ context.Context, name string) int {
	file, err := os.Open(name)
	defer file.Close()

	if err != nil {
		return Invalid
	}

	reader := bufio.NewReader(file)

	buf, _, err := reader.ReadLine()
	if err != nil {
		return Invalid
	}

	var buffer bytes.Buffer
	buffer.Write(buf)
	line := buffer.String()

	ret := Invalid

	for _, item := range Shebang {
		if strings.Contains(line, item) {
			ret = Bash
			break
		}
	}

	return ret
}
