#!/bin/bash
# V2 base entrypoint — conditionally starts yao-grpc and openai-proxy

if [ -n "$YAO_GRPC_ADDR" ] && [ -n "$YAO_SANDBOX_ID" ]; then
    tail -f /dev/null | yao-grpc serve &
fi

if [ -n "$OPENAI_PROXY_BACKEND" ]; then
    openai-proxy &
fi

exec "$@"
