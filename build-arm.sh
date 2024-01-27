#!/bin/sh
GOOS=linux \
GOARCH=arm \
CGO_ENABLED=1 \
CC=~/source/buildroot/output/host/bin/arm-buildroot-linux-gnueabihf-gcc \
PKG_CONFIG_PATH=~/source/buildroot/output/host/arm-buildroot-linux-gnueabihf/sysroot/usr/share/pkgconfig/ \
GOOS=linux \
GOARCH=arm \
go build -ldflags="-w -s" -o azur

#
