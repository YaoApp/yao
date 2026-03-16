GO ?= go
GIT ?= git
GOFMT ?= gofmt "-s"
PACKAGES ?= $(shell $(GO) list ./...)
VETPACKAGES ?= $(shell $(GO) list ./... | grep -v /examples/)
GOFILES := $(shell find . -name "*.go")
VERSION := $(shell grep 'const VERSION =' share/const.go |awk '{print $$4}' |sed 's/\"//g')
COMMIT := $(shell git log | head -n 1 | awk '{print substr($$2, 0, 12)}')
NOW := $(shell date +"%FT%T%z")
OS := $(shell uname)

# ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TESTFOLDER := $(shell $(GO) list ./... | grep -vE 'examples|openai|aigc|neo|twilio|share*|registry|agent/sandbox/v2' | awk '!/\/tests\// || /openapi\/tests/' | grep -vE 'openapi/tests/(nodes|sandbox|workspace)')
# Core tests (exclude AI-related: agent, aigc, openai, KB, sandbox, registry, grpc, and integrations which require external services)
TESTFOLDER_CORE := $(shell $(GO) list ./... | grep -vE 'examples|openai|aigc|neo|twilio|share*|agent|kb|sandbox|integrations|registry|tai|grpc' | awk '!/\/tests\// || /openapi\/tests/' | grep -vE 'openapi/tests/(nodes|sandbox|workspace)')
# Agent tests (agent, aigc) - exclude agent/search/handlers/web (requires external API keys), robot packages (tested in robot job), and agent/sandbox/v2 (WIP, has its own job)
TESTFOLDER_AGENT := $(shell $(GO) list ./agent/... ./aigc/... | grep -vE 'agent/search/handlers/web|agent/robot/|agent/sandbox/v2')
# KB tests (kb)
TESTFOLDER_KB := $(shell $(GO) list ./kb/...)
# Robot tests (agent/robot/... packages, excluding events/integrations which require Telegram etc.)
TESTFOLDER_ROBOT := $(shell $(GO) list ./agent/robot/... | grep -vE 'agent/robot/events')
# Sandbox tests (requires Docker) — excludes sandbox/v2 (has its own job)
TESTFOLDER_SANDBOX := $(shell $(GO) list ./sandbox/... | grep -v 'sandbox/v2')
# Tai SDK tests (requires Tai container with Docker socket)
TESTFOLDER_TAI := $(shell $(GO) list ./tai/...)
# Workspace tests (requires Tai for remote mode)
TESTFOLDER_WORKSPACE := $(shell $(GO) list ./workspace/...)
# gRPC tests
TESTFOLDER_GRPC := $(shell $(GO) list ./grpc/...)
TESTTAGS ?= ""

# TESTWIDGETS := $(shell $(GO) list ./widgets/...)

# Unit Test (all tests)
.PHONY: unit-test
unit-test:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER); do \
		$(GO) test -tags $(TESTTAGS) -v -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") -skip='TestMemoryLeak|TestIsolateDisposal|TestLeak_|TestScenario_' $$d > tmp.out; \
		cat tmp.out; \
		if grep -q "^--- FAIL" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "build failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "setup failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "runtime error" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		fi; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done

# Core Unit Test (exclude AI-related tests)
.PHONY: unit-test-core
unit-test-core:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER_CORE); do \
		$(GO) test -tags $(TESTTAGS) -v -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") -skip='TestMemoryLeak|TestIsolateDisposal|TestLeak_|TestScenario_' $$d > tmp.out; \
		cat tmp.out; \
		if grep -q "^--- FAIL" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "build failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "setup failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "runtime error" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		fi; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done

# Agent Unit Test (agent, aigc) - excludes robot packages (tested in unit-test-robot) and TestE2E*
.PHONY: unit-test-agent
unit-test-agent:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER_AGENT); do \
		$(GO) test -tags $(TESTTAGS) -v -timeout=50m -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") -skip='TestMemoryLeak|TestIsolateDisposal|TestE2E' $$d > tmp.out; \
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
		elif grep -q "setup failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "runtime error" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		fi; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done

