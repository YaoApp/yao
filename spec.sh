#!/bin/bash
make xiang
VERSION=$(go run . version)
rm -rf ../xiang-spec/xiang/*
cp -R dist/bin ../xiang-spec/xiang/"v$VERSION"
make clean