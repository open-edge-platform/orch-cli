# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
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
PATH := $(GOPATH)/bin:$(PATH)

INSTALL_PATH  ?= /usr/local/bin

OAPI_CODEGEN_VERSION = v2.4.1
MOCKGEN_VERSION = v0.5.2

FUZZ_TIME ?= 30m

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
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -buildmode=pie -trimpath -mod=$(GO_MOD) -gcflags="all=-spectre=all -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -extldflags=-static -X $(PKG)/internal/cli.Version=`cat VERSION`" -o build/_output/$(RELEASE_NAME) $(CMD_DIR)

install: build
	@# Help: Installs client tool
	sudo cp build/_output/$(RELEASE_NAME) ${INSTALL_PATH}

lint:
	@# Help: Runs lint stage
	golangci-lint run --timeout 10m
	yamllint .
	$(MAKE) mdlint

MD_FILES := $(shell find . -type f \( -name '*.md' \) -print )
mdlint: ## lint all markdown README.md files
	markdownlint --version
	markdownlint *.md

test: mod-update
	@# Help: Runs test stage
	@trap 'rm -rf internal/cli/preflight_error* internal/cli/import_error*' EXIT; \
	go test -race -gcflags=-l `go list $(PKG)/cmd/... $(PKG)/internal/... $(PKG)/pkg/...`

fuzz:
	@# Help: Runs all Go fuzzing functions, one at a time, in each package, continues on failure, writes output to fuzz.log
	rm -f fuzz.log
	for pkg in $$(go list ./cmd/... ./internal/... ./pkg/...); do \
		for fuzzfunc in $$(go test -list '^Fuzz' $$pkg | grep '^Fuzz' | awk '{print $$1}'); do \
			echo "==> GOMEMLIMIT=2GiB GOMAXPROCS=2 go test -fuzz=$$fuzzfunc -fuzztime=$(FUZZ_TIME) -parallel=1 $$pkg" | tee -a fuzz.log ; \
			sleep 2 ; \
			GOMEMLIMIT=2GiB GOMAXPROCS=2 go test -fuzz=^$$fuzzfunc$$ -fuzztime=$(FUZZ_TIME) -parallel=1 $$pkg 2>&1 | tee -a fuzz.log ; \
			rm -rf internal/cli/preflight_error* internal/cli/import_error*; \
		done \
	done

fetch-catalog-openapi:
	@# Help: Fetch the Catalog OpenAPI spec
	curl -sSL https://raw.githubusercontent.com/open-edge-platform/orch-utils/main/tenancy-api-mapping/openapispecs/generated/amc-app-orch-catalog-openapi.yaml -o pkg/rest/catalog/amc-app-orch-catalog-openapi.yaml
	curl -sSL https://raw.githubusercontent.com/open-edge-platform/orch-utils/main/tenancy-api-mapping/openapispecs/generated/amc-app-orch-catalog-utilities-openapi.yaml -o pkg/rest/catalogutilities/amc-app-orch-catalog-utilities-openapi.yaml

fetch-cluster-openapi:
	@# Help: Fetch the Cluster Manager OpenAPI spec
	curl -sSL https://raw.githubusercontent.com/open-edge-platform/orch-utils/main/tenancy-api-mapping/openapispecs/generated/amc-cluster-manager-openapi.yaml -o pkg/rest/cluster/amc-cluster-manager-openapi.yaml

fetch-infra-openapi:
	@# Help: Fetch the Infra Manager OpenAPI spec
	curl -sSL https://raw.githubusercontent.com/open-edge-platform/infra-core/main/apiv2/api/openapi/openapi.yaml -o pkg/rest/infra/amc-infra-core-edge-infrastructure-manager-openapi-all.yaml

fetch-rps-openapi:
	@# Help: Fetch the OpenDMT RPS OpenAPI spec
	curl -sSL https://raw.githubusercontent.com/open-edge-platform/orch-utils/main/tenancy-api-mapping/openapispecs/generated/amc-opendmt-rps-openapi.yaml -o pkg/rest/rps/amc-opendmt-rps-openapi.yaml

fetch-utils-openapi:
	@# Help: Fetch the Utils API OpenAPI spec
	curl -sSL https://raw.githubusercontent.com/open-edge-platform/orch-utils/main/tenancy-api-mapping/openapispecs/generated/orch-utils.tenancy-datamodel.openapi.yaml -o pkg/rest/tenancy/orch-utils.tenancy-datamodel.openapi.yaml

fetch-openapi: fetch-catalog-openapi fetch-cluster-openapi fetch-infra-openapi fetch-rps-openapi fetch-utils-openapi
	@# Help: Fetch OpenAPI specs for all components