# KB Unit Test (kb)
.PHONY: unit-test-kb
unit-test-kb:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER_KB); do \
		$(GO) test -tags $(TESTTAGS) -v -timeout=20m -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") -skip='TestMemoryLeak|TestIsolateDisposal|TestSearchCleanup' $$d > tmp.out; \
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
		elif grep -q "setup failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "runtime error" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		fi; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done

# Robot Test (all agent/robot/... packages) - runs ALL tests (unit + E2E) with real LLM calls
# These tests require: LLM API keys, database, and longer timeout
.PHONY: unit-test-robot
unit-test-robot:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER_ROBOT); do \
		$(GO) test -tags $(TESTTAGS) -v -timeout=50m -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") -skip='TestMemoryLeak|TestIsolateDisposal' $$d > tmp.out; \
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
		elif grep -q "setup failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "runtime error" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		fi; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done

# Registry Client Test (requires Yao Registry service)
.PHONY: unit-test-registry
unit-test-registry:
	echo "mode: count" > coverage.out
	$(GO) test -v -p 1 -timeout=5m -covermode=count -coverprofile=profile.out ./registry/... > tmp.out; \
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
	fi; \
	if [ -f profile.out ]; then \
		cat profile.out | grep -v "mode:" >> coverage.out; \
		rm profile.out; \
	fi

# ---------------------------------------------------------------------------
# Sandbox V2 CI Test (tai SDK + workspace only)
# Full sandbox/v2 integration tests (multi-pool, K8s, etc.) are run locally.
# ---------------------------------------------------------------------------

.PHONY: unit-test-sandbox-v2
unit-test-sandbox-v2: unit-test-tai unit-test-workspace
	@echo ""
	@echo "============================================="
	@echo "All Sandbox V2 CI tests passed (tai + workspace)"
	@echo "============================================="

# Workspace Unit Test (requires Tai for remote mode)
.PHONY: unit-test-workspace
unit-test-workspace:
	@echo ""
	@echo "============================================="
	@echo "Running Workspace Tests..."
	@echo "============================================="
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER_WORKSPACE); do \
		$(GO) test -tags $(TESTTAGS) -v -timeout=10m -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") $$d > tmp.out; \
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
		elif grep -q "setup failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "runtime error" tmp.out; then \
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
	@echo "All workspace tests passed"
	@echo "============================================="

# Sandbox Unit Test (requires Docker)
.PHONY: unit-test-sandbox
unit-test-sandbox:
	@echo ""
	@echo "============================================="
	@echo "Running Sandbox Tests (requires Docker)..."
	@echo "============================================="
	@echo "Pulling sandbox test images..."
	docker pull alpine:latest || true
	docker pull yaoapp/sandbox-base:latest || true
	docker pull yaoapp/sandbox-claude:latest || true
	@echo ""
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER_SANDBOX); do \
		$(GO) test -tags $(TESTTAGS) -v -timeout=10m -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") -skip='TestMemoryLeak|TestIsolateDisposal' $$d > tmp.out; \
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
		elif grep -q "setup failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "runtime error" tmp.out; then \
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
	@echo "✅ All sandbox tests passed"
	@echo "============================================="

# Tai SDK Test (requires Tai container with Docker socket)
.PHONY: unit-test-tai
unit-test-tai:
	@echo ""
	@echo "============================================="
	@echo "Running Tai SDK Tests (requires Tai container)..."
	@echo "============================================="
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER_TAI); do \
		$(GO) test -tags $(TESTTAGS) -v -timeout=5m -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") $$d > tmp.out; \
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
		elif grep -q "setup failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "runtime error" tmp.out; then \
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
	@echo "All Tai SDK tests passed"
	@echo "============================================="

# Proto codegen
.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		grpc/pb/yao.proto

# gRPC Unit Test
.PHONY: unit-test-grpc
unit-test-grpc:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER_GRPC); do \
		$(GO) test -tags $(TESTTAGS) -v -timeout=10m \
			-covermode=count -coverprofile=profile.out \
			-coverpkg=$$(echo $$d | sed "s/\/test$$//g") \
			-skip='TestMemoryLeak|TestIsolateDisposal|TestLeak_|TestScenario_' \
			$$d > tmp.out; \
		cat tmp.out; \
		if grep -q "^--- FAIL" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "build failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "setup failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "runtime error" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		fi; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done

