<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# Application Orchestration Command Line Interface

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Overview

The orchestrator CLI is a standalone utility which interacts with the Application Catalog and deployment
services using their REST API endpoints and allows the operator to manage various Orchestrator resources
from the command line.

The supported CLI usage allows user to `create`, `get`, `list`, `set` and `delete` the following
Orchestrator entities:
* registries
* artifacts
* applications
* application profiles
* deployment packages (a.k.a bundles)
* deployment profiles (a.k.a. bundle profiles)
* application references of deployment packages
* deployments

The CLI also supports usage where items can be created/updated by loading contents of a directory structure via
the `load` usage.

## Develop

The Orchestrator CLI is developed in the **Go** language and is built as a stndlone executble for Linux, Mac, and Windows platforms. The CLI uses
the industry standard [viper](https://github.com/spf13/viper) and [cobra](https://github.com/spf13/cobra) libraries to provide a command line interface.

### Dependencies

This code requires the following tools to be installed on your development machine:

- [Go\* programming language](https://go.dev)
- [golangci-lint](https://github.com/golangci/golangci-lint)
- [Python\* programming language version 3.10 or later](https://www.python.org/downloads)
- [buf](https://github.com/bufbuild/buf)
- [protoc-gen-doc](https://github.com/pseudomuto/protoc-gen-doc)
- [protoc-gen-go-grpc](https://pkg.go.dev/google.golang.org/grpc)
- [protoc-gen-go](https://pkg.go.dev/google.golang.org/protobuf)

## Build

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

## Contribute

We welcome contributions from the community! To contribute, please open a pull request to have your changes reviewed
and merged into the `main` branch. We encourage you to add appropriate unit tests and end-to-end tests if
your contribution introduces a new feature. See [Contributor Guide] for information on how to contribute to the project.

## Community and Support

To learn more about the project, its community, and governance, visit the [Edge Orchestrator Community].

For support, start with [Troubleshooting] or [Contact us].

## License

The Orchestrator CLI is licensed under [Apache 2.0 License]

[Application Orchestration Deployment]: https://github.com/open-edge-platform/app-orch-deployment
[Tenant Controller]: https://github.com/open-edge-platform/app-orch-tenant-controller
[Cluster Extensions]: https://github.com/open-edge-platform/cluster-extensions
[Platform Services]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/platform/index.html
[Contributor Guide]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html
[Troubleshooting]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/troubleshooting/index.html
[Contact us]: https://github.com/open-edge-platform
[Edge Orchestrator Community]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/index.html
[Apache 2.0 License]: LICENSES/Apache-2.0.txt
[Developer Guide App Orch Tutorial]: app-orch-tutorials/developer-guide-tutorial/README.md
