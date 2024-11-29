# ===========================================
#  Yao Development
#  docker build \
#    --build-arg VERSION="${VERSION}"  \
#    --build-arg ARCH="${ARCH}"  \
#    -t yaoapp/yao-dev:${VERSION}-${ARCH} .
#
#  Build:
#  docker build --platform linux/amd64 --build-arg VERSION=0.9.1 --build-arg ARCH=amd64 -t yaoapp/yao:0.9.1-amd64-dev .
#  docker build --platform linux/arm64 --build-arg VERSION=0.9.1 --build-arg ARCH=arm64 -t yaoapp/yao:0.9.1-arm64-dev .
#
#  Tests:
#  docker run --rm yaoapp/yao:0.9.1-amd64-dev yao version
#  docker run -d -p 5099:5099 yaoapp/yao:0.9.1-amd64-dev
#
# ===========================================
FROM ubuntu:24.04
ARG VERSION
ARG ARCH
RUN groupadd -r yao && useradd -r -g yao yao && \
    apt-get update && \
    apt-get install -y curl sudo procps net-tools
RUN curl -fsSL "https://yao.moapi.ai/archives/yao-${VERSION}-linux-${ARCH}" > /usr/local/bin/yao && \
    chmod +x /usr/local/bin/yao && \
    mkdir -p /data/app

VOLUME /data/app
WORKDIR /data/app
EXPOSE 5099
CMD ["tail", "-f","/dev/null"]
