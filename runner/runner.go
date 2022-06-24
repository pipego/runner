// Package runner implements a directed acyclic graph task runner with deterministic teardown.
// it is similar to package errgroup, in that it runs multiple tasks in parallel and returns
// the first error it encounters. Users define a Runner as a set vertices (functions) and edges
// between them. During Run, the directed acyclec graph will be validated and each vertex
// will run in parallel as soon as it's dependencies have been resolved. The Runner will only
// return after all running goroutines have stopped.
package runner

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"time"

	"github.com/pkg/errors"

	"github.com/pipego/runner/builder"
	"github.com/pipego/runner/config"
	"github.com/pipego/runner/livelog"
)

type Runner interface {
	Run(context.Context, *builder.Dag, livelog.Livelog) error
}

type Config struct {
	Config config.Config
}

// Runner collects functions and arranges them as vertices and edges of a directed acyclic graph.
// Upon validation of the graph, functions are run in parallel topological order. The zero value
// is useful.
type runner struct {
	cfg   *Config
	fn    map[string]function
	graph map[string][]string
}

type function struct {
	args []string
	name func(context.Context, []string, livelog.Livelog) error
}

type result struct {
	err  error
	name string
}

var errMissingVertex = errors.New("missing vertex")
var errCycleDetected = errors.New("dependency cycle detected")

func New(_ context.Context, cfg *Config) Runner {
	return &runner{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (r *runner) Run(ctx context.Context, dag *builder.Dag, log livelog.Livelog) error {
	for _, vertex := range dag.Vertex {
		r.AddVertex(ctx, vertex.Name, r.routine, vertex.Run)
	}

	for _, edge := range dag.Edge {
		r.AddEdge(ctx, edge.From, edge.To)
	}

	return r.runDag(ctx, log)
}

// AddVertex adds a function as a vertex in the graph. Only functions which have been added in this
// way will be executed during Run.
func (r *runner) AddVertex(_ context.Context, name string,
	fn func(context.Context, []string, livelog.Livelog) error, args []string) {
	if r.fn == nil {
		r.fn = make(map[string]function)
	}

	r.fn[name] = function{
		args: args,
		name: fn,
	}
}

// AddEdge establishes a dependency between two vertices in the graph. Both from and to must exist
// in the graph, or Run will err. The vertex at from will execute before the vertex at to.
func (r *runner) AddEdge(_ context.Context, from, to string) {
	if r.graph == nil {
		r.graph = make(map[string][]string)
	}

	r.graph[from] = append(r.graph[from], to)
}

func (r *runner) routine(ctx context.Context, args []string, log livelog.Livelog) error {
	var a []string
	var n string

	if len(args) > 1 {
		n, _ = exec.LookPath(args[0])
		a = args[1:]
	} else if len(args) == 1 {
		n, _ = exec.LookPath(args[0])
	} else {
		return errors.New("invalid args")
	}

	cmd := exec.Command(n, a...)

	// TODO: cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()

	go func(c *exec.Cmd, o io.ReadCloser) {
		_ = c.Start()
		scanner := bufio.NewScanner(o)
		pos := 1
		for scanner.Scan() {
			if err := log.Write(ctx, livelog.ID, &livelog.Line{Pos: int64(pos), Time: time.Now().Unix(), Message: scanner.Text()}); err != nil {
				// TODO: error
			}
			pos += 1
		}
		if scanner.Err() != nil {
			// TODO: error
		}
		_ = c.Wait()
	}(cmd, stdout)

	return nil
}

// Run will validate that all edges in the graph point to existing vertices, and that there are
// no dependency cycles. After validation, each vertex will be run, deterministically, in parallel
// topological order. If any vertex returns an error, no more vertices will be scheduled and
// Run will exit and return that error once all in-flight functions finish execution.
func (r *runner) runDag(ctx context.Context, log livelog.Livelog) error {
	var err error

	// sanity check
	if len(r.fn) == 0 {
		return nil
	}

	// count how many deps each vertex has
	deps := make(map[string]int)
	for vertex, edges := range r.graph {
		// every vertex along every edge must have an associated fn
		if _, ok := r.fn[vertex]; !ok {
			return errMissingVertex
		}
		for _, vertex := range edges {
			if _, ok := r.fn[vertex]; !ok {
				return errMissingVertex
			}
			deps[vertex]++
		}
	}

	if r.detectCycles(ctx) {
		return errCycleDetected
	}

	resc := make(chan result, len(r.fn))
	running := 0

	// start any vertex that has no deps
	for name := range r.fn {
		if deps[name] == 0 {
			running++
			r.start(ctx, name, r.fn[name], resc, log)
		}
	}

	// wait for all running work to complete
	for running > 0 {
		res := <-resc
		running--

		// capture the first error
		if res.err != nil && err == nil {
			err = res.err
		}

		// don't enqueue any more work on if there's been an error
		if err != nil {
			continue
		}

		// start any vertex whose deps are fully resolved
		for _, vertex := range r.graph[res.name] {
			deps[vertex]--
			if deps[vertex] == 0 {
				running++
				r.start(ctx, vertex, r.fn[vertex], resc, log)
			}
		}
	}

	return err
}

func (r *runner) detectCycles(ctx context.Context) bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for vertex := range r.graph {
		if !visited[vertex] {
			if r.detectCyclesHelper(ctx, vertex, visited, recStack) {
				return true
			}
		}
	}

	return false
}

func (r *runner) detectCyclesHelper(ctx context.Context, vertex string, visited, recStack map[string]bool) bool {
	visited[vertex] = true
	recStack[vertex] = true

	for _, v := range r.graph[vertex] {
		// only check cycles on a vertex one time
		if !visited[v] {
			if r.detectCyclesHelper(ctx, v, visited, recStack) {
				return true
			}
			// if we've visited this vertex in this recursion stack, then we have a cycle
		} else if recStack[v] {
			return true
		}
	}

	recStack[vertex] = false

	return false
}

func (r *runner) start(ctx context.Context, name string, fn function, resc chan<- result, log livelog.Livelog) {
	go func() {
		resc <- result{
			name: name,
			err:  fn.name(ctx, fn.args, log),
		}
	}()
}
