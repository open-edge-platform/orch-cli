<!--
SPDX-FileCopyrightText: (C) 2026 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

## Metadata
skill_id: onboard-hosts
component: cli
estimated_time: 10 minutes
requires_sudo: false
requires_network: true

## Trigger Phrases
 - onboard host
 - onboard hosts
 - provision host
 - create host
 - import hosts from csv

## Required Inputs

### Bulk (CSV)
 - CSV file with host definitions

### Single Host (direct flags)
 - hostname (positional argument)
 - at least one of: --serial or --uuid
 - --os-profile (required when provisioning feature is enabled)
 - --site (required when provisioning feature is enabled)

## Optional Inputs
 - OS profile override (--os-profile)
 - remote user / SSH key override (--remote-user)
 - metadata key-value pairs (--metadata)
 - security feature toggle (--secure true|false)
 - LVM size (--lvm-size)
 - cloud-init custom config (--cloud-init)
 - cluster deployment settings (--cluster-deploy, --cluster-template, --cluster-config)

## Preconditions
 - [ ] CLI is configured and authenticated (see `configure` and `login` skills)

### Additional preconditions when provisioning feature is enabled
 - [ ] Infrastructure exists: region, site (see `setup-infrastructure` skill — will be invoked if missing)
 - [ ] At least one OS profile is available (`orch-cli list osprofile`)
 - [ ] SSH key exists if remote access is needed (optional)

## Steps

### Single Host (direct flags)
1. Check prerequisites exist:
   - `orch-cli list site` (confirm target site exists — invoke `setup-infrastructure` if missing)
   - `orch-cli list osprofile` (confirm OS profile is available — if none exist, inform the user; OS profiles are managed by the orchestrator and cannot be created via CLI)
2. Dry-run to validate inputs (at least one of --serial or --uuid is required):
   - `orch-cli create host <HOSTNAME> --serial <SERIAL> --os-profile <PROFILE> --site <SITE_ID> --dry-run`
3. Create the host:
   - `orch-cli create host <HOSTNAME> --serial <SERIAL> --os-profile <PROFILE> --site <SITE_ID>`
   - Or with UUID: `orch-cli create host <HOSTNAME> --uuid <UUID> --os-profile <PROFILE> --site <SITE_ID>`
4. Verify:
   - `orch-cli list host`

### Bulk (CSV import)
1. Check prerequisites exist:
   - `orch-cli list site` (confirm target site exists — invoke `setup-infrastructure` if missing)
   - `orch-cli list osprofile` (confirm OS profile is available — if none exist, inform the user; OS profiles are managed by the orchestrator)
   - `orch-cli list sshkey` (confirm SSH key exists if remote access is needed — invoke `setup-infrastructure` if missing)
2. Generate a CSV template (if user doesn't have one):
   - `orch-cli create host --generate-csv=hosts.csv`
3. Validate the CSV with a dry run:
   - `orch-cli create host --import-from-csv hosts.csv --dry-run`
4. Create hosts from CSV:
   - `orch-cli create host --import-from-csv hosts.csv`
   - With overrides: `--site <SITE_ID> --os-profile <PROFILE> --remote-user <USER>`
5. Verify hosts were created:
   - `orch-cli list host`

## CSV Format
```
Serial,UUID,OSProfile,Site,Secure,RemoteUser,Metadata,LVMSize,CloudInitMeta,K8sEnable,K8sClusterTemplate,K8sConfig,Error - do not fill
2500JF3,4c4c4544-2046-5310-8052-cac04f515233,"Edge Microvisor Toolkit 3.0",site-c69a3c81,,localaccount-4c2c5f5a
```

Fields:
- Serial/UUID: at least one must be provided
- OSProfile: name or resource ID (mandatory)
- Site: resource ID (mandatory)
- Secure: true/false (optional)
- RemoteUser: name or resource ID (optional)
- Metadata: key=value pairs separated by & (optional)

## Behavior Notes
- `--dry-run` validates the CSV without creating any hosts. Always run this first.
- Flag overrides (--site, --os-profile, etc.) apply to ALL hosts in the CSV.
- Errors are reported per-host in the output; successful hosts are created even if others fail.
- The Error column in the CSV is output-only — do not fill it in the input file.

## Troubleshooting
| Symptom | Cause | Fix |
|---|---|---|
| "site not found" | Invalid site resource ID | Run `orch-cli list site` for valid IDs |
| "os profile not found" | Profile name mismatch | Run `orch-cli list osprofile` for exact names |
| "serial or uuid required" | Both fields empty in CSV | Provide at least one identifier per host |
| Partial failures | Some hosts have invalid data | Check error column in output, fix CSV, re-import only failed rows |

## Safety Rules
- Always run `--dry-run` before creating hosts.
- Never guess serial numbers, UUIDs, or site IDs.
- Confirm the host count with the user before running the actual import.
