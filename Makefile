MAKEFLAGS += --warn-undefined-variables
SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := test
.DELETE_ON_ERROR:
.SUFFIXES:

# ---------------------
# Environment variables
# ---------------------

GOPATH     ?= $(shell go env GOPATH)
GORELEASER ?= goreleaser
TEST_ARGS  ?=

# ------------------
# Internal variables
# ------------------

binary_name   = rabbitmq-cli-consumer
cover_file    = c.out

# -------
# Targets
# -------

.PHONY: deps
deps:
	go mod download

.PHONY: test
test: $(cover_file) $(binary_name)

$(cover_file): $(wildcard **/*_test.go)
	go test -covermode=atomic -coverprofile=$@ $(TEST_ARGS) ./...

.PHONY: build
build: $(binary_name)

$(binary_name): $(wildcard **/*.go)
	go build

.PHONY: release
release:
	$(GORELEASER) --rm-dist

.PHONY: snapshot
snapshot:
	$(GORELEASER) --rm-dist --snapshot

.PHONY: clean
clean:
	rm -rf $(binary_name) $(cover_file) *.log **/*.log
