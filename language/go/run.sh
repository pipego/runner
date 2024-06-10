#!/bin/bash

docker run --rm -v "$PWD"/workspace:/workspace craftslab/go:latest /workspace/main.go
