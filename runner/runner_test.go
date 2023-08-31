package runner

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestRun(t *testing.T) {
	var args []string
	var envs []string
	var err error
	var r runner

	defer goleak.VerifyNone(t)

	ctx := context.Background()

	err = r.Init(ctx, Log)
	assert.Equal(t, nil, err)

	err = r.Run(ctx, "", envs, args)
	assert.NotEqual(t, nil, err)

	args = []string{"invalid"}
	err = r.Run(ctx, "", envs, args)
	assert.NotEqual(t, nil, err)

	args = []string{"echo", "task"}
	err = r.Run(ctx, "", envs, args)
	assert.Equal(t, nil, err)

	envs = []string{"ENV1=task1", "ENV2=task2"}
	args = []string{"echo", "$ENV1", "$ENV2"}
	err = r.Run(ctx, "", envs, args)
	assert.Equal(t, nil, err)

	log := r.Tail(ctx)

L:
	for {
		select {
		case line := <-log.Line:
			fmt.Println("Pos:", line.Pos)
			fmt.Println("Time:", line.Time)
			fmt.Println("Message:", line.Message)
			if line.Message == "EOF" {
				break L
			}
		}
	}

	err = r.Deinit(ctx)
	assert.Equal(t, nil, err)
}
