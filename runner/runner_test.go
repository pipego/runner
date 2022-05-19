package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	args = []string{"echo", "hello runner"}
)

func TestRoutine(t *testing.T) {
	var r runner

	err := r.routine(context.Background(), args)
	assert.Equal(t, nil, err)
}

func TestZero(t *testing.T) {
	var r runner

	res := make(chan error)
	go func() { res <- r.runDag(context.Background()) }()

	select {
	case err := <-res:
		if err != nil {
			t.Errorf("%v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestOne(t *testing.T) {
	var r runner
	ctx := context.Background()

	err := errors.New("error")
	r.AddVertex(ctx, "one", func(context.Context, []string) error { return err }, []string{})

	res := make(chan error)
	go func() { res <- r.runDag(ctx) }()

	select {
	case err := <-res:
		if want, have := err, err; want != have {
			t.Errorf("want %v, have %v", want, have)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestManyNoDeps(t *testing.T) {
	var r runner
	ctx := context.Background()

	err := errors.New("error")
	r.AddVertex(ctx, "one", func(context.Context, []string) error { return err }, []string{})
	r.AddVertex(ctx, "two", func(context.Context, []string) error { return nil }, []string{})
	r.AddVertex(ctx, "three", func(context.Context, []string) error { return nil }, []string{})
	r.AddVertex(ctx, "fout", func(context.Context, []string) error { return nil }, []string{})

	res := make(chan error)
	go func() { res <- r.runDag(ctx) }()

	select {
	case err := <-res:
		if want, have := err, err; want != have {
			t.Errorf("want %v, have %v", want, have)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestManyWithCycle(t *testing.T) {
	var r runner
	ctx := context.Background()

	r.AddVertex(ctx, "one", func(context.Context, []string) error { return nil }, []string{})
	r.AddVertex(ctx, "two", func(context.Context, []string) error { return nil }, []string{})
	r.AddVertex(ctx, "three", func(context.Context, []string) error { return nil }, []string{})
	r.AddVertex(ctx, "four", func(context.Context, []string) error { return nil }, []string{})

	r.AddEdge(ctx, "one", "two")
	r.AddEdge(ctx, "two", "three")
	r.AddEdge(ctx, "three", "four")
	r.AddEdge(ctx, "three", "one")

	res := make(chan error)
	go func() { res <- r.runDag(ctx) }()

	select {
	case err := <-res:
		if want, have := errCycleDetected, err; want != have {
			t.Errorf("want %v, have %v", want, have)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestInvalidToVertex(t *testing.T) {
	var r runner
	ctx := context.Background()

	r.AddVertex(ctx, "one", func(context.Context, []string) error { return nil }, []string{})
	r.AddVertex(ctx, "two", func(context.Context, []string) error { return nil }, []string{})
	r.AddVertex(ctx, "three", func(context.Context, []string) error { return nil }, []string{})
	r.AddVertex(ctx, "four", func(context.Context, []string) error { return nil }, []string{})

	r.AddEdge(ctx, "one", "two")
	r.AddEdge(ctx, "two", "three")
	r.AddEdge(ctx, "three", "four")
	r.AddEdge(ctx, "three", "definitely-not-a-valid-vertex")

	res := make(chan error)
	go func() { res <- r.runDag(ctx) }()

	select {
	case err := <-res:
		if want, have := errMissingVertex, err; want != have {
			t.Errorf("want %v, have %v", want, have)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestInvalidFromVertex(t *testing.T) {
	var r runner
	ctx := context.Background()

	r.AddVertex(ctx, "one", func(context.Context, []string) error { return nil }, []string{})
	r.AddVertex(ctx, "two", func(context.Context, []string) error { return nil }, []string{})
	r.AddVertex(ctx, "three", func(context.Context, []string) error { return nil }, []string{})
	r.AddVertex(ctx, "four", func(context.Context, []string) error { return nil }, []string{})

	r.AddEdge(ctx, "one", "two")
	r.AddEdge(ctx, "two", "three")
	r.AddEdge(ctx, "three", "four")
	r.AddEdge(ctx, "definitely-not-a-valid-vertex", "three")

	res := make(chan error)
	go func() { res <- r.runDag(ctx) }()

	select {
	case err := <-res:
		if want, have := errMissingVertex, err; want != have {
			t.Errorf("want %v, have %v", want, have)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestManyWithDepsSuccess(t *testing.T) {
	var r runner
	ctx := context.Background()

	res := make(chan string, 7)
	r.AddVertex(ctx, "one", func(context.Context, []string) error {
		res <- "one"
		return nil
	}, []string{})
	r.AddVertex(ctx, "two", func(context.Context, []string) error {
		res <- "two"
		return nil
	}, []string{})
	r.AddVertex(ctx, "three", func(context.Context, []string) error {
		res <- "three"
		return nil
	}, []string{})
	r.AddVertex(ctx, "four", func(context.Context, []string) error {
		res <- "four"
		return nil
	}, []string{})
	r.AddVertex(ctx, "five", func(context.Context, []string) error {
		res <- "five"
		return nil
	}, []string{})
	r.AddVertex(ctx, "six", func(context.Context, []string) error {
		res <- "six"
		return nil
	}, []string{})
	r.AddVertex(ctx, "seven", func(context.Context, []string) error {
		res <- "seven"
		return nil
	}, []string{})

	r.AddEdge(ctx, "one", "two")
	r.AddEdge(ctx, "one", "three")
	r.AddEdge(ctx, "two", "four")
	r.AddEdge(ctx, "two", "seven")
	r.AddEdge(ctx, "five", "six")

	err := make(chan error)
	go func() { err <- r.runDag(ctx) }()

	select {
	case err := <-err:
		if want, have := error(nil), err; want != have {
			t.Errorf("want %v, have %v", want, have)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}

	results := make([]string, 7)
	timeout := time.After(100 * time.Millisecond)

	for i := range results {
		select {
		case results[i] = <-res:
		case <-timeout:
			t.Error("timeout")
		}
	}

	checkOrder("one", "two", results, t)
	checkOrder("one", "three", results, t)
	checkOrder("two", "four", results, t)
	checkOrder("two", "seven", results, t)
	checkOrder("five", "six", results, t)
}

func checkOrder(from, to string, results []string, t *testing.T) {
	var fromIndex, toIndex int

	for i := range results {
		if results[i] == from {
			fromIndex = i
		}
		if results[i] == to {
			toIndex = i
		}
	}

	if fromIndex > toIndex {
		t.Errorf("from vertex: %s came after to vertex: %s", from, to)
	}
}
