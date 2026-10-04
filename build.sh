#!/bin/sh
set -e
# Linux EleBBS/DOOR32 uses a Unix file descriptor in door32.sys.
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o affiliate-linux-amd64 .
GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o affiliate-linux-386 .
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o affiliate-linux-arm64 .
echo "Built affiliate-linux-amd64 affiliate-linux-386 affiliate-linux-arm64"
