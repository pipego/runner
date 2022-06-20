#!/bin/bash

# Install protoc
#OS=$(go env GOOS)
#PB_REL="https://github.com/protocolbuffers/protobuf/releases"
#VERSION="3.20.1"
#curl -LO $PB_REL/download/v${VERSION}/protoc-${VERSION}-${OS}-x86_64.zip
#unzip protoc-${VERSION}-${OS}-x86_64.zip -d protoc-${VERSION}
#sudo mv protoc-${VERSION} /opt/
#export PATH=/opt/protoc-${VERSION}/bin:$PATH

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
# TODO
