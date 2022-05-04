// Package runner implements a directed acyclic graph task runner with deterministic teardown.
// it is similar to package errgroup, in that it runs multiple tasks in parallel and returns
// the first error it encounters. Users define a Runner as a set vertices (functions) and edges
// between them. During Run, the directed acyclec graph will be validated and each vertex
// will run in parallel as soon as it's dependencies have been resolved. The Runner will only
// return after all running goroutines have stopped.
package runner

import (
	"context"
	"os"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/pipego/runner/builder"
	"github.com/pipego/runner/config"
)

type Runner interface {
	Run(dag *builder.Dag) error
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
	name func([]string) error
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

func (r *runner) Run(dag *builder.Dag) error {
	for _, vertex := range dag.Vertex {
		r.AddVertex(vertex.Name, r.routine, vertex.Run)
	}

	for _, edge := range dag.Edge {
		r.AddEdge(edge.From, edge.To)
	}

	return r.runDag()
}

// AddVertex adds a function as a vertex in the graph. Only functions which have been added in this
// way will be executed during Run.
func (r *runner) AddVertex(name string, fn func([]string) error, args []string) {
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
func (r *runner) AddEdge(from, to string) {
	if r.graph == nil {
		r.graph = make(map[string][]string)
	}

	r.graph[from] = append(r.graph[from], to)
}

func (r *runner) routine(args []string) error {
	var n string
	var a []string

	outr, outw, _ := os.Pipe()
	defer func() { _ = outr.Close() }()
	defer func() { _ = outw.Close() }()

	inr, inw, _ := os.Pipe()
	defer func() { _ = inr.Close() }()
	defer func() { _ = inw.Close() }()

	if len(args) > 1 {
		n, _ = exec.LookPath(args[0])
		a = args[1:]
	} else if len(args) == 1 {
		n, _ = exec.LookPath(args[0])
	} else if len(args) == 0 {
		return errors.New("invalid args")
	}

	_, err := os.StartProcess(n, a, &os.ProcAttr{
		Env:   os.Environ(),
		Files: []*os.File{inr, outw, outw},
	})

	if err != nil {
		return errors.Wrap(err, "failed to start")
	}

	return nil
}

// Run will validate that all edges in the graph point to existing vertices, and that there are
// no dependency cycles. After validation, each vertex will be run, deterministically, in parallel
// topological order. If any vertex returns an error, no more vertices will be scheduled and
// Run will exit and return that error once all in-flight functions finish execution.
func (r *runner) runDag() error {
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

	if r.detectCycles() {
		return errCycleDetected
	}

	running := 0
	resc := make(chan result, len(r.fn))
	var err error

	// start any vertex that has no deps
	for name := range r.fn {
		if deps[name] == 0 {
			running++
			r.start(name, r.fn[name], resc)
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
				r.start(vertex, r.fn[vertex], resc)
			}
		}
	}
	return err
}

func (r *runner) detectCycles() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for vertex := range r.graph {
		if !visited[vertex] {
			if r.detectCyclesHelper(vertex, visited, recStack) {
				return true
			}
		}
	}

	return false
}

func (r *runner) detectCyclesHelper(vertex string, visited, recStack map[string]bool) bool {
	visited[vertex] = true
	recStack[vertex] = true

	for _, v := range r.graph[vertex] {
		// only check cycles on a vertex one time
		if !visited[v] {
			if r.detectCyclesHelper(v, visited, recStack) {
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

func (r *runner) start(name string, fn function, resc chan<- result) {
	go func() {
		resc <- result{
			name: name,
			err:  fn.name(fn.args),
		}
	}()
}
