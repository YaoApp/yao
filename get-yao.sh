#!/bin/bash

# 定义 GitHub 仓库信息
REPO="sjzsdu/yao"
CONST_URL="https://raw.githubusercontent.com/$REPO/main/share/const.go"

# 从 const.go 文件获取最新版本号
VERSION=$(curl -s $CONST_URL | grep 'const VERSION' | awk -F '"' '{print $2}')

# 检测操作系统类型和架构
OS="unknown"
ARCH="unknown"

# 检测操作系统
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="darwin"
else
    echo "不支持的操作系统类型: $OSTYPE"
    exit 1
fi

# 检测架构
MACHINE_TYPE=$(uname -m)
if [[ "$MACHINE_TYPE" == "x86_64" ]]; then
    ARCH="amd64"
elif [[ "$MACHINE_TYPE" == "arm64" ]] || [[ "$MACHINE_TYPE" == "aarch64" ]]; then
    ARCH="arm64"
else
    echo "不支持的系统架构: $MACHINE_TYPE"
    exit 1
fi

# 构建下载 URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/v$VERSION/yao-$VERSION-$OS-$ARCH"

# 下载文件
echo "正在下载 yao-$OS-$ARCH 版本 $VERSION ..."
curl -L $DOWNLOAD_URL -o yao
echo "下载 URL: $DOWNLOAD_URL"

# 使文件可执行
chmod +x yao

# 确定安装目录
INSTALL_DIR="/usr/local/bin"
if [[ ! -w "$INSTALL_DIR" ]]; then
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# 移动文件到安装目录
echo "正在安装 yao 到 $INSTALL_DIR ..."
mv yao "$INSTALL_DIR/yao"

# 检查安装目录是否在 PATH 中
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "请将 $INSTALL_DIR 添加到你的 PATH 环境变量中。"
    echo "你可以通过在 ~/.bashrc 或 ~/.zshrc 中添加以下行来实现："
    echo "export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo "yao 安装完成！"
