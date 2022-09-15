package server

import (
	"context"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	fl "github.com/pipego/runner/file"
	"github.com/pipego/runner/runner"
	pb "github.com/pipego/runner/server/proto"
)

const (
	Kind   = "runner"
	Layout = "20060102150405"
)

type Server interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Run(context.Context) error
}

type Config struct {
	Addr   string
	File   fl.File
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
	if err := s.initFile(ctx); err != nil {
		return errors.Wrap(err, "failed to init file")
	}

	if err := s.initRunner(ctx); err != nil {
		return errors.Wrap(err, "failed to init runner")
	}

	return nil
}

func (s *server) Deinit(ctx context.Context) error {
	_ = s.deinitRunner(ctx)
	_ = s.deinitFile(ctx)

	return nil
}

func (s *server) initFile(ctx context.Context) error {
	c := fl.DefaultConfig()
	if c == nil {
		return errors.New("failed to config")
	}

	s.cfg.File = fl.New(ctx, c)

	return s.cfg.File.Init(ctx)
}

func (s *server) deinitFile(ctx context.Context) error {
	return s.cfg.File.Deinit(ctx)
}

func (s *server) initRunner(ctx context.Context) error {
	c := runner.DefaultConfig()
	if c == nil {
		return errors.New("failed to config")
	}

	s.cfg.Runner = runner.New(ctx, c)

	return nil
}

func (s *server) deinitRunner(ctx context.Context) error {
	return nil
}

func (s *server) Run(_ context.Context) error {
	options := []grpc.ServerOption{grpc.MaxRecvMsgSize(math.MaxInt32), grpc.MaxSendMsgSize(math.MaxInt32)}

	g := grpc.NewServer(options...)
	pb.RegisterServerProtoServer(g, s)

	lis, _ := net.Listen("tcp", s.cfg.Addr)

	return g.Serve(lis)
}

func (s *server) SendServer(srv pb.ServerProto_SendServerServer) error {
	name, file, commands, livelog, err := s.recvClient(srv)
	if err != nil {
		return srv.Send(&pb.ServerReply{Error: err.Error()})
	}

	if len(file.GetContent()) != 0 && len(commands) != 0 {
		return srv.Send(&pb.ServerReply{Error: "file and commands not supported meanwhile"})
	}

	if livelog <= 0 {
		return srv.Send(&pb.ServerReply{Error: "invalid livelog"})
	}

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	if len(file.GetContent()) != 0 {
		n, err := s.loadFile(ctx, file.GetContent(), file.GetGzip())
		defer func(ctx context.Context, n string) {
			_ = s.cfg.File.Remove(ctx, n)
		}(ctx, n)
		if err != nil {
			return srv.Send(&pb.ServerReply{Error: "failed to load"})
		}
		commands = []string{"bash", n}
	}

	if err := s.cfg.Runner.Init(ctx, livelog); err != nil {
		return srv.Send(&pb.ServerReply{Error: "failed to init"})
	}

	defer func() {
		_ = s.cfg.Runner.Deinit(ctx)
	}()

	if err := s.cfg.Runner.Run(ctx, name, commands); err != nil {
		return srv.Send(&pb.ServerReply{Error: "failed to run"})
	}

	log := s.cfg.Runner.Tail(ctx)

L:
	for {
		select {
		case <-ctx.Done():
			break L
		case line, ok := <-log.Line:
			if ok {
				_ = srv.Send(&pb.ServerReply{
					Output: &pb.Output{
						Pos:     line.Pos,
						Time:    line.Time,
						Message: line.Message,
					}})
				if line.Message == "EOF" {
					break L
				}
			}
		}
	}

	return nil
}

func (s *server) recvClient(srv pb.ServerProto_SendServerServer) (name string, file *pb.File, commands []string,
	livelog int, err error) {
	for {
		r, err := srv.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil, nil, 0, errors.Wrap(err, "failed to receive")
		}

		if r.Kind != Kind {
			return "", nil, nil, 0, errors.New("invalid kind")
		}

		name = r.Spec.Task.GetName()
		file = r.Spec.Task.GetFile()
		commands = r.Spec.Task.GetCommands()
		livelog = int(r.Spec.Task.GetLivelog())

		break
	}

	return name, file, commands, livelog, nil
}

func (s *server) loadFile(ctx context.Context, data []byte, gzip bool) (string, error) {
	var buf []byte
	var err error

	if gzip {
		buf, err = s.cfg.File.Unzip(ctx, data)
		if err != nil {
			return "", errors.Wrap(err, "failed to unzip")
		}
	} else {
		buf = data
	}

	suffix := time.Now().Format(Layout)
	name := filepath.Join(string(os.PathSeparator), "tmp", "pipego-runner-file-"+suffix)

	if err = s.cfg.File.Write(ctx, name, buf); err != nil {
		_ = s.cfg.File.Remove(ctx, name)
		return "", errors.Wrap(err, "failed to write")
	}

	if s.cfg.File.Type(ctx, name) != fl.Bash {
		_ = s.cfg.File.Remove(ctx, name)
		return "", errors.New("invalid type")
	}

	return name, nil
}
