# /bin/bash
# V8 
go clean --cache
# cp ../gou/go.mod ../gou/go.mod.bak
# echo "replace rogchap.com/v8go => ../v8go" >> ../gou/go.mod

CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ \
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
go build -ldflags="-extldflags=-static"  -v -o ./yao-arm64

# cp ../gou/go.mod.bak ../gou/go.mod
# rm ../gou/go.mod.bak

