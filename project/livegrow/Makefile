SHELL := /bin/sh

GO ?= go
GOCACHE ?= /private/tmp/livegrow-gocache
export GOCACHE

.PHONY: fmt fmt-check test vet coverage check run

fmt:
	$(GO)fmt -w $$(find . -name '*.go' -not -path './vendor/*')

fmt-check:
	@test -z "$$($(GO)fmt -l .)" || (echo 'gofmt required for:'; $(GO)fmt -l .; exit 1)

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

coverage:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

check: fmt-check test vet

run:
	$(GO) run ./cmd/livegrow
