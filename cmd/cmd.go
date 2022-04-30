package cmd

import (
	"context"
	"log"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/pipego/runner/config"
	"github.com/pipego/runner/server"
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

	s, err := initServer(c)
	if err != nil {
		return errors.Wrap(err, "failed to init server")
	}

	log.Println("running")

	if err := runPipe(s); err != nil {
		return errors.Wrap(err, "failed to run pipe")
	}

	log.Println("exiting")

	return nil
}

func initConfig() (*config.Config, error) {
	c := config.New()
	return c, nil
}

func initServer(_ *config.Config) (server.Server, error) {
	c := server.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	c.Addr = *listenUrl

	return server.New(context.Background(), c), nil
}

func runPipe(s server.Server) error {
	if err := s.Init(); err != nil {
		return errors.New("failed to init")
	}

	if err := s.Run(); err != nil {
		return errors.New("failed to run")
	}

	return nil
}