# Benchmark Test
.PHONY: benchmark
benchmark:
	@echo ""
	@echo "============================================="
	@echo "Running Benchmark Tests (agent, trace, event)..."
	@echo "============================================="
	@for d in $$($(GO) list ./agent/... ./trace/... ./event/...); do \
		if $(GO) test -list=Benchmark $$d 2>/dev/null | grep -q "^Benchmark"; then \
			echo ""; \
			echo "📊 Benchmarking: $$d"; \
			echo "---------------------------------------------"; \
			$(GO) test -bench=. -benchmem -benchtime=100x -run='^$$' $$d || true; \
		fi; \
	done
	@echo ""
	@echo "============================================="
	@echo "✅ All benchmarks completed"
	@echo "============================================="

# Memory Leak Detection Test
.PHONY: memory-leak
memory-leak:
	@echo ""
	@echo "============================================="
	@echo "Running Memory Leak Detection (agent, trace, event)..."
	@echo "============================================="
	@for d in $$($(GO) list ./agent/... ./trace/... ./event/...); do \
		if $(GO) test -list='TestMemoryLeak|TestIsolateDisposal|TestGoroutineLeak|TestLeak_|TestScenario_' $$d 2>/dev/null | grep -qE "^Test(MemoryLeak|IsolateDisposal|GoroutineLeak|Leak_|Scenario_)"; then \
			echo ""; \
			echo "🔍 Memory Leak Detection: $$d"; \
			echo "---------------------------------------------"; \
			$(GO) test -run='TestMemoryLeak|TestIsolateDisposal|TestGoroutineLeak|TestLeak_|TestScenario_' -v -timeout=5m $$d || exit 1; \
		fi; \
	done
	@echo ""
	@echo "============================================="
	@echo "✅ All memory leak tests passed"
	@echo "============================================="

# Run all tests (unit + benchmark + memory leak)
.PHONY: test
test: unit-test benchmark memory-leak

.PHONY: fmt
fmt:
	$(GOFMT) -w $(GOFILES)

.PHONY: fmt-check
fmt-check:
	@diff=$$($(GOFMT) -d $(GOFILES)); \
	if [ -n "$$diff" ]; then \
		echo "Please run 'make fmt' and commit the result:"; \
		echo "$${diff}"; \
		exit 1; \
	fi;

vet:
	$(GO) vet $(VETPACKAGES)

.PHONY: lint
lint:
	@hash golint > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u golang.org/x/lint/golint; \
	fi
	for PKG in $(PACKAGES); do golint -set_exit_status $$PKG || exit 1; done;

.PHONY: misspell-check
misspell-check:
	@hash misspell > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/client9/misspell/cmd/misspell; \
	fi
	misspell -error $(GOFILES)

.PHONY: misspell
misspell:
	@hash misspell > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/client9/misspell/cmd/misspell; \
	fi
	misspell -w $(GOFILES)

.PHONY: tools
tools:
	go install golang.org/x/lint/golint@latest; \
	go install github.com/client9/misspell/cmd/misspell@latest; \
	go install github.com/go-bindata/go-bindata/...@latest;
	
# make plugin
.PHONY: plugin
plugin: 
	rm -rf $(HOME)/data/gou-unit/plugins
	rm -rf $(HOME)/data/gou-unit/logs
	mkdir -p $(HOME)/data/gou-unit/plugins
	mkdir -p $(HOME)/data/gou-unit/logs
	GOOS=linux GOARCH=amd64 go build -o $(HOME)/data/gou-unit/plugins/user.so ./tests/plugins/user
	chmod +x $(HOME)/data/gou-unit/plugins/user.so
	ls -l $(HOME)/data/gou-unit/plugins
	ls -l $(HOME)/data/gou-unit/logs
	$(HOME)/data/gou-unit/plugins/user.so 2>&1 || true

# make plugin-mac
.PHONY: plugin-mac
plugin-mac: 
	rm -rf ./tests/plugins/user/dist
	rm -rf ./tests/plugins/user.so
	go build -o ./tests/plugins/user.so ./tests/plugins/user
	chmod +x ./tests/plugins/user.so


# make pack
.PHONY: pack 
pack: bindata fmt

.PHONY: bindata
bindata:

