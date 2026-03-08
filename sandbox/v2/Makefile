GO ?= go
GOFILES := $(shell find . -name "*.go" -not -path "./docker/*")
PACKAGES := $(shell $(GO) list ./...)
TEST_IMAGE ?= yaoapp/tai-sandbox-test:latest
TEST_TIMEOUT ?= 600s

# ---------------------------------------------------------------------------
# Local test (Docker only)
# ---------------------------------------------------------------------------
.PHONY: test-local
test-local:
	@echo "=== Sandbox V2: local mode ==="
	SANDBOX_TEST_IMAGE=$(TEST_IMAGE) \
	$(GO) test $(PACKAGES) -count=1 -v -timeout $(TEST_TIMEOUT) -run '/local'

# ---------------------------------------------------------------------------
# Remote test (requires Tai server)
# ---------------------------------------------------------------------------
.PHONY: test-remote
test-remote:
	@if [ -z "$(SANDBOX_TEST_REMOTE_ADDR)" ]; then \
		echo "SANDBOX_TEST_REMOTE_ADDR not set, skipping remote tests"; \
		exit 0; \
	fi
	@echo "=== Sandbox V2: remote mode ($(SANDBOX_TEST_REMOTE_ADDR)) ==="
	SANDBOX_TEST_IMAGE=$(TEST_IMAGE) \
	$(GO) test $(PACKAGES) -count=1 -v -timeout $(TEST_TIMEOUT) -run '/remote'

# ---------------------------------------------------------------------------
# Dual-mode test (local + remote when configured)
# ---------------------------------------------------------------------------
.PHONY: test
test:
	@echo ""
	@echo "============================================="
	@echo "Sandbox V2 Tests (dual-mode)"
	@echo "============================================="
	@echo "Image:  $(TEST_IMAGE)"
	@echo "Remote: $${SANDBOX_TEST_REMOTE_ADDR:-<not set, local only>}"
	@echo ""
	SANDBOX_TEST_IMAGE=$(TEST_IMAGE) \
	$(GO) test $(PACKAGES) -count=1 -v -timeout $(TEST_TIMEOUT)
	@echo ""
	@echo "============================================="
	@echo "Sandbox V2 Tests passed"
	@echo "============================================="

# ---------------------------------------------------------------------------
# CI test (with coverage, used by Makefile at repo root)
# ---------------------------------------------------------------------------
.PHONY: test-ci
test-ci:
	@echo ""
	@echo "============================================="
	@echo "Sandbox V2 CI Tests"
	@echo "============================================="
	echo "mode: count" > coverage.out
	@for d in $(PACKAGES); do \
		SANDBOX_TEST_IMAGE=$(TEST_IMAGE) \
		$(GO) test -v -count=1 -timeout $(TEST_TIMEOUT) \
			-covermode=count -coverprofile=profile.out \
			-coverpkg=$$(echo $$d | sed "s/\/test$$//g") \
			$$d > tmp.out; \
		cat tmp.out; \
		if grep -q "^--- FAIL" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "^FAIL" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "^panic:" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "build failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		fi; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done
	@echo ""
	@echo "============================================="
	@echo "Sandbox V2 CI Tests passed"
	@echo "============================================="

# ---------------------------------------------------------------------------
# Docker images
# ---------------------------------------------------------------------------
.PHONY: docker-build
docker-build:
	./docker/build.sh build

.PHONY: docker-push
docker-push:
	./docker/build.sh push

# ---------------------------------------------------------------------------
# fmt / vet
# ---------------------------------------------------------------------------
.PHONY: fmt
fmt:
	gofmt -s -w $(GOFILES)

.PHONY: vet
vet:
	$(GO) vet $(PACKAGES)
