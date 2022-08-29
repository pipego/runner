package runner

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	var args []string
	var err error
	var r runner

	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = r.Init(ctx, Log)
	assert.Equal(t, nil, err)

	err = r.Run(ctx, "", args, cancel)
	assert.NotEqual(t, nil, err)

	args = []string{"invalid"}
	err = r.Run(ctx, "", args, cancel)
	assert.NotEqual(t, nil, err)

	args = []string{"echo", "task"}
	err = r.Run(ctx, "", args, cancel)
	assert.Equal(t, nil, err)

	log := r.Tail(ctx)

L:
	for {
		select {
		case line := <-log.Line:
			fmt.Println("Pos:", line.Pos)
			fmt.Println("Time:", line.Time)
			fmt.Println("Message:", line.Message)
			assert.Equal(t, "task", line.Message)
		case <-ctx.Done():
			break L
		}
	}

	err = r.Deinit(ctx)
	assert.Equal(t, nil, err)
}
