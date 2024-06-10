#!/bin/bash

docker run --rm -v "$PWD"/workspace:/workspace craftslab/python:latest /workspace/main.py
