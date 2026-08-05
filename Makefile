
.PHONY : check lint install-linters dep test
.PHONY : build clean install format  bin

check: lint test ## Run linters and tests

lint: ## Run linters. Use make install-linters first
	golangci-lint run -c .golangci.yml ./m2.go
	GOOS=js GOARCH=wasm  golangci-lint run -c .golangci.yml ./b.go
	GOOS=js GOARCH=wasm  golangci-lint run -c .golangci.yml ./stl.go

test: ## Run tests
	-go clean -testcache &>/dev/null
	go test ./m2.go

tidy: ## Tidies and vendors dependencies.
	go mod tidy -v

format: tidy ## Formats the code. Must have goimports and goimports-reviser installed (use make install-linters).
	goimports -w -local github.com/0magnet/magnets ./*.go
	find . -type f -name '*.go' -not -path "./.git/*" -not -path "./vendor/*"  -exec goimports-reviser -project-name github.com/0magnet/magnets {} \;

dep: tidy ## Sorts dependencies
	go mod vendor -v

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
