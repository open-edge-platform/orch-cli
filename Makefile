# SPDX-FileCopyrightText: 2022-present Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

CMD_DIR         ?= ./cmd/orch-cli

PKG     	:= github.com/open-edge-platform/cli

RELEASE_DIR     ?= release
RELEASE_NAME    ?= orch-cli
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
	if [ "$@" == "release/orch-cli-linux-amd64" ]; \
	then \
	  go build $(linux_opts) -o "$@" $(CMD_DIR) ;\
	else \
	  go build -o "$@" $(CMD_DIR); \
	fi

release: $(RELEASE_BINS)
	@# Help: Builds releasable binaries for multiple architectures. test

mod-update:
	@# Help: Update Go modules
	go mod tidy

build: mod-update
	@# Help: Runs build stage
	@sed "s/VERSION/`cat VERSION`/g" internal/cli/version.go.template > internal/cli/version.go
	go build -o build/_output/$(RELEASE_NAME) $(CMD_DIR)

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

fetch-catalog-openapi:
	@# Help: Fetch the Catalog OpenAPI spec
	curl -sSL https://raw.githubusercontent.com/open-edge-platform/orch-utils/main/tenancy-api-mapping/openapispecs/generated/amc-app-orch-catalog-openapi.yaml -o pkg/rest/catalog/amc-app-orch-catalog-openapi.yaml

fetch-cluster-openapi:
	@# Help: Fetch the Cluster Manager OpenAPI spec
	curl -sSL https://raw.githubusercontent.com/open-edge-platform/orch-utils/main/tenancy-api-mapping/openapispecs/generated/amc-cluster-manager-openapi.yaml -o pkg/rest/cluster/amc-cluster-manager-openapi.yaml


fetch-infra-openapi:
	@# Help: Fetch the Infra Manager OpenAPI spec
	curl -sSL https://raw.githubusercontent.com/open-edge-platform/orch-utils/main/tenancy-api-mapping/openapispecs/generated/amc-infra-core-edge-infrastructure-manager-openapi-all.yaml -o pkg/rest/infra/amc-infra-core-edge-infrastructure-manager-openapi-all.yaml

fetch-openapi: fetch-catalog-openapi fetch-cluster-openapi fetch-infra-openapi
	@# Help: Fetch OpenAPI specs for all components

rest-client-gen:
	@# Help: Generate Rest client from the MT GW openapi spec.
	oapi-codegen -generate client -old-config-style -package catalog -o pkg/rest/catalog/client.go pkg/rest/catalog/amc-app-orch-catalog-openapi.yaml
	oapi-codegen -generate types -old-config-style -package catalog -o pkg/rest/catalog/types.go pkg/rest/catalog/amc-app-orch-catalog-openapi.yaml
	oapi-codegen -generate client -old-config-style -package deployment -o pkg/rest/deployment/client.go pkg/rest/deployment/amc-app-orch-deployment-app-deployment-manager-openapi.yaml
	oapi-codegen -generate types -old-config-style -package deployment -o pkg/rest/deployment/types.go pkg/rest/deployment/amc-app-orch-deployment-app-deployment-manager-openapi.yaml
	oapi-codegen -generate client -old-config-style -package cluster -o pkg/rest/cluster/client.go pkg/rest/cluster/amc-cluster-manager-openapi.yaml
	oapi-codegen -generate types -old-config-style -package cluster -o pkg/rest/cluster/types.go pkg/rest/cluster/amc-cluster-manager-openapi.yaml
	oapi-codegen -generate client -old-config-style -package infra -o pkg/rest/infra/client.go pkg/rest/infra/amc-infra-core-edge-infrastructure-manager-openapi-all.yaml
	oapi-codegen -generate types -old-config-style -package infra -o pkg/rest/infra/types.go pkg/rest/infra/amc-infra-core-edge-infrastructure-manager-openapi-all.yaml

cli-docs:
	@# Help: Generates markdowns for the orchestrator cli
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
