package task

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
	var _t task

	defer goleak.VerifyNone(t)

	ctx := context.Background()

	err = _t.Init(ctx, Log)
	assert.Equal(t, nil, err)

	err = _t.Run(ctx, "", envs, args)
	assert.NotEqual(t, nil, err)

	args = []string{"invalid"}
	err = _t.Run(ctx, "", envs, args)
	assert.NotEqual(t, nil, err)

	envs = []string{"ENV1=task1", "ENV2=task2"}
	args = []string{"bash", "-c", "echo $ENV1 $ENV2"}
	err = _t.Run(ctx, "", envs, args)
	assert.Equal(t, nil, err)

	log := _t.Tail(ctx)

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

	err = _t.Deinit(ctx)
	assert.Equal(t, nil, err)
}
