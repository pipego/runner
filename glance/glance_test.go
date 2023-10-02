package glance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDir(t *testing.T) {
	// TODO: FIXME
	assert.Equal(t, nil, nil)
}

func TestFile(t *testing.T) {
	// TODO: FIXME
	assert.Equal(t, nil, nil)
}

func TestSys(t *testing.T) {
	// TODO: FIXME
	assert.Equal(t, nil, nil)
}

func TestHost(t *testing.T) {
	g := glance{
		cfg: DefaultConfig(),
	}

	h := g._host()
	assert.NotEqual(t, "", h)
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

	_cpu, _memory, _storage, _os := g.stats(alloc, request)
	assert.NotEqual(t, nil, _cpu)
	assert.NotEqual(t, nil, _memory)
	assert.NotEqual(t, nil, _storage)
	assert.NotEqual(t, "", _os)
}
