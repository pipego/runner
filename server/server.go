package server

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"

	"github.com/pkg/errors"
	"github.com/soheilhy/cmux"
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
	l, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return errors.Wrap(err, "failed to listen")
	}

	m := cmux.New(l)

	gl := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
	gs := s.initRpc()
	go func() {
		if err := gs.Serve(gl); err != nil {
			fmt.Println("failed to serve: ", err)
		}
	}()

	hl := m.Match(cmux.HTTP1Fast())
	hs := s.initHttp()
	go func() {
		if err := hs.Serve(hl); err != nil {
			fmt.Println("failed to serve: ", err)
		}
	}()

	if err := m.Serve(); err != nil {
		return errors.Wrap(err, "failed to serve")
	}

	return nil
}

func (s *server) SendServer(in *pb.ServerRequest) (*pb.ServerReply, error) {
	return &pb.ServerReply{Message: "Hello " + in.GetMessage()}, nil
}

func (s *server) initRpc() *grpc.Server {
	options := []grpc.ServerOption{grpc.MaxRecvMsgSize(math.MaxInt32), grpc.MaxSendMsgSize(math.MaxInt32)}

	g := grpc.NewServer(options...)
	pb.RegisterServerProtoServer(g, &rpcServer{})
	reflection.Register(g)

	return g
}

func (s *server) initHttp() *http.Server {
	h := http.NewServeMux()

	h.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`pong`))
	})

	return &http.Server{
		Addr:    s.cfg.Addr,
		Handler: h,
	}
}
