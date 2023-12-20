#!/bin/bash

CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o threaddump-analyzer

