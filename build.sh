#!/bin/bash

ver=$(git describe --tags --abbrev=0)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o threaddump-analyzer-arm64-macos_$ver
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o threaddump-analyzer-amd64-linux_$ver

