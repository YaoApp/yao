GO ?= go
GIT ?= git
GOFMT ?= gofmt "-s"
PACKAGES ?= $(shell $(GO) list ./...)
VETPACKAGES ?= $(shell $(GO) list ./... | grep -v /examples/)
GOFILES := $(shell find . -name "*.go")
VERSION := $(shell grep 'const VERSION =' share/const.go |awk '{print $$4}' |sed 's/\"//g')
COMMIT := $(shell git log | head -n 1 | awk '{print substr($$2, 0, 12)}')
NOW := $(shell date +"%FT%T%z")

# ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TESTFOLDER := $(shell $(GO) list ./... | grep -vE 'examples|tests*|config|widgets')
TESTTAGS ?= ""

TESTWIDGETS := $(shell $(GO) list ./widgets/...)

# Unit Test
.PHONY: test
test:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER); do \
		$(GO) test -tags $(TESTTAGS) -v -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") $$d > tmp.out; \
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

# Unit Test
.PHONY: test-widgets
test-widgets:
	echo "mode: count" > coverage.out
	for d in $(TESTWIDGETS); do \
		$(GO) test -tags $(TESTTAGS) -v -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") $$d > tmp.out; \
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
	$(GO) env -w GONOPROXY=github.com/yaoapp/gou
	$(GO) env -w GOPRIVATE=github.com/yaoapp/gou
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
	mkdir -p .tmp/data
	cp -r ui .tmp/data/
	cp -r xgen .tmp/data/
	cp -r yao .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data

# make artifacts-linux
.PHONY: artifacts-linux
artifacts-linux: clean
	mkdir -p dist/release

#	Building XGEN v0.9
	sed -ie "s/url('\/icon/url('\/xiang\/icon/g" ../xgen-v0.9/public/icon/md_icon.css
	cd ../xgen-v0.9 && npm install && npm run build

#	Building XGEN v1.0
	export NODE_ENV=production
	rm -f ../xgen-v1.0/pnpm-lock.yaml
	echo "BASE=__yao_admin_root" > ../xgen-v1.0/packages/xgen/.env
	cd ../xgen-v1.0 && pnpm install && pnpm run build

#	Packing
	mkdir -p .tmp/data/xgen
	cp -r ./ui .tmp/data/ui
	cp -r ../xgen-v0.9/dist .tmp/data/xgen/v0.9
	cp -r ../xgen-v1.0/packages/xgen/dist .tmp/data/xgen/v1.0
	cp -r yao .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data

#	Replace PRVERSION
	sed -ie "s/const PRVERSION = \"DEV\"/const PRVERSION = \"${COMMIT}-${NOW}\"/g" share/const.go

#   Making artifacts
	mkdir -p dist
	CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=linux GOARCH=amd64 go build -v -o dist/yao-${VERSION}-linux-amd64
	CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ go build -v -o dist/yao-${VERSION}-linux-arm64

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


#	Building XGEN v0.9
	sed -ie "s/url('\/icon/url('\/xiang\/icon/g" ../xgen-v0.9/public/icon/md_icon.css
	cd ../xgen-v0.9 && npm install && npm run build

#	Building XGEN v1.0
	export NODE_ENV=production
	rm -f ../xgen-v1.0/pnpm-lock.yaml
	echo "BASE=__yao_admin_root" > ../xgen-v1.0/packages/xgen/.env
	cd ../xgen-v1.0 && pnpm install && pnpm run build

#	Packing
	mkdir -p .tmp/data/xgen
	cp -r ./ui .tmp/data/ui
	cp -r ../xgen-v0.9/dist .tmp/data/xgen/v0.9
	cp -r ../xgen-v1.0/packages/xgen/dist .tmp/data/xgen/v1.0
	cp -r yao .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data

#	Replace PRVERSION
	sed -ie "s/const PRVERSION = \"DEV\"/const PRVERSION = \"${COMMIT}-${NOW}\"/g" share/const.go

#   Making artifacts
	mkdir -p dist
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -o dist/yao-${VERSION}-darwin-amd64
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -v -o dist/yao-${VERSION}-darwin-arm64

	mkdir -p dist/release
	mv dist/yao-*-* dist/release/
	chmod +x dist/release/yao-*-*
	ls -l dist/release/
	dist/release/yao-${VERSION}-darwin-amd64 version

# 	Reset const 
#	cp -f share/const.goe share/const.go
#	rm -f share/const.goe

.PHONY: debug
debug: clean
	mkdir -p dist/release

#	Packing
	mkdir -p .tmp/data
	cp -r ui .tmp/data/ui
	cp -r yao .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data


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

#	Building XGEN v0.9
	mkdir -p .tmp/xgen/v0.9/dist
	echo "XGEN v0.9" > .tmp/xgen/v0.9/dist/index.html

#	Building XGEN v1.0
	export NODE_ENV=production
	git clone https://github.com/YaoApp/xgen-next.git .tmp/xgen/v1.0
	echo "BASE=__yao_admin_root" > .tmp/xgen/v1.0/packages/xgen/.env
	cd .tmp/xgen/v1.0 && pnpm install && pnpm run build

#	Packing
	mkdir -p .tmp/data/xgen
	cp -r ./ui .tmp/data/ui
	cp -r ./yao .tmp/data/yao
	cp -r .tmp/xgen/v0.9/dist .tmp/data/xgen/v0.9
	cp -r .tmp/xgen/v1.0/packages/xgen/dist .tmp/data/xgen/v1.0
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/xgen

#	Replace PRVERSION
	sed -ie "s/const PRVERSION = \"DEV\"/const PRVERSION = \"${COMMIT}-${NOW}\"/g" share/const.go

#   Making artifacts
	mkdir -p dist
	CGO_ENABLED=1 go build -v -o dist/release/yao
	chmod +x  dist/release/yao

# 	Reset const 
	cp -f share/const.goe share/const.go
	rm share/const.goe

.PHONY: linux-release
linux-release: clean
	mkdir -p dist/release
	mkdir .tmp

#	Building XGEN v0.9
	git clone https://github.com/YaoApp/xgen.git .tmp/xgen/v0.9
	sed -ie "s/url('\/icon/url('\/xiang\/icon/g" .tmp/xgen/v0.9/public/icon/md_icon.css
	cd .tmp/xgen/v0.9 && yarn install && yarn build
	mkdir -p .tmp/xgen/v0.9
	cp -r xgen/v0.9 .tmp/xgen/v0.9/dist

#	Building XGEN v1.0
	export NODE_ENV=production
	git clone https://github.com/YaoApp/xgen-next.git .tmp/xgen/v1.0
	rm -f .tmp/xgen/v1.0/pnpm-lock.yaml
	echo "BASE=__yao_admin_root" > .tmp/xgen/v1.0/packages/xgen/.env
	cd .tmp/xgen/v1.0 && pnpm install && pnpm run build

#	Packing
	mkdir -p .tmp/data/xgen
	cp -r ./ui .tmp/data/ui
	cp -r ./yao .tmp/data/yao
	cp -r .tmp/xgen/v0.9/dist .tmp/data/xgen/v0.9
	cp -r .tmp/xgen/v1.0/packages/xgen/dist .tmp/data/xgen/v1.0
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/xgen

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

# make migrate ( for unit test)
.PHONY: migrate
migrate:
	$(GO) test -tags $(TESTTAGS) -run TestCommandMigrate$
