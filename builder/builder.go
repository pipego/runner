package builder

import (
	"context"
)

type Builder interface {
	Run(*Config) (Dag, error)
}

type Config struct {
	Kind  string
	Type  string
	Name  string
	Tasks []Task
}

type Task struct {
	Name     string
	Commands []string
	Depends  []string
}

type Dag struct {
	// TODO
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
	// TODO
	return Dag{}, nil
}
