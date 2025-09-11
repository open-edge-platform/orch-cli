<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
<!-- 
  SPDX-License-Identifier: LicenseRef-third-party-software 
-->

# Edge Manageability Framework Command Line Interface

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Overview

The Orchestrator CLI is a standalone utility which offers command line commands
to interact and manage various Orchestrator resources using the
REST API endpoints.

Currently the `orch-cli` is supported for [Infrastructure Management] operations and
in `beta` for [Application Orchestration] and [Cluster Orchestration].

Currently allows user to `create`, `get`, `list`, `set` and `delete` the following
Orchestrator entities:

- Hosts
- Operating System Profiles (OsProfiles)
- Sites
- Regions
- Custom Configs
- Clusters
- Application images registries
- Application artifacts
- Applications
- Application profiles
- Deployment packages (a.k.a bundles)
- Deployment profiles (a.k.a. bundle profiles)
- Application references of deployment packages
- Deployments

Additionally, the CLI supports advanced operations.

- [Infrastructure Management]:

  - Host registration in bulk via the upload of a CSV file.
  - Validation of the CSV file for the host registration.

- [Application Orchestration]:

  - Deployment packages can be created/updated by loading contents of a
    directory structure via the `load` command.

## Get Started

Instructions on how to install and set up the CLI on your development machine.

### Download pre-built artefacts

Pull the orch-cli compressed artifact from the registry-rs.edgeorchestration.intel.com and unpack the archive. The archive contains the orch-cli binary.

```shell
oras pull registry-rs.edgeorchestration.intel.com/edge-orch/files/orch-cli:3.1
tar xf orch-cli-package.tar.gz
```

### Build From Source

#### Dependencies

Firstly, please verify that all dependencies have been installed. This code requires the following tools to be
installed on your development machine:

- [Go\* programming language](https://go.dev) - check [$GOVERSION_REQ](../version.mk)
- [golangci-lint](https://github.com/golangci/golangci-lint) - check [$GOLINTVERSION_REQ](../version.mk)
- [go-junit-report](https://github.com/jstemmer/go-junit-report) - check [$GOJUNITREPORTVERSION_REQ](../version.mk)
- [Python\* programming language version 3.10 or later](https://www.python.org/downloads)
- [gocover-cobertura](https://github.com/boumenot/gocover-cobertura) - check [$GOCOBERTURAVERSION_REQ](../version.mk)
- [buf](https://github.com/bufbuild/buf)
- [protoc-gen-doc](https://github.com/pseudomuto/protoc-gen-doc)
- [protoc-gen-go-grpc](https://pkg.go.dev/google.golang.org/grpc)
- [protoc-gen-go](https://pkg.go.dev/google.golang.org/protobuf)

#### Build the Binary

Build the project as follows:

```bash
# Build go binary
make build
```

The `orch-cli` binary is created in the `orch-cli/build/_output` directory

To install the cli:

```bash
# Install go binary
make install
```

By default, the binary will be installed to `/usr/local/bin`,
the location can be overridden by setting the `INSTALL_PATH` environmental variable.

## Contribute

We welcome contributions from the community! To contribute, please open a pull
request to have your changes reviewed and merged into the `main` branch.
We encourage you to add appropriate unit tests and end-to-end tests if
your contribution introduces a new feature. See [Contributor Guide] for
information on how to contribute to the project.

### Develop

The Orchestrator CLI is developed in the **Go** language and is built as a
standalone executble for Linux, Mac, and Windows platforms. The CLI uses
the industry standard [viper](https://github.com/spf13/viper) and
[cobra](https://github.com/spf13/cobra) libraries.

Below are some of the important make targets which developer should be aware about.

Build the component binary as follows:

```bash
# Build go binary
make build
```

Unit tests are run for each PR and the developer can run unit tests locally as follows:

```bash
# Run unit tests
make test
```

Linter checks are run for each PR and the developer can run linter check locally as follows:

```bash
make lint
```

License checks are run for each PR and the developer can run license check locally as follows:

```bash
make license
```

## Community and Support

To learn more about the project, its community, and governance, visit the [Edge Orchestrator Community].

For support, start with [Troubleshooting] or [Contact us].

## License

The Orchestrator CLI is licensed under [Apache 2.0 License]

Last Updated Date: June 16, 2025

[Application Orchestration]: https://github.com/open-edge-platform/app-orch-deployment
[Infrastructure Management]: https://github.com/open-edge-platform/infra-charts
[Cluster Orchestration]: https://github.com/open-edge-platform/cluster-extensions
[Contributor Guide]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html
[Troubleshooting]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/troubleshooting/index.html
[Contact us]: https://github.com/open-edge-platform
[Edge Orchestrator Community]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/index.html
[Apache 2.0 License]: LICENSES/Apache-2.0.txt
