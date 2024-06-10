#!/bin/bash

docker run --rm -v "$PWD"/workspace:/workspace craftslab/rust:latest /workspace/main.rs
