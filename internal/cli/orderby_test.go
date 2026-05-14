// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOrderByModel is a minimal struct with JSON tags used by order-by unit tests.
type testOrderByModel struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
}

// resetOrderByCache clears the package-level cache so tests start with a clean slate.
func resetOrderByCache() {
	orderBySupportCache.mu.Lock()
	orderBySupportCache.fields = map[string][]string{}
	orderBySupportCache.mu.Unlock()
}

// acceptAllProbe accepts any expression unconditionally.
func acceptAllProbe(_ string) (bool, error) { return true, nil }

// rejectAllProbe rejects any expression unconditionally.
func rejectAllProbe(_ string) (bool, error) { return false, nil }

// acceptNameAndDisplayProbe accepts only the "name" and "displayName" fields (and their
// directional variants), simulating a server that does not support "version" ordering.
func acceptNameAndDisplayProbe(orderBy string) (bool, error) {
	switch orderBy {
	case "name", "name asc", "name desc",
		"displayName", "displayName asc", "displayName desc":
		return true, nil
	}
	return false, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// camelToSnake
// ──────────────────────────────────────────────────────────────────────────────

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"name", "name"},
		{"Name", "name"},
		{"version", "version"},
		{"displayName", "display_name"},
		{"DisplayName", "display_name"},
		{"someFieldName", "some_field_name"},
		{"HTMLParser", "h_t_m_l_parser"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, camelToSnake(tt.input))
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// buildOrderByAliases
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildOrderByAliases_CanonicalList(t *testing.T) {
	_, canonical := buildOrderByAliases(testOrderByModel{})
	// Should be sorted JSON names.
	assert.Equal(t, []string{"displayName", "name", "version"}, canonical)
}

func TestBuildOrderByAliases_NameField(t *testing.T) {
	aliases, _ := buildOrderByAliases(testOrderByModel{})
	assert.Equal(t, "name", aliases["name"])
	assert.Equal(t, "name", aliases["Name"]) // struct field name
	assert.Equal(t, "name", aliases["name"]) // json name (lowercase already)
}

func TestBuildOrderByAliases_DisplayNameField(t *testing.T) {
	aliases, _ := buildOrderByAliases(testOrderByModel{})
	assert.Equal(t, "displayName", aliases["displayName"])
	assert.Equal(t, "displayName", aliases["display_name"]) // snake_case
	assert.Equal(t, "displayName", aliases["DisplayName"])  // struct field name
	assert.Equal(t, "displayName", aliases["displayname"])  // lowercase
}

func TestBuildOrderByAliases_IgnoresUntaggedFields(t *testing.T) {
	type modelWithUntagged struct {
		Name    string `json:"name"`
		private string //nolint:unused
		NoTag   string
	}
	aliases, canonical := buildOrderByAliases(modelWithUntagged{})
	assert.Equal(t, []string{"name"}, canonical)
	_, hasNoTag := aliases["NoTag"]
	assert.False(t, hasNoTag, "untagged field should not appear in aliases")
}

// ──────────────────────────────────────────────────────────────────────────────
// buildClientSortAliases
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildClientSortAliases_JsonHints(t *testing.T) {
	_, hints := buildClientSortAliases(testOrderByModel{})
	assert.Equal(t, []string{"displayName", "name", "version"}, hints)
}

func TestBuildClientSortAliases_MapsToStructFieldName(t *testing.T) {
	aliases, _ := buildClientSortAliases(testOrderByModel{})

	// name field
	assert.Equal(t, "Name", aliases["name"])
	assert.Equal(t, "Name", aliases["Name"])

	// displayName field — all variants must resolve to the Go struct field
	assert.Equal(t, "DisplayName", aliases["displayName"])
	assert.Equal(t, "DisplayName", aliases["display_name"])
	assert.Equal(t, "DisplayName", aliases["DisplayName"])
	assert.Equal(t, "DisplayName", aliases["displayname"])

	// version field
	assert.Equal(t, "Version", aliases["version"])
	assert.Equal(t, "Version", aliases["Version"])
}

