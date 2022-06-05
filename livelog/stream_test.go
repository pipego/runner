package livelog

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStreamWrite(t *testing.T) {
	s := newStream(context.Background())

	err := s.write(context.Background(), &Line{})
	assert.Equal(t, nil, err)
}

func TestStreamSubscribe(t *testing.T) {
	s := newStream(context.Background())

	err := s.write(context.Background(), &Line{Pos: 1, Time: 2022, Message: "message"})
	assert.Equal(t, nil, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()

	lineChan, errChan := s.subscribe(ctx)
	line := <-lineChan
	err = <-errChan
	assert.Equal(t, int64(1), line.Pos)
	assert.Equal(t, int64(2022), line.Time)
	assert.Equal(t, "message", line.Message)
	assert.Equal(t, nil, err)
}

func TestStreamClose(t *testing.T) {
	s := newStream(context.Background())

	err := s.close(context.Background())
	assert.Equal(t, nil, err)
}
