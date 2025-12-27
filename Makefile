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
TESTFOLDER := $(shell $(GO) list ./... | grep -vE 'examples|openai|aigc|neo|twilio|share*' | awk '!/\/tests\// || /openapi\/tests/')
# Core tests (exclude AI-related: agent, aigc, openai, and KB)
TESTFOLDER_CORE := $(shell $(GO) list ./... | grep -vE 'examples|openai|aigc|neo|twilio|share*|agent|kb' | awk '!/\/tests\// || /openapi\/tests/')
# AI tests (agent, aigc) - exclude agent/search/handlers/web (requires external API keys)
TESTFOLDER_AI := $(shell $(GO) list ./agent/... ./aigc/... | grep -v 'agent/search/handlers/web')
# KB tests (kb)
TESTFOLDER_KB := $(shell $(GO) list ./kb/...)
TESTTAGS ?= ""

# TESTWIDGETS := $(shell $(GO) list ./widgets/...)

# Unit Test (all tests)
.PHONY: unit-test
unit-test:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER); do \
		$(GO) test -tags $(TESTTAGS) -v -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") -skip='TestMemoryLeak|TestIsolateDisposal' $$d > tmp.out; \
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
		$(GO) test -tags $(TESTTAGS) -v -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") -skip='TestMemoryLeak|TestIsolateDisposal' $$d > tmp.out; \
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

# AI Unit Test (agent, aigc)
.PHONY: unit-test-ai
unit-test-ai:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER_AI); do \
		$(GO) test -tags $(TESTTAGS) -v -timeout=20m -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") -skip='TestMemoryLeak|TestIsolateDisposal' $$d > tmp.out; \
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

# Benchmark Test
.PHONY: benchmark
benchmark:
	@echo ""
	@echo "============================================="
	@echo "Running Benchmark Tests (agent & trace)..."
	@echo "============================================="
	@for d in $$($(GO) list ./agent/... ./trace/...); do \
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
	@echo "Running Memory Leak Detection (agent & trace)..."
	@echo "============================================="
	@for d in $$($(GO) list ./agent/... ./trace/...); do \
		if $(GO) test -list='TestMemoryLeak|TestIsolateDisposal|TestGoroutineLeak' $$d 2>/dev/null | grep -qE "^Test(MemoryLeak|IsolateDisposal|GoroutineLeak)"; then \
			echo ""; \
			echo "🔍 Memory Leak Detection: $$d"; \
			echo "---------------------------------------------"; \
			$(GO) test -run='TestMemoryLeak|TestIsolateDisposal|TestGoroutineLeak' -v -timeout=5m $$d || exit 1; \
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

#   Making artifacts
	mkdir -p dist
	CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=linux GOARCH=amd64 go build -v -o dist/yao-${VERSION}-unstable-linux-amd64
	CGO_ENABLED=1 CGO_LDFLAGS="-static" LD_LIBRARY_PATH=/usr/lib/gcc-cross/aarch64-linux-gnu/13 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc-13 CXX=aarch64-linux-gnu-g++-13 go build -v -o dist/yao-${VERSION}-unstable-linux-arm64

	mkdir -p dist/release
	mv dist/yao-*-* dist/release/
	chmod +x dist/release/yao-*-*
	ls -l dist/release/
	dist/release/yao-${VERSION}-unstable-linux-amd64 version

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

#   Making artifacts
	mkdir -p dist
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -o dist/yao-${VERSION}-dev-darwin-amd64
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -v -o dist/yao-${VERSION}-dev-darwin-arm64

	mkdir -p dist/release
	mv dist/yao-*-* dist/release/
	chmod +x dist/release/yao-*-*
	ls -l dist/release/
	dist/release/yao-${VERSION}-dev-darwin-amd64 version


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

.PHONY: release
release: clean
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

#   Making artifacts
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
	    codesign --deep --force --verify --verbose --sign "${APPLE_SIGN}" dist/release/yao ; \
	fi


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