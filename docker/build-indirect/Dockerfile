# 使用 yaoapp/yao-build:0.10 作为基础镜像
FROM yaoapp/yao-build:0.10.4

# 将 entrypoint.sh 复制到容器内的 /app/ 目录
COPY ../build/build.sh /app/

# 设置工作目录
WORKDIR /app

# 确保 build.sh 具有执行权限
RUN chmod +x /app/build.sh

# 设置容器启动时执行的命令
CMD ["/app/build.sh"]
