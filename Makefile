.PHONY: generate lint test fmt build docs
.DEFAULT_GOAL := help


COMMIT = $(shell git rev-parse HEAD)
ifeq ($(shell git status --porcelain),)
	VERSION = $(shell git describe --tags --abbrev=0)
endif


TEST_FORMAT ?= pkgname

define build
	@mkdir -p build
	$(eval OUTPUT := $(if $(filter windows,$(1)),rumo-$(1)-$(2).exe,rumo-$(1)-$(2)))
	$(eval URL := $(shell if [ -z "$(VERSION)" ]; then echo -n "" ; else echo -n https://github.com/malivvan/rumo/releases/download/$(VERSION)/$(OUTPUT); fi))
	$(eval SERIAL := $(shell if [ -z "$(VERSION)" ]; then uuidgen --random ; else uuidgen --sha1 --namespace @url --name $(URL); fi))
	@echo "$(OUTPUT)"
	@CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) GOFLAGS=-tags="$(4)" cyclonedx-gomod \
      app -json -packages -licenses \
      -serial=$(SERIAL) \
      -output build/$(OUTPUT).json -main ./cmd > /dev/null 2>&1
	@CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go \
	build -trimpath -tags="$(4)" \
	  -ldflags="$(3) \
	  -buildid=$(SERIAL) \
	  -X github.com/malivvan/rumo.commit=$(COMMIT) \
	  -X github.com/malivvan/rumo.version=$(VERSION)" \
	  -o build/$(OUTPUT) ./cmd
	@if [ ! -f build/release.md ]; then \
	  echo "| filename | serial |" > build/release.md; \
	  echo "|----------|--------|" >> build/release.md; \
	fi
	@if [ -z "$(VERSION)" ]; then \
	  echo "| $(OUTPUT) | $(SERIAL) |" >> build/release.md; \
	else \
	  echo "| [$(OUTPUT)]($(URL)) | [$(SERIAL)]($(URL).json) |" >> build/release.md; \
	fi
endef

install/build:
	@go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest

install/test:
	@go install golang.org/x/lint/golint@latest
	@go install gotest.tools/gotestsum@latest

install: install/build install/test

lint:
	@golint -set_exit_status ./vm/...

test: generate lint
	@gotestsum --format $(TEST_FORMAT) --format-hide-empty-pkg --hide-summary skipped --raw-command -- go test -json -race -cover ./...
	@go run ./cmd ./vm/testdata/cli/test.rumo > /dev/null 2>&1 || (echo "END TO END TEST FAILED" && exit 1)

fmt:
	@go fmt ./...

generate:
	@go generate ./vm/...

build: clean
	$(call build,$(shell go env GOOS),$(shell go env GOARCH),,)

preview: clean
	$(call build,$(shell go env GOOS),$(shell go env GOARCH),-s -w,)

release: clean
	$(call build,linux,386,-s -w,)
	$(call build,linux,amd64,-s -w,)
	$(call build,linux,arm,-s -w,)
	$(call build,linux,arm64,-s -w,)
	$(call build,darwin,amd64,-s -w,)
	$(call build,darwin,arm64,-s -w,)
	$(call build,windows,amd64,-s -w,)
	$(call build,windows,386,-s -w,)
	$(call build,windows,arm,-s -w,)
	$(call build,windows,arm64,-s -w,)

clean:
	@rm -rf ./build


info: ## Show information about the dependencies
	@goda cut -h - "github.com/malivvan/rumo/...:all" | grep --color=never github.com/malivvan/rumo | sort
	@echo
	@goda cut -h - "github.com/malivvan/rumo/...:all" | grep -v github.com/malivvan/rumo | sort
	@echo
	@goda cut -h - -std "github.com/malivvan/rumo/...:all" | grep -v github.com | grep -v golang.org | sort

.PHONY: help
help: ## Shows this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'