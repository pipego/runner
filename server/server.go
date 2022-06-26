package server

import (
	"context"
	"math"
	"net"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/pipego/runner/runner"
	pb "github.com/pipego/runner/server/proto"
)

const (
	KIND    = "runner"
	TIMEOUT = 24 * time.Hour
)

type Server interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Run(context.Context) error
}

type Config struct {
	Addr   string
	Runner runner.Runner
}

type server struct {
	cfg *Config
	pb.UnimplementedServerProtoServer
}

func New(_ context.Context, cfg *Config) Server {
	return &server{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (s *server) Init(ctx context.Context) error {
	if err := s.initRunner(ctx); err != nil {
		return errors.Wrap(err, "failed to init runner")
	}

	return nil
}

func (s *server) Deinit(ctx context.Context) error {
	_ = s.deinitRunner(ctx)
	return nil
}

func (s *server) initRunner(ctx context.Context) error {
	r := runner.DefaultConfig()
	if r == nil {
		return errors.New("failed to config")
	}

	s.cfg.Runner = runner.New(ctx, r)

	return s.cfg.Runner.Init(ctx)
}

func (s *server) deinitRunner(ctx context.Context) error {
	return s.cfg.Runner.Deinit(ctx)
}

func (s *server) Run(_ context.Context) error {
	options := []grpc.ServerOption{grpc.MaxRecvMsgSize(math.MaxInt32), grpc.MaxSendMsgSize(math.MaxInt32)}

	g := grpc.NewServer(options...)
	pb.RegisterServerProtoServer(g, s)

	lis, _ := net.Listen("tcp", s.cfg.Addr)

	return g.Serve(lis)
}

func (s *server) SendServer(in *pb.ServerRequest, srv pb.ServerProto_SendServerServer) error {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	if in.GetKind() != KIND {
		return srv.Send(&pb.ServerReply{Error: "invalid kind"})
	}

	name := in.GetSpec().GetTask().GetName()
	args := in.GetSpec().GetTask().GetCommands()

	if err := s.cfg.Runner.Run(ctx, name, args, cancel); err != nil {
		return srv.Send(&pb.ServerReply{Error: "failed to run"})
	}

	log := s.cfg.Runner.Tail(ctx)

L:
	for {
		select {
		case line := <-log.Line:
			_ = srv.Send(&pb.ServerReply{
				Output: &pb.Output{
					Pos:     line.Pos,
					Time:    line.Time,
					Message: line.Message,
				},
			})
		case <-ctx.Done():
			break L
		}
	}

	return nil
}
