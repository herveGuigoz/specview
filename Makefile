NPM ?= npm
GO ?= go

.PHONY: build frontend test integration release clean
build: frontend
	mkdir -p bin
	CGO_ENABLED=0 $(GO) build -trimpath -o bin/specview ./cmd/specview

frontend:
	$(NPM) --prefix ui ci --no-audit --no-fund
	$(NPM) --prefix ui run typecheck
	$(NPM) --prefix ui run build

test:
	$(GO) test -race ./...
	$(NPM) --prefix ui run typecheck
	$(NPM) --prefix ui run test:unit

integration: build
	SPECVIEW_REQUIRE_NETWORK_TESTS=1 $(GO) test -count=1 ./internal/server
	$(NPM) --prefix ui test

release: frontend
	$(GO) run ./scripts/release.go

clean:
	rm -rf bin internal/web/dist test-results
