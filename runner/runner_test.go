package runner

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pipego/runner/livelog"
)

var (
	log = livelog.New(context.Background(), livelog.DefaultConfig())
)

func TestRun(t *testing.T) {
	var args []string
	var err error
	var r runner

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = log.Init(ctx)
	_ = log.Create(ctx, livelog.ID)

	err = r.Run(ctx, "", args, log, cancel)
	assert.NotEqual(t, nil, err)

	args = []string{"invalid"}
	err = r.Run(ctx, "", args, log, cancel)
	assert.NotEqual(t, nil, err)

	args = []string{"echo", "task"}
	err = r.Run(ctx, "", args, log, cancel)
	assert.Equal(t, nil, err)

	lines, _ := log.Tail(ctx, livelog.ID)

L:
	for {
		select {
		case line := <-lines:
			fmt.Println("Pos:", line.Pos)
			fmt.Println("Time:", line.Time)
			fmt.Println("Message:", line.Message)
			assert.Equal(t, "task", line.Message)
		case <-ctx.Done():
			break L
		}
	}

	_ = log.Delete(ctx, livelog.ID)
}
