# ===========================================
#  Yao Production
#  docker build \
#    --build-arg VERSION="${VERSION}"  \
#    --build-arg ARCH="${ARCH}"  \
#    -t yaoapp/yao-dev:${VERSION}-${ARCH} .
#
#  Build:
#  docker build --platform linux/amd64 --build-arg VERSION=0.9.1 --build-arg ARCH=amd64 -t yaoapp/yao:0.9.1-amd64 .
#  docker build --platform linux/arm64 --build-arg VERSION=0.9.1 --build-arg ARCH=arm64 -t yaoapp/yao:0.9.1-arm64 .
#
#  Tests:
#  docker run --rm yaoapp/yao:0.9.1-amd64 yao version
#  docker run -d -p 5099:5099 yaoapp/yao:0.9.1-amd64
#
# ===========================================
FROM alpine:latest
ARG VERSION
ARG ARCH
RUN apk --no-cache add curl 
RUN curl -fsSL "https://yao.moapi.ai/archives/yao-${VERSION}-linux-${ARCH}" > /usr/local/bin/yao && \
    chmod +x /usr/local/bin/yao && \
    addgroup -S yao && adduser -S -G yao yao && \
    mkdir -p /data/app && \
    chown -R yao:yao /data/app

USER yao
VOLUME /data/app
WORKDIR /data/app
EXPOSE 5099
CMD ["/usr/local/bin/yao", "start"]
