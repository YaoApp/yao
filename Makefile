GO ?= go
GOFMT ?= gofmt "-s"
PACKAGES ?= $(shell $(GO) list ./...)
VETPACKAGES ?= $(shell $(GO) list ./... | grep -v /examples/)
GOFILES := $(shell find . -name "*.go")

# ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TESTFOLDER := $(shell $(GO) list ./... | grep -E 'xiang$$|global$$|table$$|user$$' | grep -v examples)
TESTTAGS ?= ""

# 运行单元测试
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

# 编译测试用插件
.PHONY: plugin
plugin: 
	rm -rf $(HOME)/data/gou-unit/plugins
	rm -rf $(HOME)/data/gou-unit/logs
	mkdir -p $(HOME)/data/gou-unit/plugins
	mkdir -p $(HOME)/data/gou-unit/logs
	GOOS=linux GOARCH=amd64 go build -o $(HOME)/data/gou-unit/plugins/user.so ./app/plugins/user
	chmod +x $(HOME)/data/gou-unit/plugins/user.so
	ls -l $(HOME)/data/gou-unit/plugins
	ls -l $(HOME)/data/gou-unit/logs
	$(HOME)/data/gou-unit/plugins/user.so 2>&1 || true
plugin-mac: 
	rm -rf ./app/plugins/user/dist
	rm -rf ./app/plugins/user.so
	go build -o ./app/plugins/user.so ./app/plugins/user
	chmod +x ./app/plugins/user.so

# 编译静态文件
.PHONY: static
static:
	git clone https://github.com/YaoApp/xiang-saas-ui-kxy .tmp/ui
	cd .tmp/ui && yarn install && yarn build
	rm -rf ui
	mv .tmp/ui/dist ui
	rm -rf .tmp/ui

# 将静态文件打包到命令工具
.PHONY: bindata
bindata:
	rm -f data.go
	mkdir -p .tmp/data
	cp -r ui .tmp/data/
	cp -r xiang .tmp/data/
	go-bindata -fs -pkg global -o global/data.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data

# 编译可执行文件
.PHONY: xiang
xiang: bindata

	if [ -f global/bindata.go ]; then \
		mv global/bindata.go global/bindata.go.bak; \
	fi;
	
	if [ ! -z "${XIANG_DOMAIN}" ]; then \
		mv global/vars.go global/vars.go.bak;	\
		sed "s/*.iqka.com/$(XIANG_DOMAIN)/g" global/vars.go.bak > global/vars.go; \
	fi;

	GOOS=linux GOARCH=amd64 go build -v -o .tmp/xiang-linux-amd64
	GOOS=linux GOARCH=arm GOARM=7 go build -v -o .tmp/xiang-linux-arm
	GOOS=linux GOARCH=arm64 GOARM=7 go build -v -o .tmp/xiang-linux-arm64
	GOOS=darwin GOARCH=amd64 go build -v -o .tmp/xiang-darwin-amd64
	mkdir -p dist/bin
	mv .tmp/xiang-*-* dist/bin/
	chmod +x dist/bin/xiang-*-*
	rm -f global/data.go
	if [ -f global/bindata.go.bak ]; then \
		mv global/bindata.go.bak global/bindata.go; \
	fi;
	
	if [ ! -z "${XIANG_DOMAIN}" ]; then \
		rm global/vars.go; \
		mv global/vars.go.bak global/vars.go; \
	fi;

.PHONY: clean
clean: 
	rm -rf ./tmp
	rm -rf dist


.PHONY: migrate
migrate:
	$(GO) test -tags $(TESTTAGS) -run TestCommandMigrate$