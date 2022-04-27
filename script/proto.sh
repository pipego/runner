#!/bin/bash

# Install protoc
#PB_REL="https://github.com/protocolbuffers/protobuf/releases"
#curl -LO $PB_REL/download/v3.20.1/protoc-3.20.1-linux-x86_64.zip
#unzip protoc-3.20.1-linux-x86_64.zip -d protoc-3.20.1
#sudo mv protoc-3.20.1 /opt/
#export PATH=/opt/protoc-3.20.1/bin:$PATH

# Install plugins
#go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
#export PATH="$PATH:$(go env GOPATH)/bin"

# Install mock
#go install github.com/golang/mock/mockgen@latest
#export PATH="$PATH:$(go env GOPATH)/bin"

# Build proto
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./server/proto/server.proto

# Build mock
mockgen github.com/pipego/runner/server/proto ServerProtoClient > server/mock/server_mock.go
