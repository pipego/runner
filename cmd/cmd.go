package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/pipego/runner/config"
	"github.com/pipego/runner/server"
)

const (
	routineNum = -1
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

	if err := runServer(ctx, s); err != nil {
		return errors.Wrap(err, "failed to run server")
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

func runServer(ctx context.Context, srv server.Server) error {
	if err := srv.Init(ctx); err != nil {
		return errors.New("failed to init")
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(routineNum)

	g.Go(func() error {
		if err := srv.Run(ctx); err != nil {
			return errors.Wrap(err, "failed to run")
		}
		return nil
	})

	s := make(chan os.Signal, 1)

	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can"t be caught, so don't need add it
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)

	g.Go(func() error {
		<-s
		_ = srv.Deinit(ctx)
		return nil
	})

	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "failed to wait")
	}

	return nil
}
