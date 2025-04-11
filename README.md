<!--
SPDX-FileCopyrightText: (C) 2025 Intel Corporation

SPDX-License-Identifier: Apache-2.0
-->

# Catalog CLI

The catalog service CLI is a standalone utility which interacts with the Application Catalog and deployment
services using their REST API endpoints and allows the operator to manage various Catalog resources
from the command line.

The supported CLI usage allows user to `create`, `get`, `list`, `set` and `delete` the following
Catalog entities:
* publishers
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