#   Setup Workdir
	rm -rf .tmp/data
	rm -rf .tmp/yao-init
	mkdir -p .tmp/data

#	Checkout init
	git clone https://github.com/YaoApp/yao-init.git .tmp/yao-init
	rm -rf .tmp/yao-init/.git
	rm -rf .tmp/yao-init/.gitignore
	rm -rf .tmp/yao-init/LICENSE
#	rm -rf .tmp/yao-init/README.md
	
#	Copy Files
	cp -r .tmp/yao-init .tmp/data/init
	cp -r ui .tmp/data/
	cp -r ui .tmp/data/public
	cp -r cui .tmp/data/
	cp -r yao .tmp/data/
	cp -r sui/libsui .tmp/data/
	find .tmp/data -name ".DS_Store" -type f -delete
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/yao-init

# make artifacts-linux
.PHONY: artifacts-linux
artifacts-linux: clean
	mkdir -p dist/release

#	Building CUI v1.0
	export NODE_ENV=production
# 	rm -f ../cui-v1.0/pnpm-lock.yaml
	echo "BASE=__yao_admin_root" > ../cui-v1.0/packages/cui/.env
	cd ../cui-v1.0 && pnpm install --no-frozen-lockfile && pnpm run build

#	Init Application
	cd ../yao-init && rm -rf .git
	cd ../yao-init && rm -rf .gitignore
	cd ../yao-init && rm -rf LICENSE
#	cd ../yao-init rm -rf README.md

#	Switch .env login URLs from dev mode (__yao_admin_root) to release mode (dashboard)
	sed -i.bak 's|AFTER_LOGIN_SUCCESS_URL="/__yao_admin_root/|# AFTER_LOGIN_SUCCESS_URL="/__yao_admin_root/|g' ../yao-init/.env
	sed -i.bak 's|AFTER_LOGIN_FAILURE_URL="/__yao_admin_root/|# AFTER_LOGIN_FAILURE_URL="/__yao_admin_root/|g' ../yao-init/.env
	sed -i.bak 's|# AFTER_LOGIN_SUCCESS_URL="/dashboard/|AFTER_LOGIN_SUCCESS_URL="/dashboard/|g' ../yao-init/.env
	sed -i.bak 's|# AFTER_LOGIN_FAILURE_URL="/dashboard/|AFTER_LOGIN_FAILURE_URL="/dashboard/|g' ../yao-init/.env
	rm -f ../yao-init/.env.bak

#   Yao Builder
#   Remove Yao Builder - DUI PageBuilder component will provide online design for pure HTML pages or SUI pages in the future.
#	mkdir -p .tmp/data/builder
#	curl -o .tmp/yao-builder-latest.tar.gz https://release-sv.yaoapps.com/archives/yao-builder-latest.tar.gz
#	tar -zxvf .tmp/yao-builder-latest.tar.gz -C .tmp/data/builder
#	rm -rf .tmp/yao-builder-latest.tar.gz

#	Packing
#   ** CUI will be renamed to CUI in the feature. and move to the new repository. **
#   ** new repository: https://github.com/YaoApp/cui.git **
	mkdir -p .tmp/data/cui
	cp -r ./ui .tmp/data/ui
	cp -r ../cui-v1.0/packages/cui/dist .tmp/data/cui/v1.0
	cp -r ../yao-init .tmp/data/init
	cp -r yao .tmp/data/
	cp -r sui/libsui .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data

#	Replace PRVERSION
	sed -ie "s/const PRVERSION = \"DEV\"/const PRVERSION = \"${COMMIT}-${NOW}\"/g" share/const.go
	@CUI_COMMIT=$$(cd ../cui-v1.0 && git log | head -n 1 | awk '{print substr($$2, 0, 12)}') && \
	sed -ie "s/const PRCUI = \"DEV\"/const PRCUI = \"$$CUI_COMMIT-${NOW}\"/g" share/const.go

#   Making artifacts - dev builds (full debug symbols, ~158M)
	mkdir -p dist
	CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=linux GOARCH=amd64 go build -v -o dist/yao-${VERSION}-linux-amd64
	CGO_ENABLED=1 CGO_LDFLAGS="-static" LD_LIBRARY_PATH=/usr/lib/gcc-cross/aarch64-linux-gnu/13 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc-13 CXX=aarch64-linux-gnu-g++-13 go build -v -o dist/yao-${VERSION}-linux-arm64

