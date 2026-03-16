# ===========================================
#  Yao Development
#  docker build --build-arg VERSION="1.0.0-alpha" -t yaoapp/yao:1.0.0-alpha-dev .
#  docker run --rm yaoapp/yao:1.0.0-alpha-dev yao version
# ===========================================
FROM ubuntu:24.04
ARG VERSION
ARG TARGETARCH
RUN groupadd -r yao && useradd -r -g yao yao && \
    apt-get update && \
    apt-get install -y curl sudo procps net-tools
RUN curl -fsSL "https://pub-80136338e60643edbb55c6ca8a689cf8.r2.dev/archives/yao-${VERSION}-linux-${TARGETARCH}" > /usr/local/bin/yao && \
    chmod +x /usr/local/bin/yao && \
    mkdir -p /data/app

VOLUME /data/app
WORKDIR /data/app
EXPOSE 5099
CMD ["tail", "-f","/dev/null"]
