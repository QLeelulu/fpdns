#!/usr/bin/env bash
set -x
export GOPROXY=https://goproxy.cn
export GO111MODULE=on
export CGO_ENABLED=0

go build -o ./bin/fpdns main.go
GOOS=linux   GOARCH=amd64 go build -o ./bin/fpdns_linux_amd64 main.go
GOOS=linux   GOARCH=arm   go build -o ./bin/fpdns_linux_arm main.go
GOOS=darwin  GOARCH=amd64 go build -o ./bin/fpdns_darwin_amd64 main.go
GOOS=windows GOARCH=amd64 go build -o ./bin/fpdns_windows_amd64.exe main.go
