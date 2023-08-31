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
	Addr string
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
	return nil
}

func (s *server) Deinit(ctx context.Context) error {
	return nil
}

func (s *server) Run(_ context.Context) error {
	options := []grpc.ServerOption{grpc.MaxRecvMsgSize(math.MaxInt32), grpc.MaxSendMsgSize(math.MaxInt32)}

	g := grpc.NewServer(options...)
	pb.RegisterServerProtoServer(g, s)

	lis, _ := net.Listen("tcp", s.cfg.Addr)

	return g.Serve(lis)
}

// nolint: gocyclo
func (s *server) SendServer(srv pb.ServerProto_SendServerServer) error {
	name, file, params, commands, livelog, err := s.recvClient(srv)
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

	f, err := s.newFile(ctx)
	if err != nil {
		return srv.Send(&pb.ServerReply{Error: "failed to new file"})
	}

	if err = f.Init(ctx); err != nil {
		return srv.Send(&pb.ServerReply{Error: "failed to init file"})
	}

	defer func(ctx context.Context) {
		_ = f.Deinit(ctx)
	}(ctx)

	if len(file.GetContent()) != 0 {
		n, e := s.loadFile(ctx, f, file.GetContent(), file.GetGzip())
		defer func(ctx context.Context, n string) {
			_ = f.Remove(ctx, n)
		}(ctx, n)
		if e != nil {
			return srv.Send(&pb.ServerReply{Error: "failed to load file"})
		}
		commands = []string{"bash", "-c", n}
	}

	r, err := s.newRunner(ctx)
	if err != nil {
		return srv.Send(&pb.ServerReply{Error: "failed to new runner"})
	}

	if err := r.Init(ctx, livelog); err != nil {
		return srv.Send(&pb.ServerReply{Error: "failed to init runner"})
	}

	defer func() {
		_ = r.Deinit(ctx)
	}()

	if err := r.Run(ctx, name, s.buildEnvs(ctx, params), commands); err != nil {
		return srv.Send(&pb.ServerReply{Error: "failed to run runner"})
	}

	log := r.Tail(ctx)

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

// nolint: gocritic
func (s *server) recvClient(srv pb.ServerProto_SendServerServer) (name string, file *pb.File, params []*pb.Param,
	commands []string, livelog int, err error) {
	for {
		r, err := srv.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil, nil, nil, 0, errors.Wrap(err, "failed to receive")
		}

		if r.Kind != Kind {
			return "", nil, nil, nil, 0, errors.New("invalid kind")
		}

		name = r.Spec.Task.GetName()
		file = r.Spec.Task.GetFile()
		params = r.Spec.Task.GetParams()
		commands = r.Spec.Task.GetCommands()
		livelog = int(r.Spec.Task.GetLivelog())

		break
	}

	return name, file, params, commands, livelog, nil
}

func (s *server) newFile(ctx context.Context) (fl.File, error) {
	c := fl.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	return fl.New(ctx, c), nil
}

func (s *server) loadFile(ctx context.Context, file fl.File, data []byte, gzip bool) (string, error) {
	var buf []byte
	var err error

	if gzip {
		buf, err = file.Unzip(ctx, data)
		if err != nil {
			return "", errors.Wrap(err, "failed to unzip")
		}
	} else {
		buf = data
	}

	suffix := time.Now().Format(Layout)
	name := filepath.Join(string(os.PathSeparator), "tmp", "pipego-runner-file-"+suffix)

	if err = file.Write(ctx, name, buf); err != nil {
		_ = file.Remove(ctx, name)
		return "", errors.Wrap(err, "failed to write")
	}

	if file.Type(ctx, name) != fl.Bash {
		_ = file.Remove(ctx, name)
		return "", errors.New("invalid type")
	}

	return name, nil
}

func (s *server) newRunner(ctx context.Context) (runner.Runner, error) {
	c := runner.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	return runner.New(ctx, c), nil
}

func (s *server) buildEnvs(_ context.Context, params []*pb.Param) []string {
	var buf []string

	for _, item := range params {
		buf = append(buf, item.GetName()+"="+item.GetValue())
	}

	return buf
}
