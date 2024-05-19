#!/bin/bash

docker run docker run --rm -v "$PWD"/workspace:/workspace pipego/runner/language/groovy:latest /workspace/jenkinsfile
