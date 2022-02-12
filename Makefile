GO ?= go
GOFMT ?= gofmt "-s"
PACKAGES ?= $(shell $(GO) list ./...)
VETPACKAGES ?= $(shell $(GO) list ./... | grep -v /examples/)
GOFILES := $(shell find . -name "*.go")
VERSION := $(shell grep 'const VERSION =' share/const.go |awk '{print $$4}' |sed 's/\"//g')

# ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TESTFOLDER := $(shell $(GO) list ./... | grep -vE 'examples|tests*|config')
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
	GOOS=linux GOARCH=amd64 go build -o $(HOME)/data/gou-unit/plugins/user.so ./tests/plugins/user
	chmod +x $(HOME)/data/gou-unit/plugins/user.so
	ls -l $(HOME)/data/gou-unit/plugins
	ls -l $(HOME)/data/gou-unit/logs
	$(HOME)/data/gou-unit/plugins/user.so 2>&1 || true
plugin-mac: 
	rm -rf ./tests/plugins/user/dist
	rm -rf ./tests/plugins/user.so
	go build -o ./tests/plugins/user.so ./tests/plugins/user
	chmod +x ./tests/plugins/user.so

# 编译静态文件
.PHONY: static
static:
	git clone git@github.com:YaoApp/xiang-saas-ui-kxy .tmp/ui
	cd .tmp/ui && yarn install && yarn build
	rm -rf ui
	mv .tmp/ui/dist ui
	rm -rf .tmp/ui
# 将静态文件打包到命令工具
.PHONY: bindata
bindata: gen-bindata fmt

# 将静态文件打包到命令工具
.PHONY: gen-bindata
gen-bindata:
	mkdir -p .tmp/data
	cp -r ui .tmp/data/
	cp -r xiang .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data

# 编译可执行文件
.PHONY: xiang
xiang: bindata

	if [ ! -z "${XIANG_DOMAIN}" ]; then \
		mv share/const.go share/const.go.bak;	\
		sed "s/*.iqka.com/$(XIANG_DOMAIN)/g" share/const.go.bak > share/const.go; \
	fi;

#	GOOS=linux GOARCH=amd64 go build -v -o .tmp/xiang-linux-amd64
#	GOOS=linux GOARCH=arm GOARM=7 go build -v -o .tmp/xiang-linux-arm
#	GOOS=linux GOARCH=arm64 GOARM=7 go build -v -o .tmp/xiang-linux-arm64
	GOOS=darwin GOARCH=amd64 go build -v -o .tmp/xiang-darwin-amd64
	mkdir -p dist/bin
	mv .tmp/xiang-*-* dist/bin/
	chmod +x dist/bin/xiang-*-*
	
	if [ ! -z "${XIANG_DOMAIN}" ]; then \
		rm share/const.go; \
		mv share/const.go.bak share/const.go; \
	fi;

.PHONY: release
release: clean
	mkdir -p dist/release
	git clone git@github.com:YaoApp/yao.git dist/release
	git clone git@github.com:YaoApp/kun.git dist/kun
	git clone git@github.com:YaoApp/xun.git dist/xun
	git clone git@github.com:YaoApp/gou.git dist/gou

#	UI制品
	git clone git@github.com:YaoApp/xiang-ui .tmp/ui
	sed -ie "s/url('\/icon/url('\/xiang\/icon/g" .tmp/ui/public/icon/md_icon.css
	cd .tmp/ui && cnpm install && npm run build
	rm -rf dist/release/ui
	mv .tmp/ui/dist dist/release/ui

#	静态文件打包
	mkdir -p .tmp/data
	cp -r dist/release/ui .tmp/data/
	cp -r dist/release/xiang .tmp/data/
	go-bindata -fs -pkg data -o dist/release/data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/ui

#   制品
	if [ ! -z "${XIANG_DOMAIN}" ]; then \
		mv dist/release/share/const.go dist/release/share/const.go.bak;	\
		sed "s/*.iqka.com/$(XIANG_DOMAIN)/g" dist/release/share/const.go.bak > dist/release/share/const.go; \
	fi;

#	cd dist/release && CGO_ENABLED=1 CC=x86_64-linux-musl-gcc CGO_LDFLAGS="-static" GOOS=linux GOARCH=amd64 go build -v -o ../../.tmp/xiang-${VERSION}-linux-amd64
#	cd dist/release && GOOS=linux GOARCH=arm GOARM=7 go build -v -o ../../.tmp/xiang-${VERSION}-linux-arm
#	cd dist/release && GOOS=linux GOARCH=arm64 GOARM=7 go build -v -o ../../.tmp/xiang-${VERSION}-linux-arm64
	cd dist/release && CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -o ../../.tmp/xiang-${VERSION}-darwin-amd64
#	cd dist/release && GOOS=windows GOARCH=386 go build -v -o ../../.tmp/xiang-${VERSION}-windows-386
	
	rm -rf dist/release
	mkdir -p dist/release
	mv .tmp/xiang-*-* dist/release/
	chmod +x dist/release/xiang-*-*

.PHONY: hi
hi: 
	echo ${VERSION}

