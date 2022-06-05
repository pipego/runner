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

const (
	KIND = "runner"
)

type Server interface {
	Init(context.Context) error
	Run(context.Context) error
}

type Config struct {
	Addr    string
	Builder builder.Builder
	Runner  runner.Runner
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
	if err := s.initBuilder(ctx); err != nil {
		return errors.Wrap(err, "failed to init builder")
	}

	if err := s.initRunner(ctx); err != nil {
		return errors.Wrap(err, "failed to init runner")
	}

	return nil
}

func (s *server) initBuilder(ctx context.Context) error {
	b := builder.DefaultConfig()
	if b == nil {
		return errors.New("failed to config")
	}

	s.cfg.Builder = builder.New(ctx, b)

	return nil
}

func (s *server) initRunner(ctx context.Context) error {
	r := runner.DefaultConfig()
	if r == nil {
		return errors.New("failed to config")
	}

	s.cfg.Runner = runner.New(ctx, r)

	return nil
}

func (s *server) Run(_ context.Context) error {
	options := []grpc.ServerOption{grpc.MaxRecvMsgSize(math.MaxInt32), grpc.MaxSendMsgSize(math.MaxInt32)}

	g := grpc.NewServer(options...)
	pb.RegisterServerProtoServer(g, s)

	lis, _ := net.Listen("tcp", s.cfg.Addr)

	return g.Serve(lis)
}

func (s *server) SendServer(in *pb.ServerRequest, srv pb.ServerProto_SendServerServer) error {
	metaDataHelper := func() builder.MetaData {
		var metadata builder.MetaData
		metadata.Name = in.GetMetadata().Name
		return metadata
	}

	specHelper := func() builder.Spec {
		var tasks []builder.Task
		var spec builder.Spec
		buf := in.GetSpec().GetTasks()
		for _, val := range buf {
			b := builder.Task{
				Name:     val.GetName(),
				Commands: val.GetCommands(),
				Depends:  val.GetDepends(),
			}
			tasks = append(tasks, b)
		}
		spec.Tasks = tasks
		return spec
	}

	if in.GetKind() != KIND {
		return srv.Send(&pb.ServerReply{Error: "invalid kind"})
	}

	cfg := &builder.Config{
		ApiVersion: in.GetApiVersion(),
		Kind:       in.GetKind(),
		MetaData:   metaDataHelper(),
		Spec:       specHelper(),
	}

	b, err := s.cfg.Builder.Run(context.Background(), cfg)
	if err != nil {
		return srv.Send(&pb.ServerReply{Error: "failed to build"})
	}

	if err := s.cfg.Runner.Run(context.Background(), &b); err != nil {
		return srv.Send(&pb.ServerReply{Error: "failed to run"})
	}

	return srv.Send(&pb.ServerReply{Error: "completed"})
}
