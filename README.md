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
  --[no-]help              Show context-sensitive help (also try --help-long and --help-man).
  --[no-]version           Show application version.
  --listen-url=LISTEN-URL  Listen URL (host:port)
  --log-level="INFO"       Log level (DEBUG|INFO|WARN|ERROR)
```



## Protobuf

### 1. Task

```json
{
  "apiVersion": "v1",
  "kind": "runner",
  "metadata": {
    "name": "runner"
  },
  "spec": {
    "task": {
      "name": "task",
      "file": {
        "content": "bytes",
        "gzip": true
      },
      "params": [
        {
          "name": "env",
          "value": "val"
        }
      ],
      "commands": [
        "cmd",
        "argv"
      ],
      "log": {
        "width": 500
      }
    }
  }
}
```

> `task.file`: script file in bash
>
> `task.file.content`: bytes
>
> > The shebang in `task.file.content` should be one of below:
> >
> > `#!/bin/bash`
> >
> > `#!/usr/bin/env bash`
>
> `task.file.gzip`: boolean
>
> `task.params`: parameter and value
>
> > `name1=value1` (`$name1: value1`)
> >
> > `name2=$name1` (`$name2: value1`)
> >
> > `name3=$name2` (`$name3: value1`)
> >
> > `name4=$$name1` (`$name4: 790name1`, the pid of this script)
> >
> > `name4=${BASHPID}name1` (`$name4: 790name1`, the pid of current instance)
> >
> > `name5=#name1` (`$name5: #name1`, invalid symbol in Bash)
>
> `task.commands`: command and argument
>
> `task.log`: task log
>
> `task.log.width`: line width in runes (default: 500)

**Output**

```json
{
  "pos": 1,
  "time": "1136214245000000000",
  "message": "text"
}
```

> `pos`: line position
>
> `time`: unix timestamp
>
> `message`: line message in string
>
> > The tag in the line and file as below:
> >
> > `BOL`: break of line
> >
> > `EOF`: end of file



### 2. Glance

```json
{
  "apiVersion": "v1",
  "kind": "runner",
  "metadata": {
    "name": "runner"
  },
  "spec": {
    "glance": {
      "dir": {
        "path": "/path/to/name"
      },
      "file": {
        "path": "/path/to/name",
        "maxSize": 1000
      },
      "sys": {
        "enable": true
      }
    }
  }
}
```

> `glance.dir`: list directory contents
>
> `glance.file`: fetch file content in base64
>
> `glance.file.maxSize`: maximum file size in bytes
>
> `glance.sys`: show system info
>
> `glance.sys.enable`: boolean

**Output**

```json
{
  "dir": {
    "entries": [
      {
        "name": "name",
        "isDir": true,
        "size": 1000,
        "time": "2006-01-02 15:04:05",
        "user": "name",
        "group": "name",
        "mode": "drwxr-xr-x"
      }
    ]
  },
  "file": {
    "content": "base64",
    "readable": true
  },
  "sys": {
    "resource": {
      "allocatable": {
        "milliCPU": 16000,
        "memory": 12871671808,
        "storage": 269490393088
      },
      "requested": {
        "milliCPU": 12,
        "memory": 618688512,
        "storage": 19994185728
      }
    },
    "stats": {
      "cpu": {
        "total": "16 CPU",
        "used": "0%"
      },
      "host": "172.23.179.208",
      "memory": {
        "total": "11 GB",
        "used": "0 GB"
      },
      "os": "Ubuntu 20.04",
      "storage": {
        "total": "250 GB",
        "used": "18 GB"
      },
      "processes": [
        {
          "process": {
            "name": "init",
            "cmdline": "/init",
            "memory": 684032,
            "time": 1.00,
            "pid": 1
          },
          "threads": [
            {
              "name": "child",
              "cmdline": "/child",
              "memory": 684032,
              "time": 1.00,
              "pid": 2
            }
          ]
        }
      ]
    }
  },
  "error": "text"
}
```



### 3. Maint

```json
{
  "apiVersion": "v1",
  "kind": "runner",
  "metadata": {
    "name": "runner"
  },
  "spec": {
    "maint": {
      "clock": {
        "time": 1257894000,
        "sync": true
      }
    }
  }
}
```

> `maint.clock`: clock maintenance
>
> `maint.clock.time`: clock base time (unix time)
>
> `maint.clock.sync`: enable/disable clock synchronization
> >
> > The clock synchronization on Ubuntu
> >
> > ```bash
> > sudo apt install -y ntp ntpdate ntpstat
> > sudo service ntp stop
> > sudo ntpdate -s time.nist.gov
> > sudo service ntp start
> > ```

**Output**

```json
{
  "clock": {
    "diff": {
      "time": 100,
      "dangerous": true
    },
    "sync": {
      "status": 0
    }
  },
  "error": "text"
}
```

> `clock.diff`: clock difference
>
> > `time`: clock difference in milliseconds
> >
> > `dangerous`: if the difference is big enough to be considered dangerous
>
> `clock.sync`: clock synchronization
>
> > `status`: clock synchronization status
> >
> > `0`: clock is synchronised
> >
> > `1`: clock is not synchronised
> >
> > `2`: clock state is indeterminant



## License

Project License can be found [here](LICENSE).



## Reference

- [argo-workflows](https://github.com/argoproj/argo-workflows)
- [asynq](https://github.com/hibiken/asynq)
- [asynqmon](https://github.com/hibiken/asynqmon)
- [bufio-example](https://golang.org/src/bufio/example_test.go)
- [chanx](https://github.com/smallnest/chanx)
- [cuelang](https://cuelang.org)
- [cyclone-workflow](https://github.com/caicloud/cyclone)
- [dagger](https://dagger.io/)
- [drone-dag](https://github.com/drone/dag)
- [drone-livelog](https://github.com/harness/drone/tree/master/livelog)
- [drone-pipeline](https://docs.drone.io/pipeline/overview/)
- [drone-runner](https://github.com/drone-runners/drone-runner-exec)
- [gleam-workflow](https://github.com/chrislusf/gleam)
- [go-exec](https://gist.github.com/craftslab/1fe9151fbf069a9e1341e4daebe43b5c)
- [grpctest](https://github.com/grpc/grpc-go/tree/master/internal/grpctest)
- [grpc-streaming](https://www.freecodecamp.org/news/grpc-server-side-streaming-with-go/)
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
