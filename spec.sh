#!/bin/bash

repace() {
    echo $1
    sed -E "$1" ~/Code/yao/xiang-spec/README.md > ~/Code/yao/xiang-spec/README.new.md
    rm ~/Code/yao/xiang-spec/README.md 
    mv ~/Code/yao/xiang-spec/README.new.md ~/Code/yao/xiang-spec/README.md
}

make release
VERSION=$(go run . version)
rm -rf ../xiang-spec/xiang/*
cp  dist/release/xiang-* ~/Code/bin/
rm -f ~/Code/bin/xiang
ln -s ~/Code/bin/xiang-$VERSION-darwin-amd64 ~/Code/bin/xiang

# 更新 README.md 中的版本
# repace "s/\[[0-9]+\.[0-9]+\.[0-9]+\]/[$VERSION]/g"
# repace "s/\-[0-9]+\.[0-9]+\.[0-9]+\-/-$VERSION-/g"
# repace "s/Version\:[0-9]+\.[0-9]+\.[0-9]+/Version:$VERSION/g"

make clean