#!/bin/bash
# Build script for Yao sandbox Docker images
# Supports multi-architecture builds (amd64 and arm64)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL=${1:-claude}
PUSH=${2:-false}
REGISTRY=${REGISTRY:-"yaoapp"}  # Docker Hub registry

echo "=== Building Yao Sandbox Images ==="
echo "Tool: $TOOL"
echo "Push: $PUSH"
echo "Registry: $REGISTRY"
echo "Script dir: $SCRIPT_DIR"

# Build yao-bridge for both architectures
echo ""
echo "=== Building yao-bridge (multi-arch) ==="
cd "$SCRIPT_DIR/../bridge"

echo "Building for linux/amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$SCRIPT_DIR/yao-bridge-amd64" .

echo "Building for linux/arm64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "$SCRIPT_DIR/yao-bridge-arm64" .

echo "Built: yao-bridge-amd64, yao-bridge-arm64"

cd "$SCRIPT_DIR"

# Check if buildx is available and set up
setup_buildx() {
    echo ""
    echo "=== Setting up Docker Buildx ==="
    
    # Check if buildx is available
    if ! docker buildx version > /dev/null 2>&1; then
        echo "Error: Docker Buildx is not available. Please install it first."
        exit 1
    fi
    
    # Create/use multi-arch builder
    BUILDER_NAME="yao-multiarch"
    if ! docker buildx inspect "$BUILDER_NAME" > /dev/null 2>&1; then
        echo "Creating buildx builder: $BUILDER_NAME"
        docker buildx create --name "$BUILDER_NAME" --use --bootstrap
    else
        echo "Using existing builder: $BUILDER_NAME"
        docker buildx use "$BUILDER_NAME"
    fi
}

# Build multi-arch image
build_multiarch() {
    local IMAGE_NAME=$1
    local DOCKERFILE=$2
    local PUSH_FLAG=$3
    
    echo ""
    echo "=== Building $IMAGE_NAME (linux/amd64,linux/arm64) ==="
    
    BUILD_ARGS="--platform linux/amd64,linux/arm64 -t ${REGISTRY}/${IMAGE_NAME}:latest"
    
    if [ "$PUSH_FLAG" = "true" ]; then
        BUILD_ARGS="$BUILD_ARGS --push"
    else
        # Load to local Docker (only works for single platform)
        echo "Note: Multi-arch build without push. Building for current platform only."
        BUILD_ARGS="--load -t ${REGISTRY}/${IMAGE_NAME}:latest"
    fi
    
    docker buildx build $BUILD_ARGS -f "$DOCKERFILE" .
}

# Setup buildx for multi-arch builds
setup_buildx

# Build base image
echo ""
echo "=== Building base image ==="
build_multiarch "sandbox-base" "base/Dockerfile.base" "$PUSH"

# Build tool-specific images
case $TOOL in
  claude)
    echo ""
    echo "=== Building Claude images ==="
    build_multiarch "sandbox-claude" "claude/Dockerfile" "$PUSH"
    build_multiarch "sandbox-claude-full" "claude/Dockerfile.full" "$PUSH"
    ;;
  cursor)
    echo ""
    echo "=== Building Cursor images ==="
    build_multiarch "sandbox-cursor" "cursor/Dockerfile" "$PUSH"
    ;;
  all)
    echo ""
    echo "=== Building all images ==="
    # Claude
    build_multiarch "sandbox-claude" "claude/Dockerfile" "$PUSH"
    build_multiarch "sandbox-claude-full" "claude/Dockerfile.full" "$PUSH"
    # Cursor (uncomment when ready)
    # build_multiarch "sandbox-cursor" "cursor/Dockerfile" "$PUSH"
    ;;
  *)
    echo "Unknown tool: $TOOL"
    echo "Usage: $0 [claude|cursor|all] [true|false]"
    echo "  $0 claude        # Build Claude images locally"
    echo "  $0 claude true   # Build and push Claude images"
    echo "  $0 all true      # Build and push all images"
    exit 1
    ;;
esac

echo ""
echo "=== Build complete ==="
echo "Images built for tool: $TOOL"

if [ "$PUSH" = "true" ]; then
    echo ""
    echo "Images pushed to: $REGISTRY"
    echo "  - ${REGISTRY}/sandbox-base:latest"
    case $TOOL in
      claude)
        echo "  - ${REGISTRY}/sandbox-claude:latest"
        echo "  - ${REGISTRY}/sandbox-claude-full:latest"
        ;;
      all)
        echo "  - ${REGISTRY}/sandbox-claude:latest"
        echo "  - ${REGISTRY}/sandbox-claude-full:latest"
        ;;
    esac
fi

# Show local images
docker images | grep -E "(sandbox-base|sandbox-claude|sandbox-cursor)" | head -10 || true

# Cleanup
echo ""
echo "=== Cleanup ==="
rm -f "$SCRIPT_DIR/yao-bridge-amd64" "$SCRIPT_DIR/yao-bridge-arm64"
echo "Removed temporary binary files"
