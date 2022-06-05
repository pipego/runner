# runner

[![Build Status](https://github.com/pipego/runner/workflows/ci/badge.svg?branch=main&event=push)](https://github.com/pipego/runner/actions?query=workflow%3Aci)
[![codecov](https://codecov.io/gh/pipego/runner/branch/main/graph/badge.svg?token=El8oiyaIsD)](https://codecov.io/gh/pipego/runner)
[![Go Report Card](https://goreportcard.com/badge/github.com/pipego/runner)](https://goreportcard.com/report/github.com/pipego/runner)
[![License](https://img.shields.io/github/license/pipego/runner.svg)](https://github.com/pipego/runner/blob/main/LICENSE)
[![Tag](https://img.shields.io/github/tag/pipego/runner.svg)](https://github.com/pipego/runner/tags)



## Introduction

*runner* is the runner of [pipego](https://github.com/pipego) written in Go.



## Prerequisites

- Go >= 1.18.0



## Run

```bash
version=latest make build
./bin/runner --listen-url=:29090
```



## Docker

```bash
version=latest make docker
docker run ghcr.io/pipego/runner:latest --listen-url=:29090
```



## Usage

```
usage: runner --listen-url=LISTEN-URL [<flags>]

pipego runner

Flags:
  --help                   Show context-sensitive help (also try --help-long and --help-man).
  --version                Show application version.
  --listen-url=LISTEN-URL  Listen URL (host:port)
```



## Protobuf

```json
{
  "apiVersion": "v1",
  "kind": "runner",
  "metadata": {
    "name": "runner"
  },
  "spec": {
    "tasks": [
      {
        "name": "name1",
        "commands": [
          "cmd1",
          "argv1"
        ],
        "depends": []
      },
      {
        "name": "name2",
        "commands": [
          "cmd2",
          "argv2"
        ],
        "depends": [
          "name1"
        ]
      }
    ]
  }
}
```



## License

Project License can be found [here](LICENSE).



## Reference

- [asynq](https://github.com/hibiken/asynq)
- [asynqmon](https://github.com/hibiken/asynqmon)
- [bufio-example](https://golang.org/src/bufio/example_test.go)
- [cuelang](https://cuelang.org)
- [dagger](https://dagger.io/)
- [drone-dag](https://github.com/drone/dag)
- [drone-logs](https://github.com/harness/drone/blob/master/core/logs.go)
- [drone-pipeline](https://docs.drone.io/pipeline/overview/)
- [grpctest](https://github.com/grpc/grpc-go/tree/master/internal/grpctest)
- [kube-parallelize](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/parallelize/parallelism.go)
- [kube-schduler](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/schedule_one.go)
- [kube-scheduling](https://cloud.tencent.com/developer/article/1644857)
- [kube-scheduling](https://kubernetes.io/zh/docs/concepts/scheduling-eviction/kube-scheduler/)
- [kube-scheduling](https://kubernetes.io/zh/docs/reference/scheduling/config/)
- [kube-workqueue](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/client-go/util/workqueue)
- [machinery](https://github.com/RichardKnop/machinery/blob/master/v2/example/go-redis/main.go)
- [termui](https://github.com/gizak/termui)
- [websocket-command](https://github.com/gorilla/websocket/tree/master/examples/command)
- [wiki-dag](https://en.wikipedia.org/wiki/Directed_acyclic_graph)
