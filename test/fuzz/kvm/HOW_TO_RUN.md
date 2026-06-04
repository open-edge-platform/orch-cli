# KVM Viewer REST API Fuzz Testing — Step-by-Step Guide

<!-- SPDX-FileCopyrightText: (C) 2026 Intel Corporation -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

## Overview

This guide explains how to run RESTler smoke-fuzz tests against the KVM Viewer
REST API (`/api/connect`, `/api/status`, `/api/disconnect`) using the FaaS
(Fuzzing-as-a-Service) Docker container.

The three components involved are:

| Component | Location |
|---|---|
| KVM fuzz target server | `internal/cli/kvm_fuzz_test.go` |
| RESTler token script | `test/fuzz/kvm/token.sh` |
| FaaS container + task | `containers.docker.fuzzing.faas/` |

---

## Prerequisites

- Docker installed and the FaaS image built (see step 1)
- Go toolchain with the `kvm` build tag available
- `task` (Taskfile) installed

---

## Step 1 — Build the FaaS Docker image (once)

```bash
cd containers.docker.fuzzing.faas
task build-faas
```

Only needs to be done once, or after changes to the FaaS container.

---

## Step 2 — Start the FaaS container

```bash
cd containers.docker.fuzzing.faas
task run-faas
```

This starts the FastAPI orchestration service on `http://localhost:8887`.
Leave this running in the background (or a dedicated terminal).

---

## Step 3 — Start the KVM fuzz target server

Open a dedicated terminal. Run from the `orch-cli/` root:

```bash
cd orch-cli
go test -run TestKVMFuzzServer -tags kvm -timeout 0 -v ./internal/cli/
```

Expected output:
```
=== RUN   TestKVMFuzzServer
    kvm_fuzz_test.go:102: KVM fuzz server listening on http://127.0.0.1:8585
    kvm_fuzz_test.go:103: X-Session-Token: 0123456789abcdef0123456789abcdef...
    kvm_fuzz_test.go:104: Waiting for SIGINT or SIGTERM to stop...
```

Keep this terminal open — the server must stay alive for the entire fuzz run.

> **Port conflict:** If port 8585 is already in use, kill the old process first:
> ```bash
> kill $(lsof -ti tcp:8585)
> ```

---

## Step 4 — Run the fuzz test

Open another terminal:

```bash
cd containers.docker.fuzzing.faas
task fuzz test=smoke \
  openapi=../orch-cli/internal/cli/testdata/kvm-rest-openapi.yaml \
  config=../orch-cli/test/fuzz/kvm/config.yml
```

### Expected success output

```
codeCounts: {}        ← no HTTP error codes (200s are not counted as errors)
bugCount: 0
final_spec_coverage: 3 / 3
rendered_requests_valid_status: 3 / 3
```

### Failure indicators

| Output | Cause |
|---|---|
| `codeCounts: {"400": N}` | Malformed auth header — check `token.sh` format (see Troubleshooting) |
| `codeCounts: {"403": N}` | Token mismatch — server token ≠ `token.sh` token (see Token section) |
| `codeCounts: {}` + `coverage 0/3` | KVM fuzz server not running on port 8585 |

---

## Token Management

### How the token works

1. `TestKVMFuzzServer` sets `srv.sessionToken` to a fixed value.
2. RESTler calls `token.sh` before each request batch to get the
   `X-Session-Token` header value.
3. The KVM server rejects requests where the header value does not match
   `sessionToken` → **403 Forbidden**.

**The token value in `token.sh` and the value used by `TestKVMFuzzServer`
must be identical.**

### Default token (recommended for fuzz runs)

No environment variable needed. Both the server and `token.sh` use the same
hardcoded default:

```
0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

Just run steps 3 and 4 without setting `KVM_FUZZ_TOKEN`.

### Using a custom token

If you need a different token value:

**Option A — change the default in both places (permanent):**

1. Edit `test/fuzz/kvm/token.sh` — update the fallback value:
   ```sh
   TOKEN="${KVM_FUZZ_TOKEN:-<your-new-64-char-hex-token>}"
   ```
2. Edit `internal/cli/kvm_fuzz_test.go` — update the constant:
   ```go
   const kvmFuzzDefaultToken = "<your-new-64-char-hex-token>"
   ```
3. Run steps 3 and 4 as normal.

**Option B — use `KVM_FUZZ_TOKEN` env var (one-off):**

> **Important:** `KVM_FUZZ_TOKEN` is read by the Go test server on the **host**.
> It is **NOT** propagated into the FaaS Docker container where `token.sh`
> runs. Using this option will cause a 403 mismatch unless you also hardcode
> the value into `token.sh` before running.

```bash
# Step 3 — start server with custom token
export KVM_FUZZ_TOKEN=<your-64-char-hex-token>
go test -run TestKVMFuzzServer -tags kvm -timeout 0 -v ./internal/cli/

