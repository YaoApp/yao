# ===========================================
#  Yao Production
#  docker build --build-arg VERSION="1.0.0-alpha" -t yaoapp/yao:1.0.0-alpha .
#  docker run --rm yaoapp/yao:1.0.0-alpha yao version
# ===========================================
FROM alpine:latest
ARG VERSION
ARG TARGETARCH
RUN apk --no-cache add curl 
RUN curl -fsSL "https://pub-80136338e60643edbb55c6ca8a689cf8.r2.dev/archives/yao-${VERSION}-linux-${TARGETARCH}-prod" > /usr/local/bin/yao && \
    chmod +x /usr/local/bin/yao && \
    addgroup -S yao && adduser -S -G yao yao && \
    mkdir -p /data/app && \
    chown -R yao:yao /data/app

USER yao
VOLUME /data/app
WORKDIR /data/app
EXPOSE 5099
CMD ["/usr/local/bin/yao", "start"]