// ──────────────────────────────────────────────────────────────────────────────
// normalizeOrderByForClientSorting
// ──────────────────────────────────────────────────────────────────────────────

func TestNormalizeOrderByForClientSorting(t *testing.T) {
	sample := testOrderByModel{}

	tests := []struct {
		name    string
		raw     string
		want    string // empty == expect nil result
		wantNil bool
		wantErr string
	}{
		{
			name:    "empty string returns nil",
			raw:     "",
			wantNil: true,
		},
		{
			name:    "whitespace-only returns nil",
			raw:     "   ",
			wantNil: true,
		},
		{
			name: "plain field name",
			raw:  "name",
			want: "Name",
		},
		{
			name: "plus prefix preserved",
			raw:  "+name",
			want: "+Name",
		},
		{
			name: "minus prefix preserved",
			raw:  "-version",
			want: "-Version",
		},
		{
			name: "greater-than prefix preserved",
			raw:  ">name",
			want: ">Name",
		},
		{
			name: "less-than prefix preserved",
			raw:  "<name",
			want: "<Name",
		},
		{
			name: "camelCase alias resolves",
			raw:  "displayName",
			want: "DisplayName",
		},
		{
			name: "snake_case alias resolves",
			raw:  "display_name",
			want: "DisplayName",
		},
		{
			name: "PascalCase alias resolves",
			raw:  "DisplayName",
			want: "DisplayName",
		},
		{
			name: "multiple fields sorted",
			raw:  "+name,-version",
			want: "+Name,-Version",
		},
		{
			name: "mixed aliases in multi-field",
			raw:  "display_name,+Name",
			want: "DisplayName,+Name",
		},
		{
			name:    "unknown field returns error with hints",
			raw:     "unknown",
			wantErr: `invalid --order-by field "unknown"; available fields:`,
		},
		{
			name:    "bare plus prefix returns error",
			raw:     "+",
			wantErr: `invalid --order-by term "+"`,
		},
		{
			name:    "bare minus prefix returns error",
			raw:     "-",
			wantErr: `invalid --order-by term "-"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeOrderByForClientSorting(tt.raw, sample)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.want, *got)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// normalizeOrderByWithAPIProbe
// ──────────────────────────────────────────────────────────────────────────────

func TestNormalizeOrderByWithAPIProbe_EmptyReturnsNil(t *testing.T) {
	resetOrderByCache()
	got, err := normalizeOrderByWithAPIProbe("", "key-empty", testOrderByModel{}, acceptAllProbe)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestNormalizeOrderByWithAPIProbe_WhitespaceReturnsNil(t *testing.T) {
	resetOrderByCache()
	got, err := normalizeOrderByWithAPIProbe("   ", "key-ws", testOrderByModel{}, acceptAllProbe)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestNormalizeOrderByWithAPIProbe_PlainField(t *testing.T) {
	resetOrderByCache()
	got, err := normalizeOrderByWithAPIProbe("name", "key-plain", testOrderByModel{}, acceptAllProbe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "name", *got)
}

func TestNormalizeOrderByWithAPIProbe_PlusPrefixConverted(t *testing.T) {
	resetOrderByCache()
	got, err := normalizeOrderByWithAPIProbe("+name", "key-plus", testOrderByModel{}, acceptAllProbe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "name asc", *got)
}

func TestNormalizeOrderByWithAPIProbe_MinusPrefixConverted(t *testing.T) {
	resetOrderByCache()
	got, err := normalizeOrderByWithAPIProbe("-version", "key-minus", testOrderByModel{}, acceptAllProbe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "version desc", *got)
}

func TestNormalizeOrderByWithAPIProbe_MultipleFields(t *testing.T) {
	resetOrderByCache()
	probe := func(_ string) (bool, error) {
		return true, nil
	}
	got, err := normalizeOrderByWithAPIProbe("+name,-version", "key-multi", testOrderByModel{}, probe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "name asc,version desc", *got)
}

func TestNormalizeOrderByWithAPIProbe_CamelCaseAlias(t *testing.T) {
	resetOrderByCache()
	got, err := normalizeOrderByWithAPIProbe("displayName", "key-camel", testOrderByModel{}, acceptAllProbe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "displayName", *got)
}

func TestNormalizeOrderByWithAPIProbe_SnakeCaseAlias(t *testing.T) {
	resetOrderByCache()
	got, err := normalizeOrderByWithAPIProbe("display_name", "key-snake", testOrderByModel{}, acceptAllProbe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "displayName", *got)
}

func TestNormalizeOrderByWithAPIProbe_PlusCamelCase(t *testing.T) {
	resetOrderByCache()
	got, err := normalizeOrderByWithAPIProbe("+displayName", "key-plus-camel", testOrderByModel{}, acceptAllProbe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "displayName asc", *got)
}

func TestNormalizeOrderByWithAPIProbe_RejectsGreaterThanPrefix(t *testing.T) {
	resetOrderByCache()
	_, err := normalizeOrderByWithAPIProbe(">name", "key-gt", testOrderByModel{}, acceptAllProbe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use plain field names or +/- prefixes")
}

func TestNormalizeOrderByWithAPIProbe_RejectsLessThanPrefix(t *testing.T) {
	resetOrderByCache()
	_, err := normalizeOrderByWithAPIProbe("<name", "key-lt", testOrderByModel{}, acceptAllProbe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use plain field names or +/- prefixes")
}

func TestNormalizeOrderByWithAPIProbe_RejectsKeywordForm(t *testing.T) {
	resetOrderByCache()
	_, err := normalizeOrderByWithAPIProbe("name desc", "key-kw", testOrderByModel{}, acceptAllProbe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use <field>, +<field>, or -<field>")
}

func TestNormalizeOrderByWithAPIProbe_RejectsKeywordAsc(t *testing.T) {
	resetOrderByCache()
	_, err := normalizeOrderByWithAPIProbe("name asc", "key-kw-asc", testOrderByModel{}, acceptAllProbe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use <field>, +<field>, or -<field>")
}

func TestNormalizeOrderByWithAPIProbe_UnknownField(t *testing.T) {
	resetOrderByCache()
	probeCalls := 0
	probe := func(orderBy string) (bool, error) {
		probeCalls++
		return acceptNameAndDisplayProbe(orderBy)
	}
	_, err := normalizeOrderByWithAPIProbe("bogus", "key-unknown", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Contains(t, err.Error(), "available fields:")
	// One probe call for the unknown field itself, plus 3 canonical fields for cache building.
	assert.Equal(t, 4, probeCalls)
}

func TestNormalizeOrderByWithAPIProbe_UnsupportedFieldHintsOnlySupportedFields(t *testing.T) {
	resetOrderByCache()
	// "version" is a valid model field, but the probe rejects it.
	probe := acceptNameAndDisplayProbe
	_, err := normalizeOrderByWithAPIProbe("version", "key-unsupported", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"version"`)
	assert.Contains(t, err.Error(), "available fields:")
	// Hint list should include the fields the API accepts.
	assert.Contains(t, err.Error(), "name")
	assert.Contains(t, err.Error(), "displayName")
}

func TestNormalizeOrderByWithAPIProbe_ProbeFirstAcceptedDoesNotBuildFullCache(t *testing.T) {
	resetOrderByCache()
	probeCalls := 0
	probe := func(_ string) (bool, error) {
		probeCalls++
		return true, nil
	}
	got, err := normalizeOrderByWithAPIProbe("name", "key-probe-first", testOrderByModel{}, probe)
	require.NoError(t, err)
	require.NotNil(t, got)
	// Only 1 probe call for the expression itself; no per-field discovery.
	assert.Equal(t, 1, probeCalls)
}

// ──────────────────────────────────────────────────────────────────────────────
// getSupportedOrderByFields
// ──────────────────────────────────────────────────────────────────────────────

func TestGetSupportedOrderByFields_ReturnsAcceptedFields(t *testing.T) {
	resetOrderByCache()
	supported, set, err := getSupportedOrderByFields("key-supported", testOrderByModel{}, acceptNameAndDisplayProbe)
	require.NoError(t, err)
	assert.Equal(t, []string{"displayName", "name"}, supported)
	assert.Contains(t, set, "name")
	assert.Contains(t, set, "displayName")
	assert.NotContains(t, set, "version")
}

func TestGetSupportedOrderByFields_AllAccepted(t *testing.T) {
	resetOrderByCache()
	supported, set, err := getSupportedOrderByFields("key-all", testOrderByModel{}, acceptAllProbe)
	require.NoError(t, err)
	assert.Equal(t, []string{"displayName", "name", "version"}, supported)
	assert.Len(t, set, 3)
}

func TestGetSupportedOrderByFields_NoneAccepted(t *testing.T) {
	resetOrderByCache()
	supported, set, err := getSupportedOrderByFields("key-none", testOrderByModel{}, rejectAllProbe)
	require.NoError(t, err)
	assert.Empty(t, supported)
	assert.Empty(t, set)
}

func TestGetSupportedOrderByFields_CacheHitSkipsProbe(t *testing.T) {
	resetOrderByCache()
	callCount := 0
	probe := func(_ string) (bool, error) {
		callCount++
		return true, nil
	}

	_, _, err := getSupportedOrderByFields("key-cache-hit", testOrderByModel{}, probe)
	require.NoError(t, err)
	firstCount := callCount

	// Second call with the same resource key must not invoke the probe again.
	_, _, err = getSupportedOrderByFields("key-cache-hit", testOrderByModel{}, probe)
	require.NoError(t, err)
	assert.Equal(t, firstCount, callCount, "probe should not be called on a cache hit")
}

func TestGetSupportedOrderByFields_DifferentKeysMissCache(t *testing.T) {
	resetOrderByCache()
	callCount := 0
	probe := func(_ string) (bool, error) {
		callCount++
		return true, nil
	}

	_, _, err := getSupportedOrderByFields("key-miss-A", testOrderByModel{}, probe)
	require.NoError(t, err)
	countAfterFirst := callCount

	_, _, err = getSupportedOrderByFields("key-miss-B", testOrderByModel{}, probe)
	require.NoError(t, err)
	// Both keys are distinct → probe should be called again for the second key.
	assert.Greater(t, callCount, countAfterFirst)
}

// ─────────────────────────────────────────────────────────────────────────────
// api400Error
// ─────────────────────────────────────────────────────────────────────────────

func TestApi400Error_Error(t *testing.T) {
	e := &api400Error{msg: "HTTP 400: invalid field"}
	assert.Equal(t, "HTTP 400: invalid field", e.Error())
}

// ─────────────────────────────────────────────────────────────────────────────
// buildOrderByAliases / buildClientSortAliases — json:"-," tag
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildOrderByAliases_IgnoresDashCommaTag(t *testing.T) {
	type modelWithDashTag struct {
		Name   string `json:"name"`
		Hidden string `json:"-,"`
	}
	aliases, canonical := buildOrderByAliases(modelWithDashTag{})
	assert.Equal(t, []string{"name"}, canonical)
	_, hasHidden := aliases["-"]
	assert.False(t, hasHidden, "field with json:\"-,\" should not appear in aliases")
}

func TestBuildClientSortAliases_IgnoresDashCommaTag(t *testing.T) {
	type modelWithDashTag struct {
		Name   string `json:"name"`
		Hidden string `json:"-,"`
	}
	aliases, hints := buildClientSortAliases(modelWithDashTag{})
	assert.Equal(t, []string{"name"}, hints)
	_, hasHidden := aliases["-"]
	assert.False(t, hasHidden, "field with json:\"-,\" should not appear in aliases")
}

// ─────────────────────────────────────────────────────────────────────────────
// getSupportedOrderByFields — probe returns api400Error or non-api400 error
// ─────────────────────────────────────────────────────────────────────────────

func TestGetSupportedOrderByFields_ProbeReturnsApi400Error(t *testing.T) {
	resetOrderByCache()
	// Probe returns api400Error for every field → all fields skipped → empty supported list.
	probe := func(_ string) (bool, error) {
		return false, &api400Error{msg: "API 400: bad field"}
	}
	supported, set, err := getSupportedOrderByFields("key-400-err", testOrderByModel{}, probe)
	require.NoError(t, err)
	assert.Empty(t, supported)
	assert.Empty(t, set)
}

func TestGetSupportedOrderByFields_ProbeReturnsNonApi400Error(t *testing.T) {
	resetOrderByCache()
	probe := func(_ string) (bool, error) {
		return false, fmt.Errorf("network error")
	}
	supported, set, err := getSupportedOrderByFields("key-net-err", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
	assert.Nil(t, supported)
	assert.Nil(t, set)
}

// ─────────────────────────────────────────────────────────────────────────────
// normalizeOrderByWithAPIProbe — final probe error paths
// ─────────────────────────────────────────────────────────────────────────────

func TestNormalizeOrderByWithAPIProbe_FinalProbeApi400Error(t *testing.T) {
	resetOrderByCache()
	// All individual fields probe as (true, nil) so getSupportedOrderByFields works,
	// but the combined expression probe returns api400Error.
	callNum := 0
	probe := func(_ string) (bool, error) {
		callNum++
		if callNum == 1 {
			// First call is the full expression → api400Error
			return false, &api400Error{msg: "API 400: combined sort rejected"}
		}
		// Subsequent calls are per-field probes in getSupportedOrderByFields
		return true, nil
	}
	_, err := normalizeOrderByWithAPIProbe("name", "key-final-400", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API 400: combined sort rejected")
	assert.Contains(t, err.Error(), "available fields:")
}

func TestNormalizeOrderByWithAPIProbe_FinalProbeNonApi400Error(t *testing.T) {
	resetOrderByCache()
	probe := func(_ string) (bool, error) {
		return false, fmt.Errorf("server error 500")
	}
	_, err := normalizeOrderByWithAPIProbe("name", "key-final-500", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server error 500")
}

func TestNormalizeOrderByWithAPIProbe_RejectedGetSupportedReturnsError(t *testing.T) {
	resetOrderByCache()
	// First call (full expression): rejected with (false, nil).
	// Subsequent calls (per-field in getSupportedOrderByFields): return error.
	callNum := 0
	probe := func(_ string) (bool, error) {
		callNum++
		if callNum == 1 {
			return false, nil // reject expression
		}
		return false, fmt.Errorf("discovery error")
	}
	_, err := normalizeOrderByWithAPIProbe("name", "key-rejected-disc-err", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "discovery error")
}

func TestNormalizeOrderByWithAPIProbe_AllApiFieldsInSupportedSet(t *testing.T) {
	resetOrderByCache()
	// Expression probe is called first; subsequent per-field probes accept "name" only.
	callNum := 0
	probe := func(expr string) (bool, error) {
		callNum++
		if callNum == 1 {
			return false, nil // reject the combined expression
		}
		return expr == "name", nil // accept only "name" for getSupportedOrderByFields
	}
	_, err := normalizeOrderByWithAPIProbe("name", "key-all-in-set", testOrderByModel{}, probe)
	require.Error(t, err)
	// All apiFields ("name") are in supportedSet, so falls through to expression-level error.
	assert.Contains(t, err.Error(), "invalid --order-by expression")
}

// ─────────────────────────────────────────────────────────────────────────────
// getSupportedFilterFields
// ─────────────────────────────────────────────────────────────────────────────

func acceptAllFilterProbe(_ string) (bool, error) { return true, nil }
func rejectAllFilterProbe(_ string) (bool, error) { return false, nil }

func TestGetSupportedFilterFields_AllAccepted(t *testing.T) {
	resetOrderByCache()
	supported, set, err := getSupportedFilterFields("fkey-all", testOrderByModel{}, acceptAllFilterProbe)
	require.NoError(t, err)
	assert.Equal(t, []string{"displayName", "name", "version"}, supported)
	assert.Len(t, set, 3)
}

func TestGetSupportedFilterFields_NoneAccepted(t *testing.T) {
	resetOrderByCache()
	supported, set, err := getSupportedFilterFields("fkey-none", testOrderByModel{}, rejectAllFilterProbe)
	require.NoError(t, err)
	assert.Empty(t, supported)
	assert.Empty(t, set)
}

func TestGetSupportedFilterFields_ProbeError(t *testing.T) {
	resetOrderByCache()
	probe := func(_ string) (bool, error) {
		return false, fmt.Errorf("filter probe error")
	}
	supported, set, err := getSupportedFilterFields("fkey-err", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "filter probe error")
	assert.Nil(t, supported)
	assert.Nil(t, set)
}

func TestGetSupportedFilterFields_CacheHit(t *testing.T) {
	resetOrderByCache()
	callCount := 0
	probe := func(_ string) (bool, error) {
		callCount++
		return true, nil
	}
	_, _, err := getSupportedFilterFields("fkey-cache", testOrderByModel{}, probe)
	require.NoError(t, err)
	firstCount := callCount

	_, _, err = getSupportedFilterFields("fkey-cache", testOrderByModel{}, probe)
	require.NoError(t, err)
	assert.Equal(t, firstCount, callCount, "probe should not be called on cache hit")
}

// ─────────────────────────────────────────────────────────────────────────────
// normalizeFilterWithAPIProbe
// ─────────────────────────────────────────────────────────────────────────────

func TestNormalizeFilterWithAPIProbe_EmptyReturnsNil(t *testing.T) {
	resetOrderByCache()
	got, err := normalizeFilterWithAPIProbe("", "fkey-empty", testOrderByModel{}, acceptAllFilterProbe)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestNormalizeFilterWithAPIProbe_WhitespaceReturnsNil(t *testing.T) {
	resetOrderByCache()
	got, err := normalizeFilterWithAPIProbe("   ", "fkey-ws", testOrderByModel{}, acceptAllFilterProbe)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestNormalizeFilterWithAPIProbe_InvalidSyntax(t *testing.T) {
	resetOrderByCache()
	// "invalidfilter" has no operator → pfilter.Parse fails
	_, err := normalizeFilterWithAPIProbe("invalidfilter", "fkey-syntax", testOrderByModel{}, acceptAllFilterProbe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse filter expression")
}

func TestNormalizeFilterWithAPIProbe_ProbeAccepts(t *testing.T) {
	resetOrderByCache()
	got, err := normalizeFilterWithAPIProbe("name~test", "fkey-accept", testOrderByModel{}, acceptAllFilterProbe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "name~test", *got)
}

func TestNormalizeFilterWithAPIProbe_AliasMapping(t *testing.T) {
	resetOrderByCache()
	// "display_name" is a snake_case alias → should be normalized to "displayName"
	got, err := normalizeFilterWithAPIProbe("display_name~test", "fkey-alias", testOrderByModel{}, acceptAllFilterProbe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "displayName~test", *got)
}

func TestNormalizeFilterWithAPIProbe_ProbeNonApi400Error(t *testing.T) {
	resetOrderByCache()
	// Non-api400 error from probe → normalized filter is returned (passthrough)
	probe := func(_ string) (bool, error) {
		return false, fmt.Errorf("server unavailable")
	}
	got, err := normalizeFilterWithAPIProbe("name~test", "fkey-non400", testOrderByModel{}, probe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "name~test", *got)
}

func TestNormalizeFilterWithAPIProbe_ProbeApi400ErrorWithHints(t *testing.T) {
	resetOrderByCache()
	// Main expression probe → api400Error; per-field probes (in getSupportedFilterFields) accept all.
	probe := func(expr string) (bool, error) {
		if strings.HasSuffix(expr, "~.*") {
			return true, nil
		}
		return false, &api400Error{msg: "API 400: unsupported filter expression"}
	}
	_, err := normalizeFilterWithAPIProbe("name~test", "fkey-400-hints", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API 400: unsupported filter expression")
	assert.Contains(t, err.Error(), "available fields:")
}

func TestNormalizeFilterWithAPIProbe_ProbeRejectsEmptySupported(t *testing.T) {
	resetOrderByCache()
	// Probe rejects everything → getSupportedFilterFields returns empty → canonical fallback
	_, err := normalizeFilterWithAPIProbe("name~test", "fkey-empty-supp", testOrderByModel{}, rejectAllFilterProbe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --filter expression")
	assert.Contains(t, err.Error(), "available fields:")
}

func TestNormalizeFilterWithAPIProbe_ProbeRejectsFieldNotInSupportedSet(t *testing.T) {
	resetOrderByCache()
	// Only "displayName~.*" is accepted in per-field discovery, but user queried "name"
	probe := func(expr string) (bool, error) {
		return expr == "displayName~.*", nil
	}
	_, err := normalizeFilterWithAPIProbe("name~test", "fkey-field-not-in-set", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --filter field")
	assert.Contains(t, err.Error(), `"name"`)
}

func TestNormalizeFilterWithAPIProbe_ProbeRejectsAllFieldsInSupportedSet(t *testing.T) {
	resetOrderByCache()
	// Per-field discovery accepts all fields, but expression probe rejects → expression-level error
	probe := func(expr string) (bool, error) {
		if strings.HasSuffix(expr, "~.*") {
			return true, nil
		}
		return false, nil
	}
	_, err := normalizeFilterWithAPIProbe("name~test", "fkey-expr-err", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --filter expression")
}

// ─────────────────────────────────────────────────────────────────────────────
// Pointer inputs to buildOrderByAliases / buildClientSortAliases
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildOrderByAliases_PointerInput(t *testing.T) {
	sample := &testOrderByModel{}
	_, canonical := buildOrderByAliases(sample)
	assert.Equal(t, []string{"displayName", "name", "version"}, canonical)
}

func TestBuildClientSortAliases_PointerInput(t *testing.T) {
	sample := &testOrderByModel{}
	_, hints := buildClientSortAliases(sample)
	assert.Equal(t, []string{"displayName", "name", "version"}, hints)
}

// ─────────────────────────────────────────────────────────────────────────────
// normalizeOrderByWithAPIProbe edge cases
// ─────────────────────────────────────────────────────────────────────────────

func TestNormalizeOrderByWithAPIProbe_EmptyTermInList(t *testing.T) {
	resetOrderByCache()
	// "name,,version" — middle term is empty → should be skipped gracefully
	got, err := normalizeOrderByWithAPIProbe("name,,version", "key-empty-term", testOrderByModel{}, acceptAllProbe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "name,version", *got)
}

func TestNormalizeOrderByWithAPIProbe_PlusEmptyField(t *testing.T) {
	resetOrderByCache()
	_, err := normalizeOrderByWithAPIProbe("+", "key-plus-empty", testOrderByModel{}, acceptAllProbe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"+"`)
}

func TestNormalizeOrderByWithAPIProbe_AllEmptyTermsReturnsNil(t *testing.T) {
	resetOrderByCache()
	// All-whitespace terms → normalized is empty → nil
	got, err := normalizeOrderByWithAPIProbe(" , , ", "key-all-empty-terms", testOrderByModel{}, acceptAllProbe)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestNormalizeOrderByWithAPIProbe_UnknownFieldProbeApi400(t *testing.T) {
	resetOrderByCache()
	// Probe returns api400Error for unknown field itself (first probe call for field)
	probe := func(expr string) (bool, error) {
		return false, &api400Error{msg: "API 400: unknown field " + expr}
	}
	_, err := normalizeOrderByWithAPIProbe("bogus", "key-unknown-400", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API 400: unknown field")
}

func TestNormalizeOrderByWithAPIProbe_FinalProbeApi400HintFromSupported(t *testing.T) {
	resetOrderByCache()
	// Probe accepts individual fields but rejects the combined expression with api400Error.
	// getSupportedOrderByFields returns non-empty so hints come from supported list.
	callNum := 0
	probe := func(expr string) (bool, error) {
		callNum++
		if callNum == 1 {
			// Full expression probe → api400Error
			return false, &api400Error{msg: "API 400: cannot combine fields"}
		}
		// Per-field probes: accept only "name"
		return expr == "name", nil
	}
	_, err := normalizeOrderByWithAPIProbe("name", "key-final-400-supp", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API 400: cannot combine fields")
	assert.Contains(t, err.Error(), "available fields:")
	assert.Contains(t, err.Error(), "name")
}

// ─────────────────────────────────────────────────────────────────────────────
// normalizeFilterWithAPIProbe — filter term regex mismatch (line 395)
// ─────────────────────────────────────────────────────────────────────────────

func TestNormalizeFilterWithAPIProbe_MultipleFilterTerms(t *testing.T) {
	resetOrderByCache()
	// Two filter terms joined by comma
	got, err := normalizeFilterWithAPIProbe("name~test,version~1.0", "fkey-multi-filter-term", testOrderByModel{}, acceptAllFilterProbe)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "name~test,version~1.0", *got)
}

func TestNormalizeFilterWithAPIProbe_AllEmptyFilterTermsReturnsNil(t *testing.T) {
	resetOrderByCache()
	// All-whitespace terms after comma-split → normalizedTerms is empty → nil
	// But pfilter.Parse will reject "  ,  " so we can't get there easily.
	// Instead test with a single space-only comma list — this will fail pfilter.Parse first.
	// So this path can only be reached if the first comma-term is valid but later ones aren't.
	// This is effectively unreachable from outside, skip with a comment test.
	// Already tested via empty string path. No new test needed here.
	t.Skip("empty normalizedTerms is only reachable via internal filter term building, not from valid public input")
}

func TestNormalizeFilterWithAPIProbe_GetSupportedFilterFieldsError(t *testing.T) {
	resetOrderByCache()
	// Main expression probe: api400Error. getSupportedFilterFields probe: non-api400 error.
	// This covers the "herr != nil" branch on line 441.
	callNum := 0
	probe := func(_ string) (bool, error) {
		callNum++
		if callNum == 1 {
			// Full expression → api400Error
			return false, &api400Error{msg: "API 400: filter rejected"}
		}
		// getSupportedFilterFields per-field probes → non-api400 error
		return false, fmt.Errorf("discovery error for filter")
	}
	_, err := normalizeFilterWithAPIProbe("name~test", "fkey-disc-err", testOrderByModel{}, probe)
	require.Error(t, err)
	// Since getSupportedFilterFields failed, fall back to canonical, still show api400 error
	assert.Contains(t, err.Error(), "API 400: filter rejected")
	assert.Contains(t, err.Error(), "available fields:")
}

func TestNormalizeFilterWithAPIProbe_ProbeRejectedGetSupportedReturnsError(t *testing.T) {
	resetOrderByCache()
	// Expression probe: (false, nil). getSupportedFilterFields probes: return error.
	callNum := 0
	probe := func(_ string) (bool, error) {
		callNum++
		if callNum == 1 {
			return false, nil // reject expression
		}
		return false, fmt.Errorf("filter discovery error")
	}
	_, err := normalizeFilterWithAPIProbe("name~test", "fkey-rejected-disc-err", testOrderByModel{}, probe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "filter discovery error")
}

// ─────────────────────────────────────────────────────────────────────────────
// normalizeOrderByForClientSorting — empty term in comma list
// ─────────────────────────────────────────────────────────────────────────────

func TestNormalizeOrderByForClientSorting_EmptyTermInList(t *testing.T) {
	// "name,,version" — empty middle term skipped
	got, err := normalizeOrderByForClientSorting("name,,version", testOrderByModel{})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Name,Version", *got)
}

func TestNormalizeOrderByForClientSorting_AllEmptyTermsReturnsNil(t *testing.T) {
	got, err := normalizeOrderByForClientSorting(" , , ", testOrderByModel{})
	require.NoError(t, err)
	assert.Nil(t, got)
}
