//go:build all_test

package task

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/docker/docker/client"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/pipego/runner/config"
)

func initTask() *task {
	t := task{
		cfg: DefaultConfig(),
		log: Log{},
	}

	t.cfg.Config = config.Config{}
	t.cfg.Logger = hclog.New(&hclog.LoggerOptions{
		Name:  "task",
		Level: hclog.LevelFromString("DEBUG"),
	})

	return &t
}

func TestRunEcho(t *testing.T) {
	var args []string
	var envs []string
	var lang Language
	var err error

	defer goleak.VerifyNone(t)

	_t := initTask()
	ctx := context.Background()

	err = _t.Init(ctx, lineWidth)
	assert.Equal(t, nil, err)

	err = _t.Run(ctx, "", envs, args, lang)
	assert.NotEqual(t, nil, err)

	args = []string{"invalid"}
	err = _t.Run(ctx, "", envs, args, lang)
	assert.NotEqual(t, nil, err)

	envs = []string{"ENV1=task1", "ENV2=task2"}
	args = []string{"bash", "-c", "echo $ENV1 $ENV2"}
	err = _t.Run(ctx, "", envs, args, lang)
	assert.Equal(t, nil, err)

	log := _t.Tail(ctx)

L:
	for {
		select {
		case line := <-log.Line.Out:
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
	var lang Language
	var err error

	defer goleak.VerifyNone(t)

	_t := initTask()
	ctx := context.Background()

	err = _t.Init(ctx, lineWidth)
	assert.Equal(t, nil, err)

	err = _t.Run(ctx, "", envs, args, lang)
	assert.NotEqual(t, nil, err)

	args = []string{"bash", "-c", "../test/bash.sh"}
	err = _t.Run(ctx, "", envs, args, lang)
	assert.Equal(t, nil, err)

	log := _t.Tail(ctx)

L:
	for {
		select {
		case line := <-log.Line.Out:
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
	var lang Language
	var err error

	defer goleak.VerifyNone(t)

	_t := initTask()
	ctx := context.Background()

	err = _t.Init(ctx, lineWidth)
	assert.Equal(t, nil, err)

	err = _t.Run(ctx, "", envs, args, lang)
	assert.NotEqual(t, nil, err)

	args = []string{"bash", "-c", "python3 ../test/python.py"}
	err = _t.Run(ctx, "", envs, args, lang)
	assert.Equal(t, nil, err)

	log := _t.Tail(ctx)

L:
	for {
		select {
		case line := <-log.Line.Out:
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
	var lang Language
	var err error

	defer goleak.VerifyNone(t)

	_t := initTask()
	ctx := context.Background()

	err = _t.Init(ctx, lineWidth)
	assert.Equal(t, nil, err)

	err = _t.Run(ctx, "", envs, args, lang)
	assert.NotEqual(t, nil, err)

	args = []string{"bash", "-c", "python3 ../test/split.py"}
	err = _t.Run(ctx, "", envs, args, lang)
	assert.Equal(t, nil, err)

	log := _t.Tail(ctx)

L:
	for {
		select {
		case line := <-log.Line.Out:
			if line.Message == tagEOF {
				break L
			}
			if strings.HasSuffix(line.Message, tagBOL) {
				assert.Equal(t, lineWidth, utf8.RuneCountInString(line.Message))
			}
		}
	}

	err = _t.Deinit(ctx)
	assert.Equal(t, nil, err)
}

func TestRunError(t *testing.T) {
	var args []string
	var envs []string
	var lang Language
	var err error

	defer goleak.VerifyNone(t)

	_t := initTask()
	ctx := context.Background()

	err = _t.Init(ctx, lineWidth)
	assert.Equal(t, nil, err)

	err = _t.Run(ctx, "", envs, args, lang)
	assert.NotEqual(t, nil, err)

	args = []string{"bash", "-c", "../test/error.sh"}
	err = _t.Run(ctx, "", envs, args, lang)
	assert.Equal(t, nil, err)

	log := _t.Tail(ctx)

L:
	for {
		select {
		case line := <-log.Line.Out:
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

func TestRunLanguage(t *testing.T) {
	defer goleak.VerifyNone(t)

	_t := initTask()
	_t._client, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	defer func(_client *client.Client) {
		_ = _client.Close()
	}(_t._client)

	ctx := context.Background()

	name := languages["groovy"]
	cmd := []string{"/workspace/jenkinsfile"}
	source, _ := filepath.Abs("../test")
	target := "/workspace"

	err := _t.pullImage(ctx, name, "", "")
	assert.Equal(t, nil, err)

	id, out, err := _t.runContainer(ctx, name, cmd, source, target)
	assert.Equal(t, nil, err)

	fmt.Println(string(out))

	err = _t.removeContainer(ctx, id)
	assert.Equal(t, nil, err)

	err = _t.removeImage(ctx, name)
	assert.Equal(t, nil, err)
}