oapi-codegen-dependency:
	@# Help: Install oapi-codegen if not present
	@if ! command -v oapi-codegen >/dev/null 2>&1; then \
		echo "Installing oapi-codegen $(OAPI_CODEGEN_VERSION)..."; \
		go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION); \
	else \
		echo "oapi-codegen already installed: $$(command -v oapi-codegen)"; \
	fi

mockgen-dependency:
	@# Help: Install mockgen if not present
	@if ! command -v mockgen >/dev/null 2>&1; then \
		echo "Installing mockgen $(MOCKGEN_VERSION)..."; \
		go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION); \
	else \
		echo "mockgen already installed: $$(command -v mockgen)"; \
	fi

# Supported oapi-codegen version v2.4.1
# install command: go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1
rest-client-gen: oapi-codegen-dependency
	@# Help: Generate Rest client from the MT GW openapi spec.
	oapi-codegen -generate client -old-config-style -package catalog -o pkg/rest/catalog/client.go pkg/rest/catalog/amc-app-orch-catalog-openapi.yaml
	oapi-codegen -generate types -old-config-style -package catalog -o pkg/rest/catalog/types.go pkg/rest/catalog/amc-app-orch-catalog-openapi.yaml
	oapi-codegen -generate client -old-config-style -package catalogutilities -o pkg/rest/catalogutilities/client.go pkg/rest/catalogutilities/amc-app-orch-catalog-utilities-openapi.yaml
	oapi-codegen -generate types -old-config-style -package catalogutilities -o pkg/rest/catalogutilities/types.go pkg/rest/catalogutilities/amc-app-orch-catalog-utilities-openapi.yaml	
	oapi-codegen -generate client -old-config-style -package deployment -o pkg/rest/deployment/client.go pkg/rest/deployment/amc-app-orch-deployment-app-deployment-manager-openapi.yaml
	oapi-codegen -generate types -old-config-style -package deployment -o pkg/rest/deployment/types.go pkg/rest/deployment/amc-app-orch-deployment-app-deployment-manager-openapi.yaml
	oapi-codegen -generate client -old-config-style -package cluster -o pkg/rest/cluster/client.go pkg/rest/cluster/amc-cluster-manager-openapi.yaml
	oapi-codegen -generate types -old-config-style -package cluster -o pkg/rest/cluster/types.go pkg/rest/cluster/amc-cluster-manager-openapi.yaml
	oapi-codegen -generate client -old-config-style -package infra -o pkg/rest/infra/client.go pkg/rest/infra/amc-infra-core-edge-infrastructure-manager-openapi-all.yaml
	oapi-codegen -generate types -old-config-style -package infra -o pkg/rest/infra/types.go pkg/rest/infra/amc-infra-core-edge-infrastructure-manager-openapi-all.yaml
	oapi-codegen -generate client -old-config-style -package rps -o pkg/rest/rps/client.go pkg/rest/rps/amc-opendmt-rps-openapi.yaml
	oapi-codegen -generate types -old-config-style -package rps -o pkg/rest/rps/types.go pkg/rest/rps/amc-opendmt-rps-openapi.yaml
	oapi-codegen -generate client -old-config-style -package tenancy -o pkg/rest/tenancy/client.go pkg/rest/tenancy/orch-utils.tenancy-datamodel.openapi.yaml
	oapi-codegen -generate types -old-config-style -package tenancy -o pkg/rest/tenancy/types.go pkg/rest/tenancy/orch-utils.tenancy-datamodel.openapi.yaml

# Supported mockgen version v0.5.2
# install command: go install go.uber.org/mock/mockgen@v0.5.2
mock-client-gen: mockgen-dependency
	@# Help: Generate mock clients for testing
	mockgen -source=pkg/rest/catalog/client.go -destination=pkg/rest/catalog/mock_client.go -package=catalog
	mockgen -source=pkg/rest/catalogutilities/client.go -destination=pkg/rest/catalogutilities/mock_client.go -package=catalogutilities
	mockgen -source=pkg/rest/deployment/client.go -destination=pkg/rest/deployment/mock_client.go -package=deployment
	mockgen -source=pkg/rest/cluster/client.go -destination=pkg/rest/cluster/mock_client.go -package=cluster
	mockgen -source=pkg/rest/infra/client.go -destination=pkg/rest/infra/mock_client.go -package=infra
	mockgen -source=pkg/rest/rps/client.go -destination=pkg/rest/rps/mock_client.go -package=rps
	mockgen -source=pkg/rest/tenancy/client.go -destination=pkg/rest/tenancy/mock_client.go -package=tenancy

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
