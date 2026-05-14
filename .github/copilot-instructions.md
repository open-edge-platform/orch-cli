<!--
SPDX-FileCopyrightText: (C) 2026 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# Copilot Instructions

Use [AGENTS.md](../AGENTS.md) as the repository context catalog and `.claude/skills/*/SKILL.md` as execution contracts.

## How To Select a Skill
1. Parse the user request for desired outcome.
2. Find the matching skill in [AGENTS.md](../AGENTS.md#available-skills), then read its `.claude/skills/<name>/SKILL.md` for trigger phrases and execution details.
3. Ask for missing required inputs before running commands.

For skill execution order, see [AGENTS.md](../AGENTS.md#skill-execution-order-must-follow-for-all-skills).

## Completion Criteria for Skill Runs
A run is complete only when the agent returns:
- precondition results
- validation status
- build status (if executed)
- artifact paths and names
- troubleshooting notes when failures occur
