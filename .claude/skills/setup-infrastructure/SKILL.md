<!--
SPDX-FileCopyrightText: (C) 2026 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

## Metadata
skill_id: setup-infrastructure
component: cli
estimated_time: 5 minutes
requires_sudo: false
requires_network: true

## Trigger Phrases
 - create region
 - create site
 - create ssh key
 - set up infrastructure
 - prepare for onboarding

## Required Inputs
 - region name and type (country/state/county/region/city)
 - site name
 - SSH public key file path

## Optional Inputs
 - parent region (for sub-regions)
 - site latitude and longitude
 - region to assign the site to

## Preconditions
 - [ ] CLI is configured and authenticated (see `configure` and `login` skills)

## Steps
1. Create a region:
   - `orch-cli create region <NAME> --type <TYPE>`
   - If a sub-region: `orch-cli create region <NAME> --type <TYPE> --parent <PARENT_REGION_ID>`
2. Create a site in the region:
   - `orch-cli create site <NAME> --region <REGION_ID>`
   - Optional: `--latitude <LAT> --longitude <LON>`
3. Create an SSH key for remote access:
   - `orch-cli create sshkey <NAME> <PUBLIC_KEY_FILE>`
4. Verify all resources were created:
   - `orch-cli list region`
   - `orch-cli list site`
   - `orch-cli list sshkey`

## Behavior Notes
- Region types are hierarchical: country > state > county > region > city.
- Sites must belong to a region. Get the region resource ID from step 1 output or `orch-cli list region`.
- SSH keys reference a public key file (e.g. `~/.ssh/id_rsa.pub`). The file must exist.
- All resources are scoped to the currently configured project.

## Troubleshooting
| Symptom | Cause | Fix |
|---|---|---|
| "not authorized" | Token expired | Re-run `orch-cli login` |
| "region not found" on site create | Wrong region ID | Run `orch-cli list region` to get the correct ID |
| "file not found" on sshkey create | Bad key path | Verify the public key file path exists |

## Safety Rules
- Never generate or infer SSH keys. Always use an existing public key file provided by the user.
- Confirm the region/site naming with the user before creating.
