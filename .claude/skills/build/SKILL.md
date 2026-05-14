<!--
SPDX-FileCopyrightText: (C) 2026 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

## Metadata
skill_id: build
component: cli
estimated_time: 2 minutes
requires_sudo: false
requires_network: true

## Trigger Phrases
 - build the cli
 - install the cli
 - compile
 - make build
 - make install

## Required Inputs
 - None (all paths are defined in the Makefile)

## Optional Inputs
 - install path (defaults to /usr/local/bin)

## Preconditions
 - [ ] Go 1.26+ is installed (`go version`)
 - [ ] make is installed (`make --version`)

## Steps
1. Ensure Go modules are up to date:
   - `go mod tidy`
2. Build the binary:
   - `make build`
3. If the user wants to install, install to PATH:
   - `make install`
   - (requires sudo — confirm before running)
4. Verify the installed binary:
   - `orch-cli --help`

## Behavior Notes
- `make build` outputs the binary to `build/_output/orch-cli`.
- `make install` copies the binary to `/usr/local/bin` and requires sudo.
- The build includes hardened flags: spectre mitigations, trimpath, static linking.
- The VERSION file at the repo root determines the embedded version string.

## Troubleshooting
| Symptom | Cause | Fix |
|---|---|---|
| Go module errors | Dependency mismatch | Run `go mod tidy` |
| Missing golangci-lint | Not installed | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| Permission denied on install | Missing sudo | Run `sudo make install` or change INSTALL_PATH |

## Safety Rules
- Confirm before running `make install` (requires sudo).
- Never modify the Makefile unless explicitly asked.
