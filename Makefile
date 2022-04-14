GO ?= go
GOFMT ?= gofmt "-s"
PACKAGES ?= $(shell $(GO) list ./...)
VETPACKAGES ?= $(shell $(GO) list ./... | grep -v /examples/)
GOFILES := $(shell find . -name "*.go")
VERSION := $(shell grep 'const VERSION =' share/const.go |awk '{print $$4}' |sed 's/\"//g')

# ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TESTFOLDER := $(shell $(GO) list ./... | grep -vE 'examples|tests*|config')
TESTTAGS ?= ""

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
	go install golang.org/x/lint/golint; \
	go install github.com/client9/misspell/cmd/misspell;

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
	cp -r yao .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data

# make artifacts-linux
.PHONY: artifacts-linux
artifacts-linux: clean
	mkdir -p dist/release

#	Building UI
	sed -ie "s/url('\/icon/url('\/xiang\/icon/g" ../ui/public/icon/md_icon.css
	cd ../ui && npm install && npm run build

#	Packing
	mkdir -p .tmp/data
	cp -r ../ui/dist .tmp/data/ui
	cp -r yao .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/ui

#   Making artifacts
	mkdir -p dist
	CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=linux GOARCH=amd64 go build -v -o dist/yao-${VERSION}-linux-amd64
	CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ go build -v -o dist/yao-${VERSION}-linux-arm64

	mkdir -p dist/release
	mv dist/yao-*-* dist/release/
	chmod +x dist/release/yao-*-*
	ls -l dist/release/
	dist/release/yao-${VERSION}-linux-amd64 version

# make artifacts-macos
.PHONY: artifacts-macos
artifacts-macos: clean
	mkdir -p dist/release

#	Building UI
	sed -ie "s/url('\/icon/url('\/xiang\/icon/g" ../ui/public/icon/md_icon.css
	cd ../ui && npm install && npm run build

#	Packing
	mkdir -p .tmp/data
	cp -r ../ui/dist .tmp/data/ui
	cp -r yao .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/ui

#   Making artifacts
	mkdir -p dist
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -o dist/yao-${VERSION}-darwin-amd64
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -v -o dist/yao-${VERSION}-darwin-arm64

	mkdir -p dist/release
	mv dist/yao-*-* dist/release/
	chmod +x dist/release/yao-*-*
	ls -l dist/release/
	dist/release/yao-${VERSION}-darwin-amd64 version

.PHONY: debug
debug: clean
	mkdir -p dist/release

#	Packing
	mkdir -p .tmp/data
	cp -r ui .tmp/data/ui
	cp -r yao .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data

#   Making artifacts
	mkdir -p dist
	CGO_ENABLED=1 go build -v -o dist/release/yao-debug
	chmod +x  dist/release/yao-debug

.PHONY: release
release: clean
	mkdir -p dist/release
	mkdir .tmp

#	Building UI
	git clone https://github.com/YaoApp/xgen.git .tmp/ui
	sed -ie "s/url('\/icon/url('\/xiang\/icon/g" .tmp/ui/public/icon/md_icon.css
	cd .tmp/ui && yarn install && yarn build

#	Packing
	mkdir -p .tmp/data
	cp -r .tmp/ui/dist .tmp/data/ui
	cp -r yao .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/ui

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