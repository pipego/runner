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
			Name:     "task1",
			Commands: []string{"cmd1", "args1"},
			Depends:  []string{},
		},
	}

	_, err := b.Run(context.Background(), &c)
	assert.Equal(t, nil, err)
}

func TestManyNoDeps(t *testing.T) {
	c.Spec.Tasks = []Task{
		{
			Name:     "task1",
			Commands: []string{"cmd1", "args1"},
			Depends:  []string{},
		},
		{
			Name:     "task2",
			Commands: []string{"cmd2", "args2"},
			Depends:  []string{},
		},
	}

	_, err := b.Run(context.Background(), &c)
	assert.Equal(t, nil, err)
}

func TestManyWithDepsSuccess(t *testing.T) {
	c.Spec.Tasks = []Task{
		{
			Name:     "task1",
			Commands: []string{"cmd1", "args1"},
			Depends:  []string{},
		},
		{
			Name:     "task2",
			Commands: []string{"cmd2", "args2"},
			Depends:  []string{},
		},
		{
			Name:     "task3",
			Commands: []string{"cmd3", "args3"},
			Depends:  []string{"task1", "task2"},
		},
	}

	_, err := b.Run(context.Background(), &c)
	assert.Equal(t, nil, err)
}
