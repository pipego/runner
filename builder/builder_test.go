package builder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	c = Config{
		ApiVersion: "v1",
		Kind:       "runner",
		MetaData: MetaData{
			Name: "runner",
		},
		Spec: Spec{
			Tasks: nil,
		},
	}

	b = builder{}
)

func TestZero(t *testing.T) {
	_, err := b.Run(context.Background(), &c)
	assert.Equal(t, nil, err)
}

func TestOne(t *testing.T) {
	c.Spec.Tasks = []Task{
		{
			Name:    "task1",
			Command: []string{"cmd1", "args1"},
			Depend:  []string{},
		},
	}

	_, err := b.Run(context.Background(), &c)
	assert.Equal(t, nil, err)
}

func TestManyNoDeps(t *testing.T) {
	c.Spec.Tasks = []Task{
		{
			Name:    "task1",
			Command: []string{"cmd1", "args1"},
			Depend:  []string{},
		},
		{
			Name:    "task2",
			Command: []string{"cmd2", "args2"},
			Depend:  []string{},
		},
	}

	_, err := b.Run(context.Background(), &c)
	assert.Equal(t, nil, err)
}

func TestManyWithDepsSuccess(t *testing.T) {
	c.Spec.Tasks = []Task{
		{
			Name:    "task1",
			Command: []string{"cmd1", "args1"},
			Depend:  []string{},
		},
		{
			Name:    "task2",
			Command: []string{"cmd2", "args2"},
			Depend:  []string{},
		},
		{
			Name:    "task3",
			Command: []string{"cmd3", "args3"},
			Depend:  []string{"task1", "task2"},
		},
	}

	_, err := b.Run(context.Background(), &c)
	assert.Equal(t, nil, err)
}
