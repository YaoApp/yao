#!/bin/bash
cd /app && \
git clone https://github.com/yaoapp/kun.git /app/kun && \
git clone https://github.com/yaoapp/xun.git /app/xun && \
git clone https://github.com/yaoapp/gou.git /app/gou && \
git clone https://github.com/yaoapp/v8go.git /app/v8go && \
git clone https://github.com/yaoapp/xgen.git /app/xgen-v1.0 && \
git clone https://github.com/yaoapp/yao-init.git /app/yao-init && \
git clone https://github.com/yaoapp/yao.git /app/yao


cd /app/yao && \
export VERSION=$(cat share/const.go  |grep 'const VERSION' | awk '{print $4}' | sed "s/\"//g") 

cd /app/yao && make tools && make artifacts-linux
mv /app/yao/dist/release/* /data/