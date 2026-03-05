GO ?= go
TEST_TIMEOUT ?= 120s

.PHONY: test test-v test-cover test-race

test:
	$(GO) test -timeout=$(TEST_TIMEOUT) -count=1 ./...

test-v:
	$(GO) test -v -timeout=$(TEST_TIMEOUT) -count=1 ./...

test-cover:
	$(GO) test -v -timeout=$(TEST_TIMEOUT) -count=1 \
		-coverprofile=coverage.out -covermode=count ./...
	$(GO) tool cover -func=coverage.out | tail -1

test-race:
	$(GO) test -race -v -timeout=$(TEST_TIMEOUT) -count=1 ./...

test-ci:
	@echo "mode: count" > coverage.out
	@for d in $$($(GO) list ./...); do \
		$(GO) test -v -timeout=$(TEST_TIMEOUT) -count=1 \
			-covermode=count -coverprofile=profile.out \
			-coverpkg=$$d $$d > tmp.out; \
		cat tmp.out; \
		if grep -q "^--- FAIL" tmp.out; then \
			rm -f tmp.out profile.out; \
			exit 1; \
		fi; \
		if [ -f profile.out ]; then \
			grep -v "mode:" profile.out >> coverage.out; \
			rm profile.out; \
		fi; \
		rm -f tmp.out; \
	done
