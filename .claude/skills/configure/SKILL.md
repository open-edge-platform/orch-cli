<!--
SPDX-FileCopyrightText: (C) 2026 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

## Metadata
skill_id: configure
component: cli
estimated_time: 30 seconds
requires_sudo: false
requires_network: false

## Trigger Phrases
 - configure
 - set up the cli
 - point at orchestrator
 - set api endpoint
 - set project
 - switch project

## Required Inputs
 - orchestrator URL (the cluster FQDN, e.g. `mycluster.example.com`)

## Optional Inputs
 - project name

## Preconditions
 - [ ] CLI is built and installed

## Steps
1. Initialize config file if it does not already exist:
   - `orch-cli config init`
2. Set the API endpoint:
   - `orch-cli config set api-endpoint https://api.<CLUSTER_FQDN>`
3. If user provided a project name, set it:
   - `orch-cli config set project <PROJECT>`
4. Verify the configuration:
   - `orch-cli config get api-endpoint`
   - `orch-cli config get project` (if set)

## Supported Config Keys
| Key | Purpose |
|---|---|
| `api-endpoint` | Base URL for the orchestrator API |
| `project` | Active project scope for all commands |

## Behavior Notes
- The config file is stored in the user's home directory (managed by Viper).
- `config init` is idempotent — safe to run even if config already exists.
- Setting a project scopes all subsequent commands to that project. Use `config delete project` to clear it.
- The api-endpoint must include the `api.` prefix: `https://api.<CLUSTER_FQDN>`.

## Troubleshooting
| Symptom | Cause | Fix |
|---|---|---|
| "key not supported" | Unsupported config key | Only `api-endpoint` and `project` are valid |
| Commands fail after config | Wrong URL format | Ensure `https://api.` prefix, no trailing path |
| Wrong project context | Stale project value | Run `orch-cli config get project` to check, `config delete project` to clear |

## Safety Rules
- Never overwrite existing config without confirming the user's intent.
- Never infer or guess the orchestrator URL — always ask.
