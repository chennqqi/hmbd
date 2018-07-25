#!/bin/bash
#rm phpdecoded
#go build -v
go build -ldflags "-X main.Version=$(cat VERSION) -X main.BuildTime=$(date -u +%Y%m%d)" -v 
sudo docker build -t "sort/hmbd:$(cat VERSION)" -f Dockerfile.private .

