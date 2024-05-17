#!/bin/bash

# docker run docker run --rm -v /path/to/workspace:/workspace pipego/runner/language/groovy:latest /workspace/jenkinsfile
docker build -f Dockerfile --rm -t pipego/runner/language/groovy:latest .
