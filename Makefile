# Build output knobs (override via env/CLI).
EXTENSION     ?= 
DIST_DIR      ?= dist/
GOOS          ?= linux
GOARCH        ?= $(shell go env GOARCH)
BUILDINFOSDET ?= 

VERSION       ?= $(shell git describe --abbrev --long HEAD)
VERSION_PKG   ?= $(shell echo $(VERSION) | sed 's/^v//g')
DATE          :=  $(shell date +%FT%T%z)
BUILDINFOS    ?=  ($(DATE)$(BUILDINFOSDET))
LDFLAGS       ?= '-X main.version=$(VERSION) -X main.buildinfos=$(BUILDINFOS)'

OUTPUT := $(DIST_DIR)goflow2-$(VERSION_PKG)-$(GOOS)-$(GOARCH)$(EXTENSION)

.PHONY: proto
# Generate Go protobuf bindings.
proto:
	@echo generating protobuf
	protoc --go_opt=paths=source_relative --go_out=. pb/*.proto
	protoc --go_opt=paths=source_relative --go_out=. cmd/enricher/pb/*.proto

.PHONY: vet
# Run Go static analysis on the main package.
vet:
	go vet cmd/goflow2/main.go

.PHONY: test
# Run Go test suite.
test:
	go test -v ./...

.PHONY: test-race
test-race:
	go test -race ./...

.PHONY: test-cover
test-cover:
	go test -cover ./...

.PHONY: staticcheck
staticcheck:
	staticcheck ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: check
check: vet test lint staticcheck

.PHONY: prepare
# Ensure dist directory exists.
prepare:
	mkdir -p $(DIST_DIR)

.PHONY: clean
clean:
	rm -rf $(DIST_DIR)

.PHONY: build
# Build the goflow2 binary.
build: prepare
	CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o $(OUTPUT) cmd/goflow2/main.go

.PHONY: print-output
print-output:
	@echo $(OUTPUT)
