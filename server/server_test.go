package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"

	mock "github.com/pipego/runner/server/mock"
	pb "github.com/pipego/runner/server/proto"

	"github.com/pipego/runner/external/grpctest"
)

type rpcMsg struct {
	msg proto.Message
}

type rpcTest struct {
	grpctest.Tester
}

func TestRun(t *testing.T) {
	grpctest.RunSubTests(t, rpcTest{})
}

func (r *rpcMsg) Matches(msg interface{}) bool {
	m, ok := msg.(proto.Message)
	if !ok {
		return false
	}

	return proto.Equal(m, r.msg)
}

func (r *rpcMsg) String() string {
	return fmt.Sprintf("msg: %s", r.msg)
}

func (rpcTest) TestSendServer(t *testing.T) {
	helper := func(t *testing.T, client pb.ServerProtoClient) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		r, err := client.SendServer(ctx, &pb.ServerRequest{
			ApiVersion: "v1",
			Kind:       "runner",
			Metadata: &pb.Metadata{
				Name: "runner",
			},
			Spec: &pb.Spec{
				Tasks: []*pb.Task{
					{
						Name:     "name1",
						Commands: []string{"cmd1", "args1"},
						Depends:  []string{},
					},
					{
						Name:     "name2",
						Commands: []string{"cmd2", "args2"},
						Depends:  []string{"name1"},
					},
				},
			},
		})

		if err != nil || r.Error != "" {
			t.Errorf("mocking failed")
		}

		t.Log(r.Result)
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	req := &pb.ServerRequest{
		ApiVersion: "v1",
		Kind:       "runner",
		Metadata: &pb.Metadata{
			Name: "runner",
		},
		Spec: &pb.Spec{
			Tasks: []*pb.Task{
				{
					Name:     "name1",
					Commands: []string{"cmd1", "args1"},
					Depends:  []string{},
				},
				{
					Name:     "name2",
					Commands: []string{"cmd2", "args2"},
					Depends:  []string{"name1"},
				},
			},
		},
	}

	client := mock.NewMockServerProtoClient(ctrl)
	client.EXPECT().SendServer(
		gomock.Any(),
		&rpcMsg{msg: req},
	).Return(&pb.ServerReply{Error: ""}, nil)

	helper(t, client)
}
