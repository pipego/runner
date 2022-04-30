package server

import (
	"context"
	"math"
	"net"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	pb "github.com/pipego/runner/server/proto"

	"github.com/pipego/runner/builder"
	"github.com/pipego/runner/runner"
)

type Server interface {
	Init() error
	Run() error
}

type Config struct {
	Addr    string
	Builder builder.Builder
	Runner  runner.Runner
}

type server struct {
	cfg *Config
}

type rpcServer struct {
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

func (s *server) Init() error {
	if err := s.initBuilder(); err != nil {
		return errors.Wrap(err, "failed to init builder")
	}

	if err := s.initRunner(); err != nil {
		return errors.Wrap(err, "failed to init runner")
	}

	return nil
}

func (s *server) initBuilder() error {
	b := builder.DefaultConfig()
	if b == nil {
		return errors.New("failed to config")
	}

	s.cfg.Builder = builder.New(context.Background(), b)

	return nil
}

func (s *server) initRunner() error {
	r := runner.DefaultConfig()
	if r == nil {
		return errors.New("failed to config")
	}

	s.cfg.Runner = runner.New(context.Background(), r)

	return nil
}

func (s *server) Run() error {
	options := []grpc.ServerOption{grpc.MaxRecvMsgSize(math.MaxInt32), grpc.MaxSendMsgSize(math.MaxInt32)}

	g := grpc.NewServer(options...)
	pb.RegisterServerProtoServer(g, &rpcServer{})

	lis, _ := net.Listen("tcp", s.cfg.Addr)

	return g.Serve(lis)
}

func (s *server) SendServer(in *pb.ServerRequest) (*pb.ServerReply, error) {
	helper := func() []builder.Task {
		var buf []builder.Task
		tasks := in.GetTasks()
		for _, val := range tasks {
			b := builder.Task{
				Name:     val.GetName(),
				Commands: val.GetCommands(),
				Depends:  val.GetDepends(),
			}
			buf = append(buf, b)
		}
		return buf
	}

	cfg := &builder.Config{
		Kind:  in.GetKind(),
		Type:  in.GetType(),
		Name:  in.GetName(),
		Tasks: helper(),
	}

	b, err := s.cfg.Builder.Run(cfg)
	if err != nil {
		return &pb.ServerReply{Message: "failed to build"}, nil
	}

	if err := s.cfg.Runner.Run(&b); err != nil {
		return &pb.ServerReply{Message: "failed to run"}, nil
	}

	return &pb.ServerReply{Message: ""}, nil
}
