#!/bin/sh
GOOS=linux \
GOARCH=arm \
go build -ldflags="-w -s" -o azur
