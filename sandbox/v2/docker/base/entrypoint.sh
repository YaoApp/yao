#!/bin/bash
# V2 base entrypoint — conditionally starts yao-grpc and claude-proxy

if [ -n "$YAO_GRPC_ADDR" ] && [ -n "$YAO_SANDBOX_ID" ]; then
    tail -f /dev/null | yao-grpc serve &
fi

if [ -n "$CLAUDE_PROXY_UPSTREAM" ]; then
    claude-proxy &
fi

exec "$@"
