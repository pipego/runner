package livelog

import (
	"context"
	"sync"
)

type subscriber struct {
	sync.Mutex
	handler chan *Line
	closec  chan struct{}
	closed  bool
}

func (s *subscriber) publish(_ context.Context, line *Line) {
	select {
	case <-s.closec:
	case s.handler <- line:
	default:
		// Lines are sent on a buffered channel. If there
		// is a slow consumer that is not processing events,
		// the buffered channel will fill and newer messages
		// are ignored.
	}
}

func (s *subscriber) close(_ context.Context) {
	s.Lock()
	defer s.Unlock()

	if !s.closed {
		close(s.closec)
		s.closed = true
	}
}