.PHONY: arm
arm: clean
	mkdir -p dist/release
	git clone git@github.com:YaoApp/yao.git dist/release
	git clone git@github.com:YaoApp/kun.git dist/kun
	git clone git@github.com:YaoApp/xun.git dist/xun
	git clone git@github.com:YaoApp/gou.git dist/gou

#	UI制品
	git clone git@github.com:YaoApp/xiang-ui.git .tmp/ui
	sed -ie "s/url('\/icon/url('\/xiang\/icon/g" .tmp/ui/public/icon/md_icon.css
	cd .tmp/ui && cnpm install && npm run build
	rm -rf dist/release/ui
	mv .tmp/ui/dist dist/release/ui

#	静态文件打包
	mkdir -p .tmp/data
	cp -r dist/release/ui .tmp/data/
	cp -r dist/release/xiang .tmp/data/
	go-bindata -fs -pkg data -o dist/release/data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/ui

#   制品
	mkdir -p dist
	cd dist/release && CC=arm-linux-gnueabi-gcc CXX=arm-linux-gnueabi-g++ CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 go build  -v -o ../../.tmp/xiang-${VERSION}-linux-arm

	rm -rf dist/release
	mkdir -p dist/release
	mv .tmp/xiang-*-* dist/release/
	chmod +x dist/release/xiang-*-*

.PHONY: linux
linux: clean
	mkdir -p dist/release
	git clone git@github.com:YaoApp/yao.git dist/release
	git clone git@github.com:YaoApp/kun.git dist/kun
	git clone git@github.com:YaoApp/xun.git dist/xun
	git clone git@github.com:YaoApp/gou.git dist/gou

#	UI制品
	git clone git@github.com:YaoApp/xiang-ui.git .tmp/ui
	sed -ie "s/url('\/icon/url('\/xiang\/icon/g" .tmp/ui/public/icon/md_icon.css
	cd .tmp/ui && cnpm install && npm run build
	rm -rf dist/release/ui
	mv .tmp/ui/dist dist/release/ui

#	静态文件打包
	mkdir -p .tmp/data
	cp -r dist/release/ui .tmp/data/
	cp -r dist/release/xiang .tmp/data/
	go-bindata -fs -pkg data -o dist/release/data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/ui

#   制品
	mkdir -p dist
	cd dist/release && CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=linux GOARCH=amd64 go build -v -o ../../.tmp/xiang-${VERSION}-linux-amd64

	rm -rf dist/release
	mkdir -p dist/release
	mv .tmp/xiang-*-* dist/release/
	chmod +x dist/release/xiang-*-*

.PHONY: artifact-linux
artifact-linux: clean
	mkdir -p dist/release

#	UI制品
	sed -ie "s/url('\/icon/url('\/xiang\/icon/g" ../ui/public/icon/md_icon.css
	cd ../ui && npm install && npm run build

#	静态文件打包
	mkdir -p .tmp/data
	cp -r ../ui/dist .tmp/data/ui
	cp -r xiang .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/ui

#   制品
	mkdir -p dist
	CGO_ENABLED=1 CGO_LDFLAGS="-static" go build -v -o dist/yao-${VERSION}-linux-${RUNNER_ARCH}
#	CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=linux GOARCH=amd64 go build -v -o dist/yao-${VERSION}-linux-amd64
#	CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ go build -v -o dist/yao-${VERSION}-linux-arm64

	mkdir -p dist/release
	mv dist/yao-*-* dist/release/
	chmod +x dist/release/yao-*-*
	ls -l dist/release/
	dist/release/yao-${VERSION}-linux-${RUNNER_ARCH} version

.PHONY: artifact-macos
artifact-macos: clean
	mkdir -p dist/release

#	UI制品
	sed -ie "s/url('\/icon/url('\/xiang\/icon/g" ../ui/public/icon/md_icon.css
	cd ../ui && npm install && npm run build

#	静态文件打包
	mkdir -p .tmp/data
	cp -r ../ui/dist .tmp/data/ui
	cp -r xiang .tmp/data/
	go-bindata -fs -pkg data -o data/bindata.go -prefix ".tmp/data/" .tmp/data/...
	rm -rf .tmp/data
	rm -rf .tmp/ui

#   制品
	mkdir -p dist
	CGO_ENABLED=1 go build -v -o dist/yao-${VERSION}-darwin-${RUNNER_ARCH}
#	CGO_ENABLED=1 CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -o dist/yao-${VERSION}-darwin-amd64
#	CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=darwin GOARCH=arm64 go build -v -o dist/yao-${VERSION}-linux-arm64

	mkdir -p dist/release
	mv dist/yao-*-* dist/release/
	chmod +x dist/release/yao-*-*
	ls -l dist/release/
	dist/release/yao-${VERSION}-darwin-${RUNNER_ARCH} version

.PHONY: win32
win32: bindata
	GOOS=windows GOARCH=386 go build -v -o .tmp/xiang-windows-386
	mkdir -p dist/bin
	mv .tmp/xiang-*-* dist/bin/
	chmod +x dist/bin/xiang-*-*


.PHONY: clean
clean: 
	rm -rf ./tmp
	rm -rf .tmp
	rm -rf dist


.PHONY: migrate
migrate:
	$(GO) test -tags $(TESTTAGS) -run TestCommandMigrate$