package livelog

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func initLivelog() *livelog {
	l := &livelog{
		cfg: DefaultConfig(),
	}

	_ = l.Init(context.Background())

	return l
}

func TestLivelogCreate(t *testing.T) {
	l := initLivelog()

	err := l.Create(context.Background(), 0)
	assert.Equal(t, nil, err)
}

func TestLivelogDelete(t *testing.T) {
	l := initLivelog()

	err := l.Create(context.Background(), 0)
	assert.Equal(t, nil, err)

	err = l.Delete(context.Background(), 1)
	assert.NotEqual(t, nil, err)

	err = l.Delete(context.Background(), 0)
	assert.Equal(t, nil, err)
}

func TestLivelogWrite(t *testing.T) {
	l := initLivelog()

	err := l.Create(context.Background(), 0)
	assert.Equal(t, nil, err)

	err = l.Write(context.Background(), 1, &Line{})
	assert.NotEqual(t, nil, err)

	err = l.Write(context.Background(), 0, &Line{})
	assert.Equal(t, nil, err)
}

func TestLivelogTail(t *testing.T) {
	l := initLivelog()

	err := l.Create(context.Background(), 0)
	assert.Equal(t, nil, err)

	err = l.Write(context.Background(), 0, &Line{Pos: 1, Time: 2022, Message: "message"})
	assert.Equal(t, nil, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()

	lineChan, errChan := l.Tail(ctx, 0)
	line := <-lineChan
	err = <-errChan
	assert.Equal(t, int64(1), line.Pos)
	assert.Equal(t, int64(2022), line.Time)
	assert.Equal(t, "message", line.Message)
	assert.Equal(t, nil, err)
}
