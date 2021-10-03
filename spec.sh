#!/bin/bash
make release
VERSION=$(go run . version)
rm -rf ../xiang-spec/xiang/*
cp  dist/release/xiang-* ../xiang-spec/xiang/
rm -f ~/Code/bin/xiang
ln -s ~/Code/yao/xiang-spec/xiang/xiang-$VERSION-darwin-amd64 ~/Code/bin/xiang
make clean