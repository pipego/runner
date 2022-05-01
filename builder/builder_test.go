package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	c = Config{
		Kind:  "kind",
		Type:  "type",
		Name:  "name",
		Tasks: nil,
	}

	b = builder{}
)

func TestZero(t *testing.T) {
	_, err := b.Run(&c)
	assert.Equal(t, nil, err)
}

func TestOne(t *testing.T) {
	c.Tasks = []Task{
		{
			Name:     "task1",
			Commands: []string{"run1"},
			Depends:  []string{},
		},
	}

	_, err := b.Run(&c)
	assert.Equal(t, nil, err)
}

func TestManyNoDeps(t *testing.T) {
	c.Tasks = []Task{
		{
			Name:     "task1",
			Commands: []string{"run1"},
			Depends:  []string{},
		},
		{
			Name:     "task2",
			Commands: []string{"run2"},
			Depends:  []string{},
		},
	}

	_, err := b.Run(&c)
	assert.Equal(t, nil, err)
}

func TestManyWithDepsSuccess(t *testing.T) {
	c.Tasks = []Task{
		{
			Name:     "task1",
			Commands: []string{"run1"},
			Depends:  []string{},
		},
		{
			Name:     "task2",
			Commands: []string{"run2"},
			Depends:  []string{},
		},
		{
			Name:     "task3",
			Commands: []string{"run3"},
			Depends:  []string{"task1", "task2"},
		},
	}

	_, err := b.Run(&c)
	assert.Equal(t, nil, err)
}
