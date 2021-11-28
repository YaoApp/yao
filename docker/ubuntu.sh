#!/bin/bash
# version cat /etc/issue
apt-get update
apt-get -y install curl vim git iputils-ping build-essential gcc-arm-linux-gnueabi

# Install Go
cd /usr/local && tar xvfz /code/xiang/docker/go1.17.3.linux-amd64.tar.gz
ln -s /usr/local/go/bin/go /usr/local/bin/go
ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

#  Install Node

export GO111MODULE="on"
export GOPATH="/root/go"
export GOPROXY="https://goproxy.cn"
export http_proxy="socks5://192.168.199.178:1080"
export https_proxy="socks5://192.168.199.178:1080"
export NODE_OPTIONS="--max-old-space-size=5120"

export PATH=~/go/bin:$PATH