package cmd

import (
	"context"
	"log"
	"os"

	"github.com/pipego/runner/config"
	"github.com/pipego/runner/runner"
	"github.com/pipego/runner/server"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app       = kingpin.New("runner", "pipego runner").Version(config.Version + "-build-" + config.Build)
	listenUrl = app.Flag("listen-url", "Listen URL (host:port)").Required().String()
)

func Run() error {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	c, err := initConfig()
	if err != nil {
		return errors.Wrap(err, "failed to init config")
	}

	r, err := initRunner(c)
	if err != nil {
		return errors.Wrap(err, "failed to init runner")
	}

	s, err := initServer(c)
	if err != nil {
		return errors.Wrap(err, "failed to init server")
	}

	log.Println("running")

	if err := runPipe(c, r, s); err != nil {
		return errors.Wrap(err, "failed to run pipe")
	}

	log.Println("exiting")

	return nil
}

func initConfig() (*config.Config, error) {
	c := config.New()
	return c, nil
}

func initRunner(_ *config.Config) (runner.Runner, error) {
	c := runner.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	return runner.New(context.Background(), c), nil
}

func initServer(_ *config.Config) (server.Server, error) {
	c := server.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	c.Addr = *listenUrl

	return server.New(context.Background(), c), nil
}

func runPipe(c *config.Config, r runner.Runner, s server.Server) error {
	return nil
}
