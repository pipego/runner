package cmd

import (
	"context"
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

func Run(ctx context.Context) error {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	c, err := initConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to init config")
	}

	s, err := initServer(ctx, c)
	if err != nil {
		return errors.Wrap(err, "failed to init server")
	}

	if err := runPipe(ctx, s); err != nil {
		return errors.Wrap(err, "failed to run pipe")
	}

	return nil
}

func initConfig(_ context.Context) (*config.Config, error) {
	c := config.New()
	return c, nil
}

func initServer(ctx context.Context, _ *config.Config) (server.Server, error) {
	c := server.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	c.Addr = *listenUrl

	return server.New(ctx, c), nil
}

func runPipe(ctx context.Context, s server.Server) error {
	if err := s.Init(ctx); err != nil {
		return errors.New("failed to init")
	}

	if err := s.Run(ctx); err != nil {
		return errors.New("failed to run")
	}

	return nil
}
