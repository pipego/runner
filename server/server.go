package server

import (
	"context"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	fl "github.com/pipego/runner/file"
	pb "github.com/pipego/runner/server/proto"
	"github.com/pipego/runner/task"
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
func (s *server) SendTask(srv pb.ServerProto_SendTaskServer) error {
	name, file, params, commands, livelog, err := s.recvTask(srv)
	if err != nil {
		return srv.Send(&pb.TaskReply{Error: err.Error()})
	}

	if len(file.GetContent()) != 0 && len(commands) != 0 {
		return srv.Send(&pb.TaskReply{Error: "file and commands not supported meanwhile"})
	}

	if livelog <= 0 {
		return srv.Send(&pb.TaskReply{Error: "invalid livelog"})
	}

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	f, err := s.newFile(ctx)
	if err != nil {
		return srv.Send(&pb.TaskReply{Error: "failed to new file"})
	}

	if err = f.Init(ctx); err != nil {
		return srv.Send(&pb.TaskReply{Error: "failed to init file"})
	}

	defer func(ctx context.Context) {
		_ = f.Deinit(ctx)
	}(ctx)

	if len(commands) != 0 {
		commands = []string{"bash", "-c", strings.Join(commands, " ")}
	} else if len(file.GetContent()) != 0 {
		n, e := s.loadFile(ctx, f, file.GetContent(), file.GetGzip())
		defer func(ctx context.Context, n string) {
			_ = f.Remove(ctx, n)
		}(ctx, n)
		if e != nil {
			return srv.Send(&pb.TaskReply{Error: "failed to load file"})
		}
		commands = []string{"bash", "-c", n}
	}

	t, err := s.newTask(ctx)
	if err != nil {
		return srv.Send(&pb.TaskReply{Error: "failed to new task"})
	}

	if err := t.Init(ctx, livelog); err != nil {
		return srv.Send(&pb.TaskReply{Error: "failed to init task"})
	}

	defer func() {
		_ = t.Deinit(ctx)
	}()

	if err := t.Run(ctx, name, s.buildEnvs(ctx, params), commands); err != nil {
		return srv.Send(&pb.TaskReply{Error: "failed to run task"})
	}

	log := t.Tail(ctx)

L:
	for {
		select {
		case <-ctx.Done():
			break L
		case line, ok := <-log.Line:
			if ok {
				_ = srv.Send(&pb.TaskReply{
					Output: &pb.TaskOutput{
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
func (s *server) recvTask(srv pb.ServerProto_SendTaskServer) (name string, file *pb.TaskFile, params []*pb.TaskParam,
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

func (s *server) newTask(ctx context.Context) (task.Task, error) {
	c := task.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	return task.New(ctx, c), nil
}

func (s *server) buildEnvs(_ context.Context, params []*pb.TaskParam) []string {
	var buf []string

	for _, item := range params {
		buf = append(buf, item.GetName()+"="+item.GetValue())
	}

	return buf
}
