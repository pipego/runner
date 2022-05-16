package builder

import (
	"context"
)

type Builder interface {
	Run(*Config) (Dag, error)
}

type Config struct {
	ApiVersion string
	Kind       string
	MetaData   MetaData
	Spec       Spec
}

type MetaData struct {
	Name string
}

type Spec struct {
	Tasks []Task
}

type Task struct {
	Name    string
	Command []string
	Depend  []string
}

type Dag struct {
	Vertex []Vertex
	Edge   []Edge
}

type Vertex struct {
	Name string
	Run  []string
}

type Edge struct {
	From string
	To   string
}

type builder struct {
	cfg *Config
}

func New(_ context.Context, cfg *Config) Builder {
	return &builder{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (b *builder) Run(cfg *Config) (Dag, error) {
	dag := Dag{}

	for _, task := range cfg.Spec.Tasks {
		d := Vertex{
			Name: task.Name,
			Run:  task.Command,
		}
		dag.Vertex = append(dag.Vertex, d)

		for _, dep := range task.Depend {
			e := Edge{
				From: dep,
				To:   task.Name,
			}
			dag.Edge = append(dag.Edge, e)
		}
	}

	return dag, nil
}