# Step 4 — ALSO update token.sh to hardcode the same value, then run fuzz
```

---

## Changes Made to orch-cli (reference)

### `internal/cli/kvm_fuzz_test.go`

| Change | Reason |
|---|---|
| Renamed `kvmFuzzToken` → `kvmFuzzDefaultToken` | Clarifies it is the fallback value |
| Added `KVM_FUZZ_ADDR` env var support | Allows overriding the listen address |
| Added `KVM_FUZZ_TOKEN` env var support | Allows overriding the session token at runtime |

### `internal/cli/kvm_load_test.go`

| Change | Reason |
|---|---|
| Updated two references from `kvmFuzzToken` to `kvmFuzzDefaultToken` | Compile fix after const rename |

### `test/fuzz/kvm/token.sh`

| Change | Reason |
|---|---|
| Added `{'app1': {}}` as line 1 of output | RESTler's `parse_authentication_tokens()` calls `ast.literal_eval()` on the first line and expects a Python dict. Without it RESTler emits `AUTHORIZATION TOKEN` (literal, no colon) as the header and Go's HTTP parser returns **400**. |

The required output format for `token.sh` is:
```
{'app1': {}}
X-Session-Token: <token-value>
```

---

## Collecting Evidence / Logs

### Host-side log (pytest output)

Each run creates a timestamped log folder:
```
containers.docker.fuzzing.faas/logs/log_<HH>h_<MM>m_<SS>s_<Month>_<DD>_<YYYY>/
```

### Container-side RESTler results

Find the `app_id` UUID from the pytest DEBUG line:
```
DEBUG    Fuzz-SmokeTest:test-web-service.py:56 {'app_id': 'xxxxxxxx-...'}
```

Copy all result files to the host:
```bash
CONTAINER=$(docker ps --filter ancestor=faas --format "{{.ID}}" | head -1)
APP=<app_id>
DEST=test/fuzz/kvm/evidence

mkdir -p "$DEST"
docker cp "$CONTAINER:/restler-workdir/$APP/Test/ResponseBuckets/." "$DEST/"
LOGS=$(docker exec "$CONTAINER" sh -c "ls -d /restler-workdir/$APP/Test/RestlerResults/*/logs")
docker cp "$CONTAINER:$LOGS/." "$DEST/"
```

### Key result files

| File | Contents |
|---|---|
| `testing_summary.json` | Coverage stats: `final_spec_coverage`, `rendered_requests_valid_status`, `num_fully_valid` |
| `runSummary.json` | `bugCount`, `codeCounts`, `errorBuckets` |
| `speccov.json` | Per-endpoint status codes with sample request/response |
| `network.testing.*.txt` | Full HTTP request/response traces (auth token redacted as `_OMITTED_AUTH_TOKEN_`) |
| `main.txt` | RESTler engine run log with final statistics |
| `PayloadBodyChecker.*.txt` | Payload mutation checker results |

### Creating a ZIP for upload

```bash
cd orch-cli/test/fuzz
zip -r kvm-evidences.zip kvm/
```

---

## Troubleshooting

### `codeCounts: {"400": N}` — Bad Request

`token.sh` output is not in the correct two-line format. RESTler sends
`AUTHORIZATION TOKEN` (a literal placeholder with no colon) which Go's HTTP
parser rejects.

**Fix:** Ensure `token.sh` outputs exactly:
```
{'app1': {}}
X-Session-Token: <token>
```

### `codeCounts: {"403": N}` — Forbidden

The token value sent by RESTler does not match the value the server is using.

**Fix:** Stop the server (`Ctrl+C`), then restart step 3 **without**
`KVM_FUZZ_TOKEN` set. The default token in `token.sh` and the server will then
match.

```bash
unset KVM_FUZZ_TOKEN
go test -run TestKVMFuzzServer -tags kvm -timeout 0 -v ./internal/cli/
```

### `address already in use` on port 8585

```bash
kill $(lsof -ti tcp:8585)
```

### FaaS container not reachable

```bash
docker ps --filter ancestor=faas
# If empty, restart:
cd containers.docker.fuzzing.faas && task run-faas
```
