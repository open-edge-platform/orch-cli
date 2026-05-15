<!--
SPDX-FileCopyrightText: (C) 2026 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

## Metadata
skill_id: user-management
component: cli
estimated_time: 2-5 minutes
requires_sudo: false
requires_network: true

## Trigger Phrases
 - create user
 - delete user
 - list users
 - set password
 - add user to group
 - remove user from group
 - assign role
 - manage users

## Required Inputs
 - username (for all operations except list)

## Optional Inputs
 - email, first-name, last-name (for create)
 - password (prompted interactively if --password flag is used without a value)
 - --temporary-password (user must change on first login)
 - --disabled (create user in disabled state)
 - group names (--add-group, --remove-group)
 - realm role names (--add-realm-role, --remove-realm-role)
 - --realm (defaults to "master")

## Preconditions
 - [ ] CLI is configured and authenticated
 - [ ] Multitenancy feature is enabled on the orchestrator

## Steps

### List Users
- `orch-cli list users`
- With realm: `orch-cli list users --realm master`

### Get User Details
- `orch-cli get user <USERNAME>`
- With groups: `orch-cli get user <USERNAME> --groups`
- With roles: `orch-cli get user <USERNAME> --roles`
- Both: `orch-cli get user <USERNAME> --groups --roles`

### Create User
1. Create the user:
   - `orch-cli create user <USERNAME> --email <EMAIL> --first-name <FIRST> --last-name <LAST>`
2. Optionally set password (will prompt interactively):
   - `orch-cli create user <USERNAME> --password`
   - Or via environment variable: `ORCH_PASSWORD=<value> orch-cli create user <USERNAME> --password`

### Set Password
- Interactive prompt: `orch-cli set user <USERNAME> --password`
- Via environment variable: `ORCH_PASSWORD=<value> orch-cli set user <USERNAME> --password`
- Temporary (must change on first login): `orch-cli set user <USERNAME> --password --temporary-password`

### Manage Group Membership
- Add to group: `orch-cli set user <USERNAME> --add-group <GROUP_NAME>`
- Remove from group: `orch-cli set user <USERNAME> --remove-group <GROUP_NAME>`
- Multiple in one command: `orch-cli set user <USERNAME> --add-group edge-manager-group --remove-group edge-operator-group`

### Manage Realm Roles
- Assign role: `orch-cli set user <USERNAME> --add-realm-role <ROLE_NAME>`
- Remove role: `orch-cli set user <USERNAME> --remove-realm-role <ROLE_NAME>`

### Grant Project Membership
Project membership is controlled via a computed realm role with the pattern `{ORG_UID}_{PROJ_UID}_m`.

To construct the role name:
1. Get the organization UID: `orch-cli get organization <ORG_NAME>` (look for "UID" in output)
2. Get the project UID: `orch-cli get project <PROJECT_NAME>` (look for "UID" in output)
3. Combine: `{ORG_UID}_{PROJ_UID}_m`

Example:
```
# Organization UID: org-abc123
# Project UID: proj-def456
# Resulting role: org-abc123_proj-def456_m

orch-cli set user sample-user --add-realm-role "org-abc123_proj-def456_m"
```

### Delete User
- `orch-cli delete user <USERNAME>`

## Password Resolution Order
1. Inline flag value (`--password="value"`) — visible in shell history, use with caution
2. Environment variable (`ORCH_PASSWORD`)
3. Interactive terminal prompt (most secure)

## Behavior Notes
- User commands talk directly to the Keycloak Admin REST API.
- Group and role names are resolved by exact match. Use `orch-cli list users` or `orch-cli get user <NAME> --groups --roles` to see current assignments.
- The `--realm` flag defaults to `master` (the Edge Orchestrator's realm).
- Multiple group/role changes can be combined in a single `set user` command.
- Commands are gated behind the Multitenancy feature flag.

## Troubleshooting
| Symptom | Cause | Fix |
|---|---|---|
| "command not found" or not listed | Multitenancy feature not enabled | Verify with `orch-cli list features` |
| "group not found" | Exact group name mismatch | Run `orch-cli list groups` to see available names (they may include org/project UUID prefixes) |
| "realm role not found" | Role name doesn't exist | Realm roles follow the pattern `${ORG_UID}_${PROJ_UID}_m` |
| "not authorized" | Insufficient admin privileges | The logged-in user must have Keycloak admin access |
| "user created but failed to set password" | Password policy violation | Check Keycloak realm password policies |

## Safety Rules
- Never pass passwords as inline flag values in shared environments. Prefer interactive prompt or ORCH_PASSWORD env var.
- Never infer or guess usernames, group names, or role names.
- Confirm before deleting users.
