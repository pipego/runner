package task

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestRunEcho(t *testing.T) {
	var args []string
	var envs []string
	var err error
	var _t task

	defer goleak.VerifyNone(t)

	ctx := context.Background()

	err = _t.Init(ctx, logLen)
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
			if line.Message == tagEOF {
				break L
			}
		}
	}

	err = _t.Deinit(ctx)
	assert.Equal(t, nil, err)
}

func TestRunBash(t *testing.T) {
	var args []string
	var envs []string
	var err error
	var _t task

	defer goleak.VerifyNone(t)

	ctx := context.Background()

	err = _t.Init(ctx, logLen)
	assert.Equal(t, nil, err)

	err = _t.Run(ctx, "", envs, args)
	assert.NotEqual(t, nil, err)

	args = []string{"bash", "-c", "../test/bash.sh"}
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
			if line.Message == tagEOF {
				break L
			}
		}
	}

	err = _t.Deinit(ctx)
	assert.Equal(t, nil, err)
}

func TestRunPython(t *testing.T) {
	var args []string
	var envs []string
	var err error
	var _t task

	defer goleak.VerifyNone(t)

	ctx := context.Background()

	err = _t.Init(ctx, logLen)
	assert.Equal(t, nil, err)

	err = _t.Run(ctx, "", envs, args)
	assert.NotEqual(t, nil, err)

	args = []string{"bash", "-c", "python3 ../test/python.py"}
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
			if line.Message == tagEOF {
				break L
			}
		}
	}

	err = _t.Deinit(ctx)
	assert.Equal(t, nil, err)
}

func TestRunSplit(t *testing.T) {
	var args []string
	var envs []string
	var err error
	var _t task

	defer goleak.VerifyNone(t)

	ctx := context.Background()

	err = _t.Init(ctx, logLen)
	assert.Equal(t, nil, err)

	err = _t.Run(ctx, "", envs, args)
	assert.NotEqual(t, nil, err)

	args = []string{"bash", "-c", "python3 ../test/split.py"}
	err = _t.Run(ctx, "", envs, args)
	assert.Equal(t, nil, err)

	log := _t.Tail(ctx)

L:
	for {
		select {
		case line := <-log.Line:
			if line.Message == tagEOF {
				break L
			}
			if strings.HasSuffix(line.Message, tagBOL) {
				assert.Equal(t, splitLen, len(line.Message))
			}
		}
	}

	err = _t.Deinit(ctx)
	assert.Equal(t, nil, err)
}
