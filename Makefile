# SPDX-FileCopyrightText: 2022-present Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

CMD_DIR         ?= ./cmd/catalog-cli

PKG     	:= github.com/open-edge-platform/cli
GOPRIVATE   := github.com/open-edge-platform/*

RELEASE_DIR     ?= release
RELEASE_NAME    ?= catalog
RELEASE_OS_ARCH ?= linux-amd64 linux-arm64 windows-amd64 darwin-amd64
RELEASE_BINS    := $(foreach rel,$(RELEASE_OS_ARCH),$(RELEASE_DIR)/$(RELEASE_NAME)-$(rel))

GOLANG_COVER_VERSION = v0.2.0
GOLANG_GOCOVER_COBERTURA_VERSION = v1.2.0
GOPATH := $(shell go env GOPATH)

INSTALL_PATH  ?= /usr/local/bin

.PHONY: build test

all:  build lint test
	@# Help: Runs build, lint, test stages

# Functions to extract the OS/ARCH
rel_os    = $(word 2, $(subst -, ,$(notdir $@)))
rel_arch  = $(word 3, $(subst -, ,$(notdir $@)))

linux_opts = -trimpath -gcflags="all=-spectre=all -N -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w"

$(RELEASE_BINS):
	export GOOS=$(rel_os) ;\
	export GOARCH=$(rel_arch) ;\
	if [ "$@" == "release/catalog-linux-amd64" ]; \
	then \
	  GOPRIVATE=$(GOPRIVATE) go build $(linux_opts) -o "$@" $(CMD_DIR) ;\
	else \
	  GOPRIVATE=$(GOPRIVATE) go build -o "$@" $(CMD_DIR); \
	fi

release: $(RELEASE_BINS)
	@# Help: Builds releasable binaries for multiple architectures. test

mod-update:
	@# Help: Update Go modules
	GOPRIVATE=$(GOPRIVATE) go mod tidy

build: mod-update
	@# Help: Runs build stage
	@sed "s/VERSION/`cat VERSION`/g" internal/cli/version.go.template > internal/cli/version.go
	GOPRIVATE=$(GOPRIVATE) go build -o build/_output/$(RELEASE_NAME) $(CMD_DIR)

install: build
	@# Help: Installs client tool
	cp build/_output/$(RELEASE_NAME) ${INSTALL_PATH}

lint:
	@# Help: Runs lint stage
	golangci-lint run --timeout 10m
	yamllint .

test: mod-update
	@# Help: Runs test stage
	go test -race -gcflags=-l `go list $(PKG)/cmd/... $(PKG)/internal/... $(PKG)/pkg/...`

rest-client-gen:
	@# Help: Generate Rest client from the MT GW openapi spec.
	oapi-codegen -generate client -old-config-style -package catalog -o pkg/rest/catalog/client.go pkg/rest/catalog/application-catalog.yaml
	oapi-codegen -generate types -old-config-style -package catalog -o pkg/rest/catalog/types.go pkg/rest/catalog/application-catalog.yaml
	oapi-codegen -generate client -old-config-style -package deployment -o pkg/rest/deployment/client.go pkg/rest/deployment/application-deployment.yaml
	oapi-codegen -generate types -old-config-style -package deployment -o pkg/rest/deployment/types.go pkg/rest/deployment/application-deployment.yaml

cli-docs:
	@# Help: Generates markdowns for the catalog cli
	go run cmd/cli-docs-gen/main.go

go-cover-dependency:
	go tool cover -V || go install golang.org/x/tools/cmd/cover@${GOLANG_COVER_VERSION}
	go install github.com/boumenot/gocover-cobertura@${GOLANG_GOCOVER_COBERTURA_VERSION}

coverage: go-cover-dependency
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	go test -gcflags=-l `go list $(PKG)/cmd/... $(PKG)/internal/... $(PKG)/pkg/... | grep -v "/mocks" | grep -v "/test/"` -v -coverprofile=coverage.txt -covermode count
	${GOPATH}/bin/gocover-cobertura < coverage.txt > coverage.xml
	#$(GOCMD) tool cover -html=cover.out -o cover.html
	#$(GOCMD) tool cover -func cover.out -o cover.function-coverage.log
	@echo "---END MAKEFILE COVERAGE---"

reuse-tool:
	@# Help: Install reuse if not present
	command -v reuse || pip install reuse

license: reuse-tool
	@# Help: Check licensing with the reuse tool
	reuse lint

list: help
	@# Help: displays make targets

clean: ## remove the test collateral
	rm -rf vendor build/_output
	go clean -testcache

help:	
	@printf "%-20s %s\n" "Target" "Description"
	@printf "%-20s %s\n" "------" "-----------"
	@make -pqR : 2>/dev/null \
        | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' \
        | sort \
        | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' \
        | xargs -I _ sh -c 'printf "%-20s " _; make _ -nB | (grep -i "^# Help:" || echo "") | tail -1 | sed "s/^# Help: //g"'
