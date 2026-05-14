<!--
SPDX-FileCopyrightText: (C) 2026 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

## Metadata
skill_id: login
component: cli
estimated_time: 1 minute
requires_sudo: false
requires_network: true

## Trigger Phrases
 - log in
 - login
 - connect to orchestrator
 - authenticate

## Required Inputs
 - orchestrator URL
 - username

## Optional Inputs
 - project name
 - keycloak endpoint (auto-derived from orchestrator URL by default)
 - client-id (defaults to "system-client")

## Preconditions
 - [ ] CLI is built and installed
 - [ ] Config file exists (`orch-cli config init` if not)

## Steps
1. Initialize config if needed:
   - `orch-cli config init`
2. Configure CLI with orchestrator URL:
   - `orch-cli config set api-endpoint https://api.<ORCHESTRATOR-BASE-URL>`
3. Ask the user if they would like to select a specific project. If yes, configure the project:
   - `orch-cli config set project <PROJECT>`
4. Log in (password will be prompted interactively — do NOT pass it as an argument):
   - `orch-cli login <USERNAME>`
5. Verify the CLI can connect to the orchestrator:
   - `orch-cli list features`

## Behavior Notes
- If already logged in, the CLI automatically logs out before re-authenticating. No need to run `orch-cli logout` first.
- After login, the CLI loads feature flags from the orchestrator to determine available commands. A partial feature set is normal — it reflects the orchestrator's installed components, not a failure.

## Troubleshooting
| Symptom | Cause | Fix |
|---|---|---|
| 401 Unauthorized | Wrong username or password | Verify credentials and retry |
| Keycloak well-known endpoint unreachable | Bad api-endpoint or network issue | Check URL format and connectivity |
| "failed to determine keycloak endpoint" | api-endpoint missing subdomain (needs `api.<domain>`) | Use full URL: `https://api.<CLUSTER_FQDN>` |
| "Edge Orchestrator Component Status service info not available" | Older orchestrator without info endpoint | Non-fatal — features default to enabled for backward compatibility |

## Safety Rules
- Never pass passwords as command-line arguments. Let the CLI prompt interactively.
- Never infer credentials, keys, or secrets.
- Never print full private key contents or tokens unless the user requests --show-token.
- Stop on precondition or validation failure and provide next-action guidance.
