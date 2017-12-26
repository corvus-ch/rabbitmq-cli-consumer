MAKEFLAGS += --warn-undefined-variables
SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := test
.DELETE_ON_ERROR:
.SUFFIXES:

# ---------------------
# Environment variables
# ---------------------

GOPATH := $(shell go env GOPATH)

# ------------------
# Internal variables
# ------------------

package_name   := rabbitmq-cli-consumer
test_args      := -v

# -------
# Targets
# -------

.PHONY: install
install:
	go get -t -d -tags=integration ./...

build: ${package_name}

.PHONY: test
test:
	go test ${test_args} ./...

.PHONY: test-integration
test-integration:
	go test -tags=integration $(test_args) ./...

${package_name}: **/*.go *.go
	go build

.PHONY: clean
clean:
	rm -rf ${package_name}
	rm *.log
