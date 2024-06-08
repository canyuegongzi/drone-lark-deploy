#!/bin/bash
export GOOS=linux
export GOARCH=amd64
go build -o lark main.go
tag=${tag:-"4.0"}
echo 'Docker Tag = '$tag
docker build -f dockerfile -t registry.cn-hangzhou.aliyuncs.com/canyuegongzi/drone-plugin-feishu:$tag .
docker push registry.cn-hangzhou.aliyuncs.com/canyuegongzi/drone-plugin-feishu:$tag
docker image prune -f