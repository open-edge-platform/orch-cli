// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
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
	got, err := normalizeOrderByWithAPIProbe("+name,-version", "key-multi", testOrderByModel{}, acceptAllProbe)
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
	probe := func(orderBy string) (bool, error) {
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
	probe := func(orderBy string) (bool, error) {
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
	probe := func(orderBy string) (bool, error) {
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
