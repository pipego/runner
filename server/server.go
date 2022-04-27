package server

import (
	"context"
	"math"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/pipego/runner/server/proto"
)

type Server interface {
	Run() error
}

type Config struct {
	Addr string
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

func (s *server) Run() error {
	return nil
}

func (s *server) initRpc() *grpc.Server {
	options := []grpc.ServerOption{grpc.MaxRecvMsgSize(math.MaxInt32), grpc.MaxSendMsgSize(math.MaxInt32)}

	g := grpc.NewServer(options...)
	pb.RegisterServerProtoServer(g, &rpcServer{})
	reflection.Register(g)

	return g
}

func (s *server) SendServer(in *pb.ServerRequest) (*pb.ServerReply, error) {
	return &pb.ServerReply{Message: "Hello " + in.GetMessage()}, nil
}
