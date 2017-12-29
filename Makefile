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
WORK_DIR := ".coverage"

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
	go get github.com/axw/gocov/gocov

build: ${package_name}

.PHONY: test
test:
	WORK_DIR=$(WORK_DIR) ./script/coverage ${test_args}

.PHONY: test-integration
test-integration:
	WORK_DIR=$(WORK_DIR) ./script/coverage -tags=integration ${test_args}

${package_name}: **/*.go *.go
	go build

.PHONY: clean
clean:
	rm -rf ${package_name} *.log *.out $(WORK_DIR)
