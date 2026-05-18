#!/bin/bash
set -euo pipefail

echo "=== Stack Verification ==="
PASS=0
FAIL=0

check() {
    local name="$1" cmd="$2"
    if eval "$cmd" >/dev/null 2>&1; then
        echo "  PASS: $name"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $name"
        FAIL=$((FAIL + 1))
    fi
}

# 1. mock-llm health
check "mock-llm healthz" \
    "wget -q -O- http://mock-llm:9999/healthz | grep -q ok"

# 2. mock-llm OpenAI echo (non-stream)
check "mock-llm OpenAI echo" \
    "wget -q -O- --header='Content-Type: application/json' --header='X-Mock-Mode: echo' --header='Authorization: Bearer mock' --post-data='{\"model\":\"test\",\"messages\":[{\"role\":\"user\",\"content\":\"hello\"}],\"stream\":false}' http://mock-llm:9999/v1/chat/completions | grep -q 'echo:'"

# 3. mock-llm Anthropic echo (non-stream)
check "mock-llm Anthropic echo" \
    "wget -q -O- --header='Content-Type: application/json' --header='X-Mock-Mode: echo' --header='x-api-key: mock' --header='anthropic-version: 2023-06-01' --post-data='{\"model\":\"test\",\"messages\":[{\"role\":\"user\",\"content\":\"hello\"}],\"max_tokens\":100,\"stream\":false}' http://mock-llm:9999/v1/messages | grep -q 'echo:'"

# 4. mock-llm fixture (OpenAI)
check "mock-llm fixture openai/simple-chat" \
    "wget -q -O- --header='Content-Type: application/json' --header='X-Mock-Mode: fixture' --header='X-Mock-Fixture: openai/simple-chat' --header='Authorization: Bearer mock' --post-data='{\"model\":\"test\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}],\"stream\":false}' http://mock-llm:9999/v1/chat/completions | grep -q 'gpt-4o-mini'"

# 5. mock-llm fixture (DeepSeek thinking)
check "mock-llm fixture deepseek/thinking" \
    "wget -q -O- --header='Content-Type: application/json' --header='X-Mock-Mode: fixture' --header='X-Mock-Fixture: deepseek/thinking' --header='Authorization: Bearer mock' --post-data='{\"model\":\"test\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}],\"stream\":false}' http://mock-llm:9999/v1/chat/completions | grep -q 'reasoning_content'"

# 6. Tai Docker connectivity (if running)
if wget -q --spider --timeout=2 http://tai-docker:8099/healthz 2>/dev/null; then
    check "tai-docker healthz" "wget -q --spider http://tai-docker:8099/healthz"
else
    echo "  SKIP: tai-docker not running"
fi

# 7. Tai HostExec connectivity (if running)
if wget -q --spider --timeout=2 http://tai-hostexec:8101/healthz 2>/dev/null; then
    check "tai-hostexec healthz" "wget -q --spider http://tai-hostexec:8101/healthz"
else
    echo "  SKIP: tai-hostexec not running"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] || exit 1
