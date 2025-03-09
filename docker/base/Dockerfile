# ===========================================
#  Yao Base Image
#  A minimal base image for running Yao applications
#
#  Build:
#  docker build --platform linux/amd64 --build-arg VERSION=0.10.4 --build-arg ARCH=amd64 -t yaoapp/yao-base:0.10.4-amd64 .
#  docker build --platform linux/arm64 --build-arg VERSION=0.10.4 --build-arg ARCH=arm64 -t yaoapp/yao-base:0.10.4-arm64 .
#
#  Tests:
#  docker run --rm yaoapp/yao-base:0.10.4-amd64 yao version
#
# ===========================================
FROM alpine:latest
ARG VERSION
ARG ARCH

# Install minimal dependencies
RUN apk --no-cache add curl 

# Copy and install Yao binary from build artifacts
COPY dist/yao-${VERSION}-linux-${ARCH} /usr/local/bin/yao
RUN chmod +x /usr/local/bin/yao && \
    addgroup -S yao && adduser -S -G yao yao

# Switch to non-root user
USER yao