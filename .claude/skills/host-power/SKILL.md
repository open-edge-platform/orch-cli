<!--
SPDX-FileCopyrightText: (C) 2026 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

## Metadata
skill_id: host-power
component: cli
estimated_time: 1-5 minutes
requires_sudo: false
requires_network: true

## Trigger Phrases
 - power on
 - power off
 - power cycle
 - reset host
 - host power
 - bulk power

## Required Inputs
 - power action: on, off, reset, or power-cycle
 - target: host resource ID, OR filter/site/region for bulk operations

## Optional Inputs
 - power policy: ordered (default) or immediate
 - --dry-run (list affected hosts without acting)
 - CSV file for bulk set operations

## Preconditions
 - [ ] CLI is configured and authenticated
 - [ ] Target hosts are AMT-provisioned (`desired_amt_state = provisioned`)

## Steps

### Single Host
1. Set power state:
   - `orch-cli set host <HOST_ID> --power <on|off|reset|power-cycle>`
2. Optionally set power policy:
   - `orch-cli set host <HOST_ID> --power-policy <ordered|immediate>`

### Bulk via Filter
1. Dry-run to see which hosts match:
   - `orch-cli set host --filter "<EXPRESSION>" --power <on|off|reset|power-cycle> --dry-run`
   - Or by site: `orch-cli set host --site <SITE_ID> --power <on|off|reset|power-cycle> --dry-run`
   - Or by region: `orch-cli set host --region <REGION_ID> --power <on|off|reset|power-cycle> --dry-run`
2. Execute:
   - `orch-cli set host --filter "<EXPRESSION>" --power <on|off|reset|power-cycle>`

### Bulk via CSV
1. Generate a CSV template:
   - `orch-cli set host --generate-csv=power.csv`
2. Fill in the CSV with host names, resource IDs, and desired power state.
3. Dry-run:
   - `orch-cli set host --import-from-csv power.csv --dry-run`
4. Execute:
   - `orch-cli set host --import-from-csv power.csv`

## CSV Format
```
Name,ResourceID,DesiredAmtState,ControlMode,DesiredPowerState
host-a,host-1234abcd,,,on
host-b,host-5678efgh,,,reset
```

## Behavior Notes
- Setting `desired_power_state` triggers the device controller's reconciliation loop — the actual power action happens asynchronously within seconds.
- The CLI iterates sequentially over matched hosts (one API call per host).
- `--dry-run` lists which hosts would be affected without making changes.
- Power actions require the host to be AMT-provisioned. Non-provisioned hosts are skipped with an error.
- Power policy `ordered` (default) gracefully shuts down the OS before power-off; `immediate` cuts power without warning.

## Troubleshooting
| Symptom | Cause | Fix |
|---|---|---|
| "AMT not provisioned" | Host hasn't completed AMT setup | Run `orch-cli set host <ID> --amt-state provisioned` first |
| No hosts matched | Filter too restrictive | Test filter with `orch-cli list host --filter "..."` first |
| Power state unchanged | Desired already matches current | Check `orch-cli get host <ID>` for current vs desired state |
| Partial completion | CLI interrupted mid-run | Re-run the same command — operation is idempotent |

## Safety Rules
- Always run `--dry-run` before bulk power operations.
- `power off` and `reset` are disruptive — confirm the host count with the user.
- Never assume AMT provisioning state; verify before issuing power commands.
