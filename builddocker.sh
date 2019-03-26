#!/bin/bash
go build -o simpledns -ldflags "-s -w" -v 
sudo docker build -t "sort/simpledns:$(cat VERSION)" -f Dockerfile.local .

