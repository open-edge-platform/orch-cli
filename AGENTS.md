<!--
SPDX-FileCopyrightText: (C) 2026 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# orch-cli - Agent Context File
> CLI tool for managing edge infrastructure and orchestrator resources (hosts, clusters, sites, etc.) via REST APIs and Keycloak authentication.

## Platform Overview
- Go (>=1.20) CLI application, monorepo structure.
- Cobra (CLI), Viper (config), Keycloak (OIDC), REST API clients (OpenAPI-generated).

## Component Map
| Component | Purpose |
|---|---|
| `cmd/orch-cli/main.go` | Main CLI entrypoint |
| `internal/cli/` | All CLI command implementations (login, host, cluster, etc.) |
| `pkg/rest/` | Auto-generated REST API clients |
| `pkg/auth/` | Keycloak and token management |
| `docs/cli/` | Markdown docs for each command |

## Build, Test, and Lint
All tasks go through the Makefile. Do not run `go build` or `go test` directly unless debugging.

```sh
make build        # builds to build/_output/orch-cli
make install      # install to PATH
make test         # go test ./... -race
make coverage     # test with coverage
make lint         # golangci-lint + yamllint
make cli-docs     # generate CLI documentation
make fetch-openapi && make rest-client-gen  # regenerate REST clients
make clean        # remove build artifacts
make license      # check license compliance
```

Prerequisites: Go 1.20+, make, golangci-lint, yamllint, markdownlint, oapi-codegen.

## Available Skills
Skills are in `.claude/skills/`. Use trigger phrases to activate:
- `build`: Build and install the CLI binary.
- `configure`: Set up the CLI to point at an orchestrator URL and project.
- `login`: Authenticate the CLI against an orchestrator instance.
- `setup-infrastructure`: Create regions, sites, and SSH keys (prerequisite for onboarding).
- `setup-amt`: Configure AMT profiles for out-of-band management (ACM/CCM).
- `onboard-hosts`: Register and provision hosts via CSV import.
- `host-power`: Power on/off/cycle/reset hosts individually or in bulk.
- `user-management`: Create/delete users, set passwords, manage groups and realm roles.

## Skill Execution Order (MUST follow for all skills)
Every skill execution follows this mandatory sequence:
1. Collect required inputs
2. Run all preconditions
3. Execute build/deployment steps
4. Run validation checks
5. Report results and propose rollback if needed

Do not skip preconditions or validation.

## Constraints
- Do not store secrets or tokens in plaintext. Use environment variables or OS keyring.
- Ask for confirmation before any `sudo` or destructive step.
- Never infer credentials, certificates, SSH keys, or secrets.
- Always run `go mod tidy` if dependencies change.
- Generated code (OpenAPI clients) must be updated via Makefile targets, not manually.
- Always run `make lint` and `make test` before submitting changes.
- Always report artifact paths and validation results at the end of skill runs.

## Quick Tryout Prompts
1. `Use the login skill to connect to my orchestrator. Ask me for missing inputs first.`
2. `Run only preconditions for login — do not execute yet.`
3. `Create a dry-run plan for login with commands and expected results.`
