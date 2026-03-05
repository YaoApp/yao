#!/bin/bash
# Build script for Sandbox V2 Docker images (base + test)
# Usage: ./build.sh [true|false]  — push to registry or build locally

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUSH=${1:-false}
REGISTRY=${REGISTRY:-"yaoapp"}
YAO_ROOT="$SCRIPT_DIR/../../.."

echo "=== Building Sandbox V2 Images ==="
echo "Push: $PUSH"
echo "Registry: $REGISTRY"

# --- Cross-compile Go binaries ---

echo ""
echo "=== Building yao-grpc (multi-arch) ==="
cd "$YAO_ROOT/tai/grpc/cmd"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$SCRIPT_DIR/base/yao-grpc-amd64" .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "$SCRIPT_DIR/base/yao-grpc-arm64" .
echo "Built: yao-grpc-amd64, yao-grpc-arm64"

echo ""
echo "=== Building claude-proxy (multi-arch) ==="
cd "$YAO_ROOT/sandbox/proxy/cmd/claude-proxy"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$SCRIPT_DIR/base/claude-proxy-amd64" .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "$SCRIPT_DIR/base/claude-proxy-arm64" .
echo "Built: claude-proxy-amd64, claude-proxy-arm64"

cd "$SCRIPT_DIR"

# --- Setup buildx ---

BUILDER_NAME="yao-multiarch"
if ! docker buildx inspect "$BUILDER_NAME" > /dev/null 2>&1; then
    echo "Creating buildx builder: $BUILDER_NAME"
    docker buildx create --name "$BUILDER_NAME" --use --bootstrap
else
    docker buildx use "$BUILDER_NAME"
fi

build_image() {
    local IMAGE_NAME=$1
    local CONTEXT_DIR=$2
    local PUSH_FLAG=$3

    echo ""
    echo "=== Building $IMAGE_NAME (linux/amd64,linux/arm64) ==="

    local BUILD_ARGS="--platform linux/amd64,linux/arm64 -t ${REGISTRY}/${IMAGE_NAME}:latest"

    if [ "$PUSH_FLAG" = "true" ]; then
        BUILD_ARGS="$BUILD_ARGS --push"
    else
        echo "Note: Multi-arch build without push. Building for current platform only."
        BUILD_ARGS="--load -t ${REGISTRY}/${IMAGE_NAME}:latest"
    fi

    docker buildx build $BUILD_ARGS -f "$CONTEXT_DIR/Dockerfile" "$CONTEXT_DIR"
}

# --- Build images ---

build_image "sandbox-v2-base" "$SCRIPT_DIR/base" "$PUSH"
build_image "sandbox-v2-test" "$SCRIPT_DIR/test" "$PUSH"

# --- Cleanup binaries ---

echo ""
echo "=== Cleanup ==="
rm -f "$SCRIPT_DIR/base/yao-grpc-amd64" "$SCRIPT_DIR/base/yao-grpc-arm64"
rm -f "$SCRIPT_DIR/base/claude-proxy-amd64" "$SCRIPT_DIR/base/claude-proxy-arm64"
echo "Removed temporary binary files"

echo ""
echo "=== Build complete ==="
docker images | grep -E "sandbox-v2" | head -10 || true
