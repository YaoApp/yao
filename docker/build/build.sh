#!/bin/bash
cd /app && \
git clone https://github.com/yaoapp/cui.git /app/cui-v1.0 && \
git clone https://github.com/yaoapp/yao-init.git /app/yao-init && \
git clone https://github.com/yaoapp/yao.git /app/yao

# yao/go.mod 现在直接依赖发布版 gou / v8go；make deps 会准备 v8go 原生库
        

cd /app/yao && \
export VERSION=$(cat share/const.go  |grep 'const VERSION' | awk '{print $4}' | sed "s/\"//g") 

cd /app/yao && make tools && make artifacts-linux
mv /app/yao/dist/release/* /data/