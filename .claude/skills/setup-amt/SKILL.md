<!--
SPDX-FileCopyrightText: (C) 2026 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

## Metadata
skill_id: setup-amt
component: cli
estimated_time: 5 minutes
requires_sudo: false
requires_network: true

## Trigger Phrases
 - create amt profile
 - configure amt
 - set up amt
 - amt provisioning
 - amt domain profile

## Control Modes: CCM vs ACM

Intel AMT supports two provisioning control modes that determine the level of management access:

### CCM — Client Control Mode
- **User consent required** for remote actions (power control, KVM, etc.). The physical user at the machine must approve each operation via a consent prompt on the display.
- **No certificate or domain required.** CCM can be provisioned without an AMT domain profile.
- Suitable for environments where the end user retains physical control and must authorize remote management.

### ACM — Admin Control Mode
- **No user consent required.** Remote power actions, KVM, and other OOB operations execute without any prompt at the physical machine.
- **Requires a provisioning certificate (PFX) and a matching domain suffix.** The certificate must be issued by a CA that is pre-loaded in the AMT firmware's trusted root store. The domain suffix must match the network's DNS suffix so AMT can validate the provisioning server.
- Suitable for headless/unattended deployments (labs, data centers, edge sites) where no operator is present to grant consent.

### Which to choose
| Scenario | Mode | AMT Profile Required? |
|---|---|---|
| Unattended edge site, no physical operator | ACM | Yes (cert + domain) |
| Lab machines with operator present | CCM | No |
| Mixed fleet — some attended, some not | Both | Yes for ACM hosts |

## Required Inputs (ACM — full profile creation)
 - profile name
 - PFX certificate file path (must be from a CA trusted by AMT firmware)
 - certificate password (prompted interactively if not provided)
 - certificate format: `string` or `raw`
 - domain suffix (must match the network DNS suffix, e.g. `example.com`)

## Required Inputs (CCM — no profile needed)
 - host resource ID
 - Only the control mode flag: `--control-mode client`

## Preconditions (all modes)
 - [ ] CLI is configured and authenticated (see `configure` and `login` skills)
 - [ ] OOB (Out-of-Band) feature is enabled on the orchestrator (`orch-cli list features`)

## Preconditions (ACM only)
 - [ ] PFX certificate file exists on disk

## Steps

### ACM Setup (requires AMT profile)
1. Verify the OOB feature is available:
   - `orch-cli list features` (confirm OOB/AMT is enabled)
2. Verify the certificate file exists:
   - Check the provided path is accessible
3. Create the AMT profile:
   - `orch-cli create amtprofile <NAME> --cert <PATH_TO_PFX> --cert-format <string|raw> --domain-suffix <DOMAIN>`
   - The CLI will prompt for the certificate password interactively. To provide it inline (caution: visible in shell history): add `--cert-pass <PASSWORD>`.
4. Verify the profile was created:
   - `orch-cli list amtprofile`
   - `orch-cli get amtprofile <NAME>`
5. Provision hosts in ACM:
   - `orch-cli set host <HOST_ID> --amt-state provisioned --control-mode admin`

### CCM Setup (no profile needed)
1. Verify the OOB feature is available:
   - `orch-cli list features`
2. Provision hosts in CCM:
   - `orch-cli set host <HOST_ID> --amt-state provisioned --control-mode client`

### Bulk Provisioning (either mode)
Bulk AMT provisioning is supported via CSV import:

1. Dry-run to validate:
   - `orch-cli set host --import-from-csv <FILE> --dry-run`
2. Execute:
   - `orch-cli set host --import-from-csv <FILE>`

CSV example:

```
Name,ResourceID,DesiredAmtState,ControlMode
host-lab-01,host-1234abcd,provisioned,admin
host-lab-02,host-5678efgh,provisioned,client
```

## Behavior Notes
- AMT profiles are domain-level certificates that bind a provisioning server to a DNS domain. They are only required for ACM.
- In CCM, the physical user sees a consent code on the host's display that must be entered to authorize remote operations. This makes CCM unsuitable for headless deployments.
- In ACM, no consent is needed — the trust is established by the certificate chain. The cert must come from a CA embedded in AMT's firmware trust store (Intel maintains this list).
- A host must be AMT-provisioned (in either mode) before power actions will work via the `host-power` skill.
- The `AmtDnsSuffix` field on a host is relevant to ACM only — it must match the domain suffix configured in the AMT profile.
- The certificate password can be provided inline (`--cert-pass value`) or prompted interactively. Prefer the interactive prompt to avoid shell history exposure.
- Certificate format `string` is base64-encoded PFX; `raw` is binary PFX.

## Troubleshooting
| Symptom | Cause | Fix |
|---|---|---|
| "certificate path must be provided" | Missing --cert flag or empty value | Provide the full path to the PFX file |
| "certificate password must be provided" | Empty --cert-pass and no interactive prompt | Provide password via flag or let the CLI prompt |
| "certificate format must be provided" | Missing or invalid --cert-format | Use `string` or `raw` |
| "Not Found" on delete | Profile name doesn't exist | Run `orch-cli list amtprofile` to verify the name |
| OOB feature not available | Feature not installed on orchestrator | Contact orchestrator admin to enable OOB |

## Safety Rules
- Never infer or guess certificate passwords. Prefer the interactive prompt.
- Never print certificate contents or passwords in output.
- Confirm the domain suffix with the user — it must match the certificate's domain.
