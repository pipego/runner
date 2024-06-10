#!/bin/bash

docker run --rm -v "$PWD"/workspace:/workspace craftslab/groovy:latest /workspace/jenkinsfile