#   Making artifacts - prod builds (stripped, ~111M)
	sed -i.tmp 's/const BUILDOPTIONS = ""/const BUILDOPTIONS = "-s -w (production, stripped)"/g' share/const.go && rm -f share/const.go.tmp
	CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=linux GOARCH=amd64 go build -v -ldflags="-s -w" -o dist/yao-${VERSION}-linux-amd64-prod
	CGO_ENABLED=1 CGO_LDFLAGS="-static" LD_LIBRARY_PATH=/usr/lib/gcc-cross/aarch64-linux-gnu/13 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc-13 CXX=aarch64-linux-gnu-g++-13 go build -v -ldflags="-s -w" -o dist/yao-${VERSION}-linux-arm64-prod

	mkdir -p dist/release
	mv dist/yao-*-* dist/release/
	chmod +x dist/release/yao-*-*
	ls -l dist/release/
	dist/release/yao-${VERSION}-linux-amd64 version

# 	Reset const 
#	cp -f share/const.goe share/const.go
#	rm -f share/const.goe

# make artifacts-macos
.PHONY: artifacts-macos
artifacts-macos: clean

	mkdir -p dist/release

#	Building CUI v1.0
	export NODE_ENV=production
#   rm -f ../cui-v1.0/pnpm-lock.yaml
	echo "BASE=__yao_admin_root" > ../cui-v1.0/packages/cui/.env
	cd ../cui-v1.0 && pnpm install --no-frozen-lockfile && pnpm run build

#	Init Application
	cd ../yao-init && rm -rf .git
	cd ../yao-init && rm -rf .gitignore
	cd ../yao-init && rm -rf LICENSE
#	 cd ../yao-init && rm -rf README.md

#	Switch .env login URLs from dev mode (__yao_admin_root) to release mode (dashboard)
	sed -i.bak 's|AFTER_LOGIN_SUCCESS_URL="/__yao_admin_root/|# AFTER_LOGIN_SUCCESS_URL="/__yao_admin_root/|g' ../yao-init/.env
	sed -i.bak 's|AFTER_LOGIN_FAILURE_URL="/__yao_admin_root/|# AFTER_LOGIN_FAILURE_URL="/__yao_admin_root/|g' ../yao-init/.env
	sed -i.bak 's|# AFTER_LOGIN_SUCCESS_URL="/dashboard/|AFTER_LOGIN_SUCCESS_URL="/dashboard/|g' ../yao-init/.env
	sed -i.bak 's|# AFTER_LOGIN_FAILURE_URL="/dashboard/|AFTER_LOGIN_FAILURE_URL="/dashboard/|g' ../yao-init/.env
	rm -f ../yao-init/.env.bak

#	Packing
	mkdir -p .tmp/data/cui
	cp -r ./ui .tmp/data/ui
	cp -r ../cui-v1.0/packages/cui/dist .tmp/data/cui/v1.0
	cp -r ../yao-init .tmp/data/init
	cp -r yao .tmp/data/
	cp -r sui/libsui .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data

#	Replace PRVERSION
	sed -ie "s/const PRVERSION = \"DEV\"/const PRVERSION = \"${COMMIT}-${NOW}\"/g" share/const.go
	@CUI_COMMIT=$$(cd ../cui-v1.0 && git log | head -n 1 | awk '{print substr($$2, 0, 12)}') && \
	sed -ie "s/const PRCUI = \"DEV\"/const PRCUI = \"$$CUI_COMMIT-${NOW}\"/g" share/const.go

#   Making artifacts - dev builds (full debug symbols)
	mkdir -p dist
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -o dist/yao-${VERSION}-darwin-amd64
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -v -o dist/yao-${VERSION}-darwin-arm64

#   Making artifacts - prod builds (stripped, no UPX on macOS)
	sed -i.tmp 's/const BUILDOPTIONS = ""/const BUILDOPTIONS = "-s -w (production, stripped)"/g' share/const.go && rm -f share/const.go.tmp
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -ldflags="-s -w" -o dist/yao-${VERSION}-darwin-amd64-prod
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -v -ldflags="-s -w" -o dist/yao-${VERSION}-darwin-arm64-prod

	mkdir -p dist/release
	mv dist/yao-*-* dist/release/
	chmod +x dist/release/yao-*-*
	ls -l dist/release/
	dist/release/yao-${VERSION}-darwin-amd64 version


