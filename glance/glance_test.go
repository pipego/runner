package glance

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	invalidFile = "/path/to/invalid"
	validFile   = "/etc/hostname"

	maxSize = 1000
)

func TestDir(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	ctx := context.Background()

	_, err := g.Dir(ctx, "/path/to/invalid")
	assert.NotEqual(t, nil, err)

	entries, err := g.Dir(ctx, Root)
	assert.Equal(t, nil, err)
	assert.Less(t, 2, len(entries))
}

func TestFile(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	ctx := context.Background()

	_, readable, err := g.File(ctx, invalidFile, maxSize)
	assert.Equal(t, false, readable)
	assert.NotEqual(t, nil, err)

	content, readable, err := g.File(ctx, validFile, maxSize)
	assert.NotEqual(t, "", content)
	assert.Equal(t, true, readable)
	assert.Equal(t, nil, err)
}

func TestSys(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	ctx := context.Background()

	alloc, request, _cpu, _memory, _storage, _host, _os, _ := g.Sys(ctx)
	assert.NotEqual(t, -1, alloc.MilliCPU)
	assert.NotEqual(t, -1, alloc.Memory)
	assert.NotEqual(t, -1, alloc.Storage)
	assert.NotEqual(t, -1, request.MilliCPU)
	assert.NotEqual(t, -1, request.Memory)
	assert.NotEqual(t, -1, request.Storage)
	assert.NotEqual(t, nil, _cpu)
	assert.NotEqual(t, nil, _memory)
	assert.NotEqual(t, nil, _storage)
	assert.NotEqual(t, "", _host)
	assert.NotEqual(t, "", _os)
}

func TestEntry(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	d, _ := os.Getwd()
	buf, _ := os.ReadDir(d)

	for _, item := range buf {
		e, _ := g.entry(d, item.Name())
		assert.NotEqual(t, "", e.Name)
		assert.NotEqual(t, "", e.Time)
		assert.NotEqual(t, "", e.Mode)
	}
}

func TestIsText(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	s := g.isText(invalidFile)
	assert.Equal(t, false, s)

	s = g.isText(validFile)
	assert.Equal(t, true, s)
}

func TestValidSize(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	s := g.validSize(invalidFile, maxSize)
	assert.Equal(t, false, s)

	s = g.validSize(validFile, 0)
	assert.Equal(t, false, s)

	s = g.validSize(validFile, maxSize)
	assert.Equal(t, true, s)
}

func TestReadFile(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	_, err := g.readFile(invalidFile)
	assert.NotEqual(t, nil, err)

	b, err := g.readFile(validFile)
	assert.Equal(t, nil, err)
	assert.NotEqual(t, "", b)
}

func TestMilliCPU(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	alloc, request := g.milliCPU()
	assert.NotEqual(t, -1, alloc)
	assert.NotEqual(t, -1, request)
}

func TestMemory(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	alloc, request := g.memory()
	assert.NotEqual(t, -1, alloc)
	assert.NotEqual(t, -1, request)
}

func TestStorage(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	alloc, request := g.storage()
	assert.NotEqual(t, -1, alloc)
	assert.NotEqual(t, -1, request)
}

func TestStats(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	alloc := Resource{}
	request := Resource{}

	alloc.MilliCPU, request.MilliCPU = g.milliCPU()
	alloc.Memory, request.Memory = g.memory()
	alloc.Storage, request.Storage = g.storage()

	_cpu, _memory, _storage := g.stats(alloc, request)
	assert.NotEqual(t, nil, _cpu)
	assert.NotEqual(t, nil, _memory)
	assert.NotEqual(t, nil, _storage)
}

func TestHost(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	h := g._host()
	assert.NotEqual(t, "", h)
}

func TestOs(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	o := g._os()
	assert.NotEqual(t, "", o)
}
