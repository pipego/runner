package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	c = Config{
		Kind: "kind",
		Type: "type",
		Name: "name",
		Task: nil,
	}

	b = builder{}
)

func TestZero(t *testing.T) {
	_, err := b.Run(&c)
	assert.Equal(t, nil, err)
}

func TestOne(t *testing.T) {
	c.Task = []Task{
		{
			Name:    "task1",
			Command: "run1",
			Depend:  []string{},
		},
	}

	_, err := b.Run(&c)
	assert.Equal(t, nil, err)
}

func TestManyNoDeps(t *testing.T) {
	c.Task = []Task{
		{
			Name:    "task1",
			Command: "run1",
			Depend:  []string{},
		},
		{
			Name:    "task2",
			Command: "run2",
			Depend:  []string{},
		},
	}

	_, err := b.Run(&c)
	assert.Equal(t, nil, err)
}

func TestManyWithDepsSuccess(t *testing.T) {
	c.Task = []Task{
		{
			Name:    "task1",
			Command: "run1",
			Depend:  []string{},
		},
		{
			Name:    "task2",
			Command: "run2",
			Depend:  []string{},
		},
		{
			Name:    "task3",
			Command: "run3",
			Depend:  []string{"task1", "task2"},
		},
	}

	_, err := b.Run(&c)
	assert.Equal(t, nil, err)
}
