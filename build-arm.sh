#!/bin/sh
CGO_ENABLED=0 \
GOOS=linux \
GOARCH=arm \
go build -o azur
