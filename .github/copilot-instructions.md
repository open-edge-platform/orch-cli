# Copilot Coding Agent Onboarding Instructions for orch-cli

## High-Level Repository Overview

- **Purpose:**
  - This repository provides `orch-cli`, a command-line tool for managing edge infrastructure and orchestrator resources (hosts, clusters, sites, etc.) via REST APIs and Keycloak authentication.
- **Project Type:**
  - Go (Golang) CLI application, monorepo structure.
- **Languages/Frameworks:**
  - Go (>=1.20 recommended)
  - Cobra (CLI), Viper (config), Keycloak (OIDC), REST API clients (OpenAPI-generated)
- **Repository Size:**
  - Medium (dozens of Go files, multiple subdirectories, and documentation)

## Build, Test, and Validation Instructions

### Prerequisites
- Go 1.20+ must be installed and in your PATH.
- `make` is required for build/test/lint tasks (see Makefile).
- Some targets require `golangci-lint`, `yamllint`, and `oapi-codegen` (install via `go install` or system package manager).

### Bootstrap/Setup
- No explicit bootstrap step. Ensure Go and Make are installed.
- Always run `go mod tidy` if dependencies change.

### Build
- To build the CLI:
  ```sh
  make build
  ```
  - Builds the binary to `build/_output/orch-cli`.
  - Install to your PATH with `make install`.

### Test
- To run all tests:
  ```sh
  make test
  ```
  - Runs `go test ./... -race`.
  - For coverage:
    ```sh
    make coverage
    ```

### Lint
- To lint Go and YAML files:
  ```sh
  make lint
  ```
  - Requires `golangci-lint` and `yamllint`.

### Documentation Generation
- Generate CLI docs:
  ```sh
  make cli-docs
  ```

### OpenAPI Client Generation
- Fetch OpenAPI specs and generate REST clients:
  ```sh
  make fetch-openapi
  make rest-client-gen
  ```

### Clean
- Clean build artifacts:
  ```sh
  make clean
  ```

### License Compliance
- Check license compliance:
  ```sh
  make license
  ```

### Common Issues & Workarounds
- If Go module errors occur, run `go mod tidy`.
- If `make` fails due to missing tools, install them as described above.
- If install fails due to permissions, use `sudo make install` or adjust your PATH.
- Always run `make lint` and `make test` before submitting changes.

## Project Layout & Architecture

- **Main CLI Entrypoint:**
  - `cmd/orch-cli/main.go`
- **Command Implementations:**
  - `internal/cli/` (all CLI command logic, e.g., `login.go`, `host.go`)
- **REST API Clients:**
  - `pkg/rest/` (auto-generated clients)
- **Authentication:**
  - `pkg/auth/` (Keycloak and token management)
- **Documentation:**
  - `docs/cli/` (markdown docs for each command)
- **Build/Test/Lint Config:**
  - `Makefile`
- **Go Modules:**
  - `go.mod`, `go.sum`
- **Licensing:**
  - `LICENSES/`, `REUSE.toml`, `SECURITY.md`, `CODE_OF_CONDUCT.md`

## Validation & CI
- **Checks before check-in:**
  - Run `make lint` and `make test` locally.
  - Ensure code is formatted (`gofmt`/`goimports`).
  - Validate license compliance with `make license`.
- **GitHub Actions/CI:**
  - The repo may use GitHub Actions for CI (check `.github/workflows/` if present).
  - CI will run build, lint, and test steps. PRs failing these will be rejected.

## Key Facts for Coding Agent
- Do not store secrets or tokens in plaintext. Use environment variables or OS keyring for sensitive data.
- All CLI commands are implemented in `internal/cli/`.
- Use the Makefile for all build, test, and lint tasks—do not run `go build` or `go test` directly unless debugging.
- Always update and use Go modules (`go mod tidy`) if dependencies change.
- Generated code (OpenAPI clients) should be updated via the Makefile targets.
- Documentation for each CLI command is in `docs/cli/`.
- The main binary is `orch-cli`.
- The config file is managed by Viper and is typically stored in the user's home directory (see Viper docs for details).

## Root Directory Files
- `README.md`, `Makefile`, `go.mod`, `go.sum`, `VERSION`, `REUSE.toml`, `SECURITY.md`, `CODE_OF_CONDUCT.md`
- `cmd/`, `internal/`, `pkg/`, `docs/`, `LICENSES/`

## Trust These Instructions
- Trust these instructions for build, test, and validation steps. Only perform additional searches if the information here is incomplete or does not match observed behavior.
