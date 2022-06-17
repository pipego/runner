package livelog

import (
	"context"
	"sync"
)

const (
	// This is the amount of items that are stored in memory
	// in the buffer. This should result in approximately 10kb
	// of memory allocated per-stream and per-subscriber, not
	// including any logdata stored in these structures.
	bufferSize = 5000
)

type stream struct {
	sync.Mutex
	lines []*Line
	list  map[*subscriber]struct{}
}

func newStream(_ context.Context) *stream {
	return &stream{
		list: map[*subscriber]struct{}{},
	}
}

func (s *stream) write(ctx context.Context, line *Line) error {
	s.Lock()
	defer s.Unlock()

	s.lines = append(s.lines, line)

	for l := range s.list {
		l.publish(ctx, line)
	}

	// The history should not be unbounded. The history
	// slice is capped and items are removed in a FIFO
	// ordering when capacity is reached.
	if size := len(s.lines); size >= bufferSize {
		s.lines = s.lines[size-bufferSize:]
	}

	return nil
}

func (s *stream) subscribe(ctx context.Context) (line <-chan *Line, err <-chan error) {
	e := make(chan error)

	sub := &subscriber{
		handler: make(chan *Line, bufferSize),
		closec:  make(chan struct{}),
	}

	s.Lock()

	for _, item := range s.lines {
		sub.publish(ctx, item)
	}

	s.list[sub] = struct{}{}

	s.Unlock()

	go func() {
		defer close(e)
		select {
		case <-sub.closec:
		case <-ctx.Done():
			sub.close(ctx)
		}
	}()

	return sub.handler, e
}

func (s *stream) close(ctx context.Context) error {
	s.Lock()
	defer s.Unlock()

	for sub := range s.list {
		delete(s.list, sub)
		sub.close(ctx)
	}

	return nil
}