.PHONY: debug
debug: clean
	mkdir -p dist/release

#	Packing
#	mkdir -p .tmp/data
#	cp -r ui .tmp/data/ui
#	cp -r yao .tmp/data/
#	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
#	rm -rf .tmp/data


#	Replace PRVERSION
	sed -ie "s/const PRVERSION = \"DEV\"/const PRVERSION = \"${COMMIT}-${NOW}-debug\"/g" share/const.go

#   Making artifacts
	mkdir -p dist
	CGO_ENABLED=1 go build -v -o dist/release/yao-debug
	chmod +x  dist/release/yao-debug

# 	Reset const 
	cp -f share/const.goe share/const.go
	rm -f share/const.goe

# make prepare (build CUI, yao-init, bindata - shared by release and prod)
.PHONY: prepare
prepare: clean
	mkdir -p dist/release
	mkdir .tmp

#	Building CUI v0.9
	mkdir -p .tmp/cui/v0.9/dist
	echo "CUI v0.9" > .tmp/cui/v0.9/dist/index.html

#	Building CUI v1.0
#   ** CUI will be renamed to CUI in the feature. and move to the new repository. **
#   ** new repository: https://github.com/YaoApp/cui.git **
	export NODE_ENV=production
	git clone https://github.com/YaoApp/cui.git .tmp/cui/v1.0
# 	cd .tmp/cui/v1.0 && git checkout 5002c3fded585aaa69a4366135b415ea3234964e
	echo "BASE=__yao_admin_root" > .tmp/cui/v1.0/packages/cui/.env
	cd .tmp/cui/v1.0 && pnpm install --no-frozen-lockfile && pnpm run build
	CUI_COMMIT=$$(cd .tmp/cui/v1.0 && git rev-parse --short HEAD)

#	Checkout init
	git clone https://github.com/YaoApp/yao-init.git .tmp/yao-init
	rm -rf .tmp/yao-init/.git
	rm -rf .tmp/yao-init/.gitignore
	rm -rf .tmp/yao-init/LICENSE
	rm -rf .tmp/yao-init/README.md

#	Switch .env login URLs from dev mode (__yao_admin_root) to release mode (dashboard)
	sed -i.bak 's|AFTER_LOGIN_SUCCESS_URL="/__yao_admin_root/|# AFTER_LOGIN_SUCCESS_URL="/__yao_admin_root/|g' .tmp/yao-init/.env
	sed -i.bak 's|AFTER_LOGIN_FAILURE_URL="/__yao_admin_root/|# AFTER_LOGIN_FAILURE_URL="/__yao_admin_root/|g' .tmp/yao-init/.env
	sed -i.bak 's|# AFTER_LOGIN_SUCCESS_URL="/dashboard/|AFTER_LOGIN_SUCCESS_URL="/dashboard/|g' .tmp/yao-init/.env
	sed -i.bak 's|# AFTER_LOGIN_FAILURE_URL="/dashboard/|AFTER_LOGIN_FAILURE_URL="/dashboard/|g' .tmp/yao-init/.env
	rm -f .tmp/yao-init/.env.bak

#   Yao Builder
#   Remove Yao Builder - DUI PageBuilder component will provide online design for pure HTML pages or SUI pages in the future.
#	mkdir -p .tmp/data/builder
#	curl -o .tmp/yao-builder-latest.tar.gz https://release-sv.yaoapps.com/archives/yao-builder-latest.tar.gz
#	tar -zxvf .tmp/yao-builder-latest.tar.gz -C .tmp/data/builder
#	rm -rf .tmp/yao-builder-latest.tar.gz

#	Packing
	cp -f data/bindata.go data/bindata.go.bak
	mkdir -p .tmp/data/cui
	cp -r ./ui .tmp/data/ui
	cp -r ./yao .tmp/data/yao
	cp -r ./sui/libsui .tmp/data/libsui
	cp -r .tmp/cui/v0.9/dist .tmp/data/cui/v0.9
	cp -r .tmp/cui/v1.0/packages/cui/dist .tmp/data/cui/v1.0
	cp -r .tmp/yao-init .tmp/data/init
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...

