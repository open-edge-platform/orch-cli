<!--
SPDX-FileCopyrightText: (C) 2026 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# Claude Skill Discovery and Execution Rules

Use this file with `AGENTS.md` as the source of truth for this repository.

## Skill Discovery
1. Discover skills from `.claude/skills/`.
2. Match user intent to trigger phrases in each `SKILL.md`.
3. If multiple skills could match, ask the user to choose.

For skill execution order, see [AGENTS.md](AGENTS.md#skill-execution-order-must-follow-for-all-skills).
