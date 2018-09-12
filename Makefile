SHELL = bash

GO ?= go
arch ?= amd64

PLATFORMS := linux freebsd darwin

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
current_dir := $(notdir $(patsubst %/,%,$(dir $(mkfile_path))))

.PHONY: all, test, bench, clean, $(PLATFORMS)

all: $(PLATFORMS)

clean:
	@rm build/*

test:
	@$(GO) test -v -race -cover ./...

bench:
	@$(GO) test -bench . ./...

$(PLATFORMS):
	@echo "Build application"
	@GOOS=$@ GOARCH=$(arch) $(GO) build -o build/$(current_dir)-$@-$(arch) cmd/bot/main.go
	@echo "Setting right permissions"
	@chmod 6755 build/$(current_dir)-$@-$(arch)
