#!/bin/bash

docker run --rm -v "$PWD"/workspace:/workspace craftslab/java:latest /workspace/main.java
