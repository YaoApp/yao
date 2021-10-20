#!/bin/bash
make release
VERSION=$(go run . version)
rm -rf ../xiang-spec/xiang/*
cp  dist/release/xiang-* ../xiang-spec/xiang/
rm -f ~/Code/bin/xiang
ln -s ~/Code/yao/xiang-spec/xiang/xiang-$VERSION-darwin-amd64 ~/Code/bin/xiang

# 更新 README.md 中的版本号
sed "s/[ v-][0-9]\.[0-9]\.[0-9]/$VERSION/g" ~/Code/yao/xiang-spec/README.md > ~/Code/yao/xiang-spec/README.new.md
rm ~/Code/yao/xiang-spec/README.md 
mv ~/Code/yao/xiang-spec/README.new.md ~/Code/yao/xiang-spec/README.md
make clean