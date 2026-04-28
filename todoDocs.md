# Documentation TODO

## Filtering (Deployment Packages)

Two independent filter flags are available on `list deployment-packages`:

### `--filter` — API-side filter (all output types)
- Sent to the server; reduces data returned before any local processing.
- Uses **JSON field names** (camelCase).
- Syntax follows [Google AIP-160](https://google.aip.dev/160): `field=value`, `field!='value'`, `field~'regex'`.
- Supported operators: `=`, `!=`, `>`, `<`, `>=`, `<=`, `~` (regex).
- Multiple conditions are comma-separated (AND logic).
- Not all model fields are supported by the API (e.g. `isDeployed` is **not** a supported API filter — use `--output-filter` for that).

**Examples:**
```bash
# API prefix match on name
--filter "name=pg-test-4"

# API exact match
--filter "name='pg-test-46'"

# API combined (kind + name)
--filter "kind='KIND_NORMAL',name=pg-test"
```

### `--output-filter` — Client-side filter (table output only)
- Applied locally after all data is fetched from the API.
- Uses **Go struct field names** (PascalCase), but **now supports case-insensitive matching**.
- Accepts: `Name`, `name`, `displayName`, `display_name` — all work!
- Same AIP-160 syntax as `--filter`.
- Works on any field in the model, including ones the API does not support filtering on (e.g. `IsDeployed`, `Kind`).
- Ignored when `--output-type json` or `--output-type yaml` is used.

**Field name mapping (JSON → Go struct):**

| `--filter` (API / JSON) | `--output-filter` (Go struct) | Aliases (all work) |
|---|---|---|
| `name` | `Name` | `name`, `Name` |
| `displayName` | `DisplayName` | `displayname`, `display_name`, `DisplayName` |
| `version` | `Version` | `version`, `Version` |
| `kind` | `Kind` | `kind`, `Kind` |
| `isDeployed` | `IsDeployed` | `isdeployed`, `is_deployed`, `IsDeployed` |
| `defaultProfileName` | `DefaultProfileName` | `defaultprofilename`, `default_profile_name`, `DefaultProfileName` |
| `description` | `Description` | `description`, `Description` |

**Examples:**
```bash
# Case-insensitive exact match (all equivalent)
--output-filter "Name=pg-test-46"
--output-filter "name=pg-test-46"
--output-filter "NAME=pg-test-46"

# Quotes are stripped automatically
--output-filter "name='pg-test-46'"

# Regex match (use .* not *)
--output-filter "name~'pg-test-4.'"        # pg-test-40 through pg-test-49
--output-filter "name~'pg-test-4.*'"       # pg-test-4 and anything starting with pg-test-4
--output-filter "name~pg-test-4."          # same, quotes optional

# Case-insensitive snake_case also works
--output-filter "display_name~test"
--output-filter "is_deployed=true"

# Filter on fields not supported by the API
--output-filter "IsDeployed=true"
--output-filter "kind~'KIND_NORMAL'"

# Combined API filter + client filter
--filter "name=pg-test-4" --output-filter "kind='KIND_NORMAL'"
--filter "name=pg-test-4" --output-filter "is_deployed=false"
```

**Important regex note:** `*` in regex means "zero or more of the preceding character", not a glob wildcard.
- `Name~'pg-test-4*'` → matches `pg-test-` (zero `4`s) or `pg-test-4` (one `4`) — probably not what you want
- `Name~'pg-test-4.*'` → matches `pg-test-4` followed by anything — correct glob equivalent

### Quote stripping behaviour
Both `--filter` and `--output-filter` strip one layer of surrounding single or double quotes from values:
- `Name='pg-test-46'` → value is `pg-test-46`
- `Name="pg-test-46"` → value is `pg-test-46`
- `Name=pg-test-46` → value is `pg-test-46`

### ⚠️ CRITICAL: String Value Syntax Difference

**Server-side `--filter` (API) requires quotes around string values:**
```bash
# ✓ Correct - string values MUST be quoted
./build/_output/orch-cli list providers --filter 'name="infra_onboarding"'

# ✗ Incorrect - missing quotes causes API error
./build/_output/orch-cli list providers --filter 'name=infra_onboarding'
# Error: undeclared identifier 'infra_onboarding'
```

**Client-side `--output-filter` does NOT require quotes:**
```bash
# ✓ Correct - quotes optional for client filter
./build/_output/orch-cli list providers --output-filter 'Name=infra_onboarding'
./build/_output/orch-cli list providers --output-filter 'Name="infra_onboarding"'
```

**Why the difference?**
- `--filter` uses strict AIP-160 parser on the server (treats unquoted values as identifiers/variables)
- `--output-filter` uses relaxed client-side parser (assumes string values by default)

This applies to **all resources** with server-side filtering: providers, sshkeys, customconfig, osprofile, osupdatepolicy, hosts, etc.

### Field Name Normalization (New!)
The `--output-filter` flag now supports **case-insensitive field names** with automatic normalization:
- **PascalCase** (Go struct names): `Name`, `DisplayName`, `IsDeployed`
- **camelCase** (JSON field names): `name`, `displayName`, `isDeployed`
- **snake_case**: `name`, `display_name`, `is_deployed`
- **lowercase**: `name`, `displayname`, `isdeployed`

All variants map to the correct Go struct field. This matches the behavior of `--order-by` flag.

---

## Order-By Behavior Note (Deployment Packages)

- Table output (`--output-type table`) supports direction prefixes in `--order-by`: `+field` (ascending), `-field` (descending).
- JSON/YAML output (`--output-type json|yaml`) now supports only:
  - `field`
  - `+field`
  - `-field`
- For JSON/YAML, the CLI converts `+field` / `-field` to API-friendly ordering internally.
- JSON/YAML does **not** support keyword forms:
  - `field asc`
  - `field desc`
  - `asc field`
  - `desc field`
- JSON/YAML does **not** support symbolic `<` or `>` prefixes.
- Invalid `--order-by` terms should return a clear usage error.
- The order-by validation probe uses **no filter** (`Filter: nil`) to avoid misclassifying API filter errors as invalid order-by fields.

---

## Output Template Override Note (Deployment Packages)

- Custom output template overrides are now **table-only** for deployment packages.
- `--output-template` and `--output-template-file` apply only to table output.
- `--verbose` always uses the built-in inspect template and ignores custom template overrides.
- Environment variable override currently uses `ORCH_CLI_DEPLOYMENT_PACKAGE_OUTPUT_TEMPLATE` and applies only to table output.
- Follow-up docs decision: consider renaming env var to something explicit like `ORCH_CLI_DEPLOYMENT_PACKAGE_TABLE_TEMPLATE`.

---

## Custom Output Template Syntax (Applications and other resources)

### Template Function Usage: `str` vs Direct Field Access

**IMPORTANT:** The `str` template function is ONLY for **pointer fields** (`*string`), NOT regular string fields.

#### Working Examples:
```bash
# Regular string fields - use direct access
./build/_output/orch-cli list applications --filter name=metal \
  --output-template 'table{{.Name}}\t{{.Version}}\t{{.HelmRegistryName}}'

# Pointer string fields - use str function
./build/_output/orch-cli list applications --filter name=metal \
  --output-template 'table{{.Name}}\t{{str .DisplayName}}\t{{str .Description}}'
```

#### Field Type Reference (Applications):
| Field | Type | Template Access |
|---|---|---|
| `Name` | `string` | `{{.Name}}` |
| `Version` | `string` | `{{.Version}}` |
| `HelmRegistryName` | `string` | `{{.HelmRegistryName}}` |
| `ChartName` | `string` | `{{.ChartName}}` |
| `DisplayName` | `*string` | `{{str .DisplayName}}` |
| `Description` | `*string` | `{{str .Description}}` |
| `DefaultProfileName` | `*string` | `{{str .DefaultProfileName}}` |
| `ImageRegistryName` | `*string` | `{{str .ImageRegistryName}}` |

#### Error Encountered:
```bash
# WRONG - using str on non-pointer field
--output-template 'table{{str .Name}}\t{{str .Version}}\t{{str .HelmRegistryName}}'

# Error: "wrong type for value; expected *string; got string"
```

### Template Functions Available:
- `str .Field` — Nil-safe deref for `*string` (returns `""` for nil)
- `deref .Field` — Nil-safe deref for any pointer type (returns zero value for nil)
- `timestamp .Field` — Format timestamp fields

### Custom Headers:
The `table` prefix auto-generates headers from field names. There is no built-in way to specify completely custom header text while maintaining table formatting. Headers are generated by converting field names to uppercase with spaces (e.g., `HelmRegistryName` → `HELM REGISTRY NAME`).

---

## Profile Listing Limitations

Profiles are different from most other resources - they are **not** fetched via a dedicated list endpoint. Instead, they are retrieved as a nested array within an application object via `GET /applications/{name}/{version}`.

### Unsupported Flags (No Server-Side API):
- ❌ `--filter` (server-side filtering) - not available
- ❌ `--order-by` (server-side ordering) - not available
- ❌ `--page-size` / `--offset` (pagination) - not needed (all profiles returned in one call)

### Supported Flags (Client-Side):
- ✅ `--output-filter` - client-side filtering after fetching all profiles
- ✅ `--output-template` / `--output-template-file` - custom formatting
- ✅ `--output-type` (table/json/yaml)
- ✅ `--verbose` - detailed profile information

### Field Names for `--output-filter`:
Use **PascalCase** Go struct field names:

| Field Name | Type | Example Filter |
|---|---|---|
| `Name` | `string` | `--output-filter 'Name=default'` |
| `DisplayName` | `*string` | `--output-filter 'DisplayName=Default'` |
| `Description` | `*string` | `--output-filter 'Description~software'` |
| `ChartValues` | `*string` | — |
| `CreateTime` | `time.Time` | — |
| `UpdateTime` | `time.Time` | — |

**Examples:**
```bash
# Filter to specific profile
./build/_output/orch-cli list profiles kubevirt 1.4.4 --project itep \
  --output-filter 'Name=default'

# Exclude default profile
./build/_output/orch-cli list profiles kubevirt 1.4.4 --project itep \
  --output-filter 'Name!=default'

# Regex match on name
./build/_output/orch-cli list profiles kubevirt 1.4.4 --project itep \
  --output-filter 'Name~.*emulation.*'

# Filter by display name
./build/_output/orch-cli list profiles kubevirt 1.4.4 --project itep \
  --output-filter 'DisplayName=Software Emulation'
```

### Why No Server-Side Filtering?
Since all profiles for an application are returned in a single API call (typically a small number), and there's no pagination needed, the lack of `--filter` and `--order-by` is less problematic than it would be for resources with potentially thousands of items. Use `--output-filter` for any filtering needs.

---

## Order-By Flag Behavior (All Resources)

As of 2026-04-17, all list commands support **case-insensitive field name matching** for the `--order-by` flag.

### Supported Field Name Formats:
The CLI accepts multiple naming conventions and converts them to the internal Go struct field names:

- **PascalCase** (Go struct fields): `Name`, `DisplayName`, `CreateTime`
- **camelCase** (JSON/API fields): `name`, `displayName`, `createTime`
- **snake_case** (alternate style): `display_name`, `create_time`
- **lowercase**: `name`, `displayname`

**Examples (all equivalent):**
```bash
# All of these work the same:
--order-by Name
--order-by name
--order-by NAME

# All of these work the same:
--order-by DisplayName
--order-by displayName
--order-by display_name
--order-by DISPLAY_NAME
```

### Direction Prefixes:
- `+field` or `field` — ascending order (default)
- `-field` — descending order

**Examples:**
```bash
# Descending by name
--order-by -name
--order-by -Name

# Ascending by createTime
--order-by createTime
--order-by +create_time
```

### Error Messages with Field Hints:
When an invalid field name is used, the CLI provides helpful suggestions:

```bash
$ orch-cli list applications --order-by invalidField
invalid --order-by field "invalidField"; available fields: chartName, chartVersion, 
createTime, defaultProfileName, description, displayName, helmRegistryName, 
ignoredResources, imageRegistryName, kind, name, profiles, updateTime, version
```

### Table vs JSON/YAML Output:
- **Table output** (`--output-type table`): Uses client-side sorting after fetching all paginated results
- **JSON/YAML output** (`--output-type json|yaml`): Uses server-side API sorting when available

---

## Filter Flag Behavior (Resource-Specific)

### Server-Side API Filtering (`--filter`)

Support for the `--filter` flag varies by resource based on API capabilities:

| Resource | `--filter` Support | Notes |
|---|---|---|
| **Applications** | ✅ Full support | API filtering with camelCase field names |
| **Deployments** | ✅ Full support | API filtering with camelCase field names |
| **Deployment Packages** | ✅ Full support | API filtering with camelCase field names |
| **Deployment Profiles** | ✅ Full support | API filtering with camelCase field names |
| **Cluster Templates** | ✅ Full support | API filtering with camelCase field names |
| **Artifacts** | ❌ **Not supported** | API returns 400 Bad Request on filter parameter |
| **Profiles** | ❌ **Not supported** | No dedicated list endpoint (nested in application) |
| **Registries** | ✅ Full support | API filtering with camelCase field names |

### When `--filter` Works (API-Level):
- Uses **camelCase** JSON field names (e.g., `name`, `displayName`)
- Sent to the API server; reduces data transfer
- Follows [Google AIP-160](https://google.aip.dev/160) syntax
- Applies to all output types (table, json, yaml)

**Example:**
```bash
# API filtering on deployments
./build/_output/orch-cli list deployments --project itep \
  --filter 'name~deployment-j'
```

### When `--filter` Doesn't Work:
For **artifacts** and **profiles**, using `--filter` will result in errors:

```bash
# Artifacts - results in "error listing artifacts:[400 Bad Request]"
./build/_output/orch-cli list artifacts --project itep \
  --filter 'Name=prod-config'

# Profiles - no --filter flag available (use --output-filter instead)
```

### Client-Side Filtering (`--output-filter`)

**Available for ALL resources**, regardless of API support:
- Uses **Go struct field names**, but **case-insensitive** (accepts PascalCase, camelCase, snake_case, lowercase)
- Applied locally after fetching all data
- Only affects table output (ignored for JSON/YAML)
- Works on any model field, including those not supported by API filters

**Examples:**
```bash
# Case-insensitive field names (all work!)
./build/_output/orch-cli list artifacts --project itep \
  --output-filter 'name=prod-config'
  
./build/_output/orch-cli list artifacts --project itep \
  --output-filter 'Name=prod-config'
  
./build/_output/orch-cli list artifacts --project itep \
  --output-filter 'display_name~config'

# Profiles (no API endpoint, use output-filter)
./build/_output/orch-cli list profiles kubevirt 1.4.4 --project itep \
  --output-filter 'name=default'

# Deployments (can combine both filters)
./build/_output/orch-cli list deployments --project itep \
  --filter 'name~deployment-' \
  --output-filter 'status.state!=NO_TARGET_CLUSTERS'
```

### Recommendation:
- For resources with API filtering support: use `--filter` to reduce data transfer
- For artifacts and profiles: use `--output-filter` for client-side filtering
- When you need to filter on fields not supported by the API: use `--output-filter`
