# /bin/bash
go clean --cache
CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ \
    CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
    go build -ldflags="-extldflags=-static"  -v -o ./yao-arm64

