PACKAGES := ./...
DEPENDENCIES := ./...
TEST_ARGS ?=

all: build test-silent

build: rabbitmq-cli-consumer

rabbitmq-cli-consumer: **/*.go *.go
	go build

.PHONY: test
test:
	go test -v $(PACKAGES)

.PHONY: test-silent
test-silent:
	go test $(PACKAGES)

.PHONY: test-integration
test-integration:
	go test -tags=integration -v $(TEST_ARGS) $(PACKAGES)

.PHONY: format
format:
	go fmt $(PACKAGES)

.PHONY: deps
deps:
	go get $(DEPENDENCIES)

.PHONY: clean
clean:
	rm rabbitmq-cli-consumer

