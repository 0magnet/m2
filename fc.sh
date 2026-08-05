#!/bin/bash
echo "go mod tidy -v"
go mod tidy -v
echo "gofmt -w $(echo *.go)"
gofmt -w $(echo *.go)
echo "gofmt -w $(echo ui/*.go)"
gofmt -w  $(echo ui/*.go)
echo "gofmt -w $(echo wasm/*.go)"
gofmt -w  $(echo wasm/*.go)
echo "goimports -w $(echo *.go) $(echo ui/*.go) $(echo wasm/*.go)"
goimports -w $(echo *.go) $(echo ui/*.go) $(echo wasm/*.go)
echo "go run github.com/golangci/golangci-lint/cmd/golangci-lint@main --version"
go run github.com/golangci/golangci-lint/cmd/golangci-lint@main --version
echo "go run github.com/golangci/golangci-lint/cmd/golangci-lint@main run -c .golangci.yml $(echo *.go)"
go run github.com/golangci/golangci-lint/cmd/golangci-lint@main run -c .golangci.yml $(echo *.go)
echo "go run github.com/golangci/golangci-lint/cmd/golangci-lint@main run -c .golangci.yml $(echo ui/*.go)"
go run github.com/golangci/golangci-lint/cmd/golangci-lint@main run -c .golangci.yml $(echo ui/*.go)
#echo "go run github.com/golangci/golangci-lint/cmd/golangci-lint@main run -c .golangci.yml $(echo wasm/*.go)"
#go run github.com/golangci/golangci-lint/cmd/golangci-lint@main run -c .golangci.yml $(echo wasm/*.go)
#go vet -all -mod=vendor
