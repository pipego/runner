package livelog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscriberPublish(t *testing.T) {
	s := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
		closed:  false,
	}

	s.publish(context.Background(), &Line{Pos: 1, Time: 2022, Message: "message"})
	h := <-s.handler
	assert.Equal(t, int64(1), h.Pos)
	assert.Equal(t, int64(2022), h.Time)
	assert.Equal(t, "message", h.Message)
}

func TestSubscriberClose(t *testing.T) {
	s := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
		closed:  false,
	}

	s.close(context.Background())
	assert.Equal(t, true, s.closed)
}
