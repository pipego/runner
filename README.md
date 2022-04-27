# runner

[![Build Status](https://github.com/pipego/runner/workflows/ci/badge.svg?branch=main&event=push)](https://github.com/pipego/runner/actions?query=workflow%3Aci)
[![codecov](https://codecov.io/gh/pipego/runner/branch/main/graph/badge.svg?token=El8oiyaIsD)](https://codecov.io/gh/pipego/runner)
[![Go Report Card](https://goreportcard.com/badge/github.com/pipego/runner)](https://goreportcard.com/report/github.com/pipego/runner)
[![License](https://img.shields.io/github/license/pipego/runner.svg)](https://github.com/pipego/runner/blob/main/LICENSE)
[![Tag](https://img.shields.io/github/tag/pipego/runner.svg)](https://github.com/pipego/runner/tags)



## Introduction

*runner* is the runner of [pipego](https://github.com/pipego) written in Go.



## Prerequisites

- Go >= 1.17.0



## Run

```bash
version=latest make build
./bin/runner --config-file="$PWD/config/config.yml --listen-url=:29090"
```



## Docker

```bash
version=latest make docker
docker run -v "$PWD"/config:/tmp ghcr.io/pipego/runner:latest --config-file="/tmp/config.yml --listen-url=:29090"
```



## Usage

```
usage: runner --config-file=CONFIG-FILE --listen-url=LISTEN-URL [<flags>]

pipego runner

Flags:
  --help                     Show context-sensitive help (also try --help-long and --help-man).
  --version                  Show application version.
  --config-file=CONFIG-FILE  Config file (.yml)
  --listen-url=LISTEN-URL    Listen URL (host:port)
```



## Settings

*runner* parameters can be set in the directory [config](https://github.com/pipego/runner/blob/main/config).

An example of configuration in [config.yml](https://github.com/pipego/runner/blob/main/config/config.yml):

```yaml
apiVersion: v1
kind: runner
metadata:
  name: runner
spec:
```



## License

Project License can be found [here](LICENSE).



## Reference

- [asynq](https://github.com/hibiken/asynq)
- [asynqmon](https://github.com/hibiken/asynqmon)
- [cuelang](https://cuelang.org)
- [dag](https://github.com/drone/dag)
- [dagger](https://dagger.io/)
- [drone](https://drone.io)
- [grpctest](https://github.com/grpc/grpc-go/tree/master/internal/grpctest)
- [machinery](https://github.com/RichardKnop/machinery/blob/master/v2/example/go-redis/main.go)
- [termui](https://github.com/gizak/termui)