#	Replace PRVERSION
	cp -f share/const.go share/const.go.bak
	sed -ie "s/const PRVERSION = \"DEV\"/const PRVERSION = \"${COMMIT}-${NOW}\"/g" share/const.go
	@CUI_COMMIT=$$(cd .tmp/cui/v1.0 && git log | head -n 1 | awk '{print substr($$2, 0, 12)}') && \
	sed -ie "s/const PRCUI = \"DEV\"/const PRCUI = \"$$CUI_COMMIT-${NOW}\"/g" share/const.go

# make release (development build only, ~158M)
.PHONY: release
release: prepare
#   Making artifacts - dev build
	mkdir -p dist
	CGO_ENABLED=1 go build -v -o dist/release/yao
	chmod +x  dist/release/yao

# 	Clean up and restore bindata.go and const.go
	cp data/bindata.go.bak data/bindata.go
	cp share/const.go.bak share/const.go
	rm data/bindata.go.bak
	rm share/const.go.bak
	rm -rf .tmp

#   MacOS Application Signing
	@if [ "$(OS)" = "Darwin" ]; then \
	    codesign --deep --force --verbose --timestamp --options runtime \
	        --entitlements .github/codesign/entitlements.plist \
	        --sign "${APPLE_SIGN}" dist/release/yao ; \
	fi

# make prod (production build only, ~111M on macOS)
.PHONY: prod
prod: prepare
#	Set BUILDOPTIONS
	@if [ "$$(uname)" = "Linux" ]; then \
		sed -i.tmp 's/const BUILDOPTIONS = ""/const BUILDOPTIONS = "-s -w +upx (production, compressed)"/g' share/const.go && rm -f share/const.go.tmp; \
	else \
		sed -i.tmp 's/const BUILDOPTIONS = ""/const BUILDOPTIONS = "-s -w (production, stripped)"/g' share/const.go && rm -f share/const.go.tmp; \
	fi

#   Making artifacts - prod build
	mkdir -p dist
	CGO_ENABLED=1 go build -v -ldflags="-s -w" -o dist/release/yao-prod
	chmod +x dist/release/yao-prod

#	UPX compression (Linux only)
	@if [ "$$(uname)" = "Linux" ]; then \
		echo "Compressing with UPX..."; \
		if command -v upx > /dev/null 2>&1; then \
			upx --best dist/release/yao-prod; \
		else \
			echo "WARNING: UPX not found. Install with: apt install upx"; \
			echo "Skipping compression."; \
		fi; \
	else \
		echo "Note: UPX compression skipped on macOS (not supported)"; \
	fi

# 	Clean up and restore bindata.go and const.go
	cp data/bindata.go.bak data/bindata.go
	cp share/const.go.bak share/const.go
	rm data/bindata.go.bak
	rm share/const.go.bak
	rm -rf .tmp

#   MacOS Application Signing
	@if [ "$(OS)" = "Darwin" ]; then \
	    codesign --deep --force --verbose --timestamp --options runtime \
	        --entitlements .github/codesign/entitlements.plist \
	        --sign "${APPLE_SIGN}" dist/release/yao-prod ; \
	fi

	@echo ""
	@echo "Done! Production binary:"
	@ls -lh dist/release/yao-prod
	@echo ""
	@echo "Test with: dist/release/yao-prod version --all"

# make release-all (build both dev and prod in one go)
.PHONY: release-all
release-all: prepare
#   Making artifacts - dev build (~158M)
	@echo "Building dev binary..."
	mkdir -p dist
	CGO_ENABLED=1 go build -v -o dist/release/yao
	chmod +x dist/release/yao

#   Making artifacts - prod build (~111M on macOS)
	@echo "Building prod binary..."
	@if [ "$$(uname)" = "Linux" ]; then \
		sed -i.tmp 's/const BUILDOPTIONS = ""/const BUILDOPTIONS = "-s -w +upx (production, compressed)"/g' share/const.go && rm -f share/const.go.tmp; \
	else \
		sed -i.tmp 's/const BUILDOPTIONS = ""/const BUILDOPTIONS = "-s -w (production, stripped)"/g' share/const.go && rm -f share/const.go.tmp; \
	fi
	CGO_ENABLED=1 go build -v -ldflags="-s -w" -o dist/release/yao-prod
	chmod +x dist/release/yao-prod

#	UPX compression (Linux only)
	@if [ "$$(uname)" = "Linux" ]; then \
		echo "Compressing with UPX..."; \
		if command -v upx > /dev/null 2>&1; then \
			upx --best dist/release/yao-prod; \
		else \
			echo "WARNING: UPX not found. Install with: apt install upx"; \
			echo "Skipping compression."; \
		fi; \
	else \
		echo "Note: UPX compression skipped on macOS (not supported)"; \
	fi

# 	Clean up and restore bindata.go and const.go
	cp data/bindata.go.bak data/bindata.go
	cp share/const.go.bak share/const.go
	rm data/bindata.go.bak
	rm share/const.go.bak
	rm -rf .tmp

#   MacOS Application Signing
	@if [ "$(OS)" = "Darwin" ]; then \
	    codesign --deep --force --verbose --timestamp --options runtime \
	        --entitlements .github/codesign/entitlements.plist \
	        --sign "${APPLE_SIGN}" dist/release/yao ; \
	    codesign --deep --force --verbose --timestamp --options runtime \
	        --entitlements .github/codesign/entitlements.plist \
	        --sign "${APPLE_SIGN}" dist/release/yao-prod ; \
	fi

	@echo ""
	@echo "Done! Binaries:"
	@ls -lh dist/release/yao dist/release/yao-prod
	@echo ""
	@echo "Test with:"
	@echo "  dist/release/yao version --all"
	@echo "  dist/release/yao-prod version --all"


.PHONY: linux-release
linux-release: clean
	mkdir -p dist/release
	mkdir .tmp

#	Building CUI v1.0
#   ** CUI will be renamed to CUI in the feature. and move to the new repository. **
#   ** new repository: https://github.com/YaoApp/cui.git **
	export NODE_ENV=production
	git clone https://github.com/YaoApp/cui.git .tmp/cui/v1.0
	rm -f .tmp/cui/v1.0/pnpm-lock.yaml
	echo "BASE=__yao_admin_root" > .tmp/cui/v1.0/packages/cui/.env
	cd .tmp/cui/v1.0 && pnpm install --no-frozen-lockfile && pnpm run build

#   Setup UI
	cd .tmp/cui/v1.0/packages/setup  && pnpm install --no-frozen-lockfile && pnpm run build


#	Checkout init
	git clone https://github.com/YaoApp/yao-init.git .tmp/yao-init
	rm -rf .tmp/yao-init/.git
	rm -rf .tmp/yao-init/.gitignore
	rm -rf .tmp/yao-init/LICENSE
	rm -rf .tmp/yao-init/README.md

#   Yao Builder
#   Remove Yao Builder - DUI PageBuilder component will provide online design for pure HTML pages or SUI pages in the future.
# 	mkdir -p .tmp/data/builder
# 	curl -o .tmp/yao-builder-latest.tar.gz https://release-sv.yaoapps.com/archives/yao-builder-latest.tar.gz
# 	tar -zxvf .tmp/yao-builder-latest.tar.gz -C .tmp/data/builder
# 	rm -rf .tmp/yao-builder-latest.tar.gz

#	Packing
	mkdir -p .tmp/data/cui
	cp -r ./ui .tmp/data/ui
	cp -r ./yao .tmp/data/yao
	cp -r .tmp/cui/v0.9/dist .tmp/data/cui/v0.9
	cp -r .tmp/cui/v1.0/packages/setup/build .tmp/data/cui/setup
	cp -r .tmp/cui/v1.0/packages/cui/dist .tmp/data/cui/v1.0
	cp -r .tmp/yao-init .tmp/data/init
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/cui

#   Making artifacts
	mkdir -p dist
	CGO_ENABLED=1 CGO_LDFLAGS="-static" go build -v -o dist/release/yao
	chmod +x  dist/release/yao

# make clean
.PHONY: clean
clean: 
	rm -rf ./tmp
	rm -rf .tmp
	rm -rf dist