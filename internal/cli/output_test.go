// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// outputTestItem is a minimal struct used as test data for GenerateOutput tests.
type outputTestItem struct {
	Name    string
	Version string
}

const testTableFormat format.Format = `table{{.Name}}\t{{.Version}}
`

// captureGenerateOutputFatal calls GenerateOutput and captures the fatal message
// produced by Fatalf instead of letting os.Exit terminate the process.
// It returns (output written to writer, fatal message, true) if Fatalf was called,
// or (output written, "", false) if it completed normally.
func captureGenerateOutputFatal(t *testing.T, writer *bytes.Buffer, result *CommandResult) (fatal string, didFatal bool) {
	t.Helper()
	old := exitFunc
	defer func() { exitFunc = old }()

	var fatalMsg string
	fatalCalled := false
	exitFunc = func(_ int) {
		fatalCalled = true
		panic("__fatalf__")
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				if s, ok := r.(string); ok && s == "__fatalf__" {
					return
				}
				panic(r)
			}
		}()
		GenerateOutput(writer, result)
	}()

	// Capture what was printed to stdout by Fatalf via bytes.Buffer
	// (Fatalf prints via fmt.Printf which goes to real stdout, not writer).
	// We just track whether it was called; the panic sentinel confirms the path.
	_ = fatalMsg
	return "", fatalCalled
}

// ─────────────────────────────────────────────────────────────────────────────
// toOutputType
// ─────────────────────────────────────────────────────────────────────────────

func TestToOutputType(t *testing.T) {
	assert.Equal(t, OUTPUT_TABLE, toOutputType("table"))
	assert.Equal(t, OUTPUT_TABLE, toOutputType(""))
	assert.Equal(t, OUTPUT_TABLE, toOutputType("unknown"))
	assert.Equal(t, OUTPUT_JSON, toOutputType("json"))
	assert.Equal(t, OUTPUT_YAML, toOutputType("yaml"))
}

// ─────────────────────────────────────────────────────────────────────────────
// Fatalf — exercises the function itself
// ─────────────────────────────────────────────────────────────────────────────

func TestFatalf_CallsExit(t *testing.T) {
	old := exitFunc
	defer func() { exitFunc = old }()

	called := false
	exitFunc = func(code int) {
		called = true
		assert.Equal(t, 1, code)
		panic("__exit__")
	}

	require.Panics(t, func() { Fatalf("test %s", "error") })
	assert.True(t, called)
}

// ─────────────────────────────────────────────────────────────────────────────
// GenerateOutput — nil / empty cases
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateOutput_NilResult(t *testing.T) {
	var buf bytes.Buffer
	// Should not panic
	GenerateOutput(&buf, nil)
	assert.Empty(t, buf.String())
}

func TestGenerateOutput_NilData(t *testing.T) {
	var buf bytes.Buffer
	result := &CommandResult{
		Format:   testTableFormat,
		OutputAs: OUTPUT_TABLE,
		Data:     nil,
	}
	GenerateOutput(&buf, result)
	assert.Empty(t, buf.String())
}

func TestGenerateOutput_NilWriter(_ *testing.T) {
	// writer=nil should default to stdout — exercise the nil-writer branch
	// without actually writing to stdout in a disruptive way; just ensure no panic.
	result := &CommandResult{
		Format:   testTableFormat,
		OutputAs: OUTPUT_TABLE,
		Data:     nil,
	}
	// Data is nil so nothing is written, but the nil-writer branch is traversed.
	GenerateOutput(nil, result)
}

// ─────────────────────────────────────────────────────────────────────────────
// GenerateOutput — TABLE output
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateOutput_TableOutput(t *testing.T) {
	var buf bytes.Buffer
	items := []outputTestItem{
		{Name: "alpha", Version: "1.0"},
		{Name: "beta", Version: "2.0"},
	}
	result := &CommandResult{
		Format:   `table{{.Name}}\t{{.Version}}` + "\n",
		OutputAs: OUTPUT_TABLE,
		Data:     items,
	}
	GenerateOutput(&buf, result)
	out := buf.String()
	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, "beta")
}

// ─────────────────────────────────────────────────────────────────────────────
// GenerateOutput — JSON output
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateOutput_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	items := []outputTestItem{
		{Name: "alpha", Version: "1.0"},
	}
	result := &CommandResult{
		Format:   testTableFormat,
		OutputAs: OUTPUT_JSON,
		Data:     items,
	}
	GenerateOutput(&buf, result)
	out := buf.String()
	assert.Contains(t, out, `"Name"`)
	assert.Contains(t, out, `"alpha"`)
	assert.Contains(t, out, `"Version"`)
}

// ─────────────────────────────────────────────────────────────────────────────
// GenerateOutput — YAML output
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateOutput_YAMLOutput(t *testing.T) {
	var buf bytes.Buffer
	items := []outputTestItem{
		{Name: "alpha", Version: "1.0"},
	}
	result := &CommandResult{
		Format:   testTableFormat,
		OutputAs: OUTPUT_YAML,
		Data:     items,
	}
	GenerateOutput(&buf, result)
	out := buf.String()
	assert.Contains(t, out, "name: alpha")
	assert.Contains(t, out, `version: "1.0"`)
}

// ─────────────────────────────────────────────────────────────────────────────
// GenerateOutput — Filter
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateOutput_FilterMatchesSomeItems(t *testing.T) {
	var buf bytes.Buffer
	items := []outputTestItem{
		{Name: "alpha", Version: "1.0"},
		{Name: "beta", Version: "2.0"},
	}
	result := &CommandResult{
		Format:   `table{{.Name}}\t{{.Version}}` + "\n",
		OutputAs: OUTPUT_TABLE,
		Filter:   "Name=alpha",
		Data:     items,
	}
	GenerateOutput(&buf, result)
	out := buf.String()
	assert.Contains(t, out, "alpha")
	assert.NotContains(t, out, "beta")
}

func TestGenerateOutput_FilterMatchesAll(t *testing.T) {
	var buf bytes.Buffer
	items := []outputTestItem{
		{Name: "alpha", Version: "1.0"},
		{Name: "beta", Version: "2.0"},
	}
	result := &CommandResult{
		Format:   `table{{.Name}}\t{{.Version}}` + "\n",
		OutputAs: OUTPUT_JSON,
		Filter:   "Version~.*",
		Data:     items,
	}
	GenerateOutput(&buf, result)
	out := buf.String()
	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, "beta")
}

func TestGenerateOutput_FilterParseError(t *testing.T) {
	var buf bytes.Buffer
	result := &CommandResult{
		Format:   testTableFormat,
		OutputAs: OUTPUT_TABLE,
		Filter:   "invalidfilter",
		Data:     []outputTestItem{{Name: "alpha"}},
	}
	_, didFatal := captureGenerateOutputFatal(t, &buf, result)
	assert.True(t, didFatal, "expected Fatalf to be called on bad filter syntax")
}

func TestGenerateOutput_FilterProcessError_UnknownField(t *testing.T) {
	var buf bytes.Buffer
	// Filter references a field that doesn't exist on the struct → Process error
	result := &CommandResult{
		Format:   `table{{.Name}}\t{{.Version}}` + "\n",
		OutputAs: OUTPUT_TABLE,
		Filter:   "NonExistent=alpha",
		Data:     []outputTestItem{{Name: "alpha"}},
	}
	_, didFatal := captureGenerateOutputFatal(t, &buf, result)
	assert.True(t, didFatal, "expected Fatalf to be called on unknown filter field")
}

func TestGenerateOutput_FilterProcessError_DottedOnNonStruct(t *testing.T) {
	var buf bytes.Buffer
	// Filter uses dotted notation on a non-struct field (Name.Sub on a string field)
	// → f.Process returns "did not resolve to a valid field" error
	// The table format has header fields so the specific branch (line 79-81) is taken.
	result := &CommandResult{
		Format:   format.Format(`table{{.Name}}\t{{.Version}}` + "\n"),
		OutputAs: OUTPUT_TABLE,
		Filter:   "Name.Sub=alpha",
		Data:     []outputTestItem{{Name: "alpha"}},
	}
	_, didFatal := captureGenerateOutputFatal(t, &buf, result)
	assert.True(t, didFatal, "expected Fatalf to be called on dotted-on-non-struct filter")
}

// ─────────────────────────────────────────────────────────────────────────────
// GenerateOutput — OrderBy
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateOutput_OrderByASC(t *testing.T) {
	var buf bytes.Buffer
	items := []outputTestItem{
		{Name: "beta", Version: "2.0"},
		{Name: "alpha", Version: "1.0"},
	}
	result := &CommandResult{
		Format:   `table{{.Name}}\t{{.Version}}` + "\n",
		OutputAs: OUTPUT_JSON,
		OrderBy:  "+Name",
		Data:     items,
	}
	GenerateOutput(&buf, result)
	out := buf.String()
	idxAlpha := strings.Index(out, "alpha")
	idxBeta := strings.Index(out, "beta")
	require.True(t, idxAlpha >= 0 && idxBeta >= 0)
	assert.Less(t, idxAlpha, idxBeta, "alpha should appear before beta in ASC sort")
}

func TestGenerateOutput_OrderByDSC(t *testing.T) {
	var buf bytes.Buffer
	items := []outputTestItem{
		{Name: "alpha", Version: "1.0"},
		{Name: "beta", Version: "2.0"},
	}
	result := &CommandResult{
		Format:   `table{{.Name}}\t{{.Version}}` + "\n",
		OutputAs: OUTPUT_JSON,
		OrderBy:  "-Name",
		Data:     items,
	}
	GenerateOutput(&buf, result)
	out := buf.String()
	idxAlpha := strings.Index(out, "alpha")
	idxBeta := strings.Index(out, "beta")
	require.True(t, idxAlpha >= 0 && idxBeta >= 0)
	assert.Greater(t, idxAlpha, idxBeta, "beta should appear before alpha in DSC sort")
}

func TestGenerateOutput_OrderByProcessError(t *testing.T) {
	var buf bytes.Buffer
	// Sort by a field that is a struct → Process returns error
	type itemWithStruct struct {
		Name   string
		Nested struct{ X string }
	}
	result := &CommandResult{
		Format:   `table{{.Name}}` + "\n",
		OutputAs: OUTPUT_TABLE,
		OrderBy:  "+Nested",
		Data:     []itemWithStruct{{Name: "alpha"}, {Name: "beta"}},
	}
	_, didFatal := captureGenerateOutputFatal(t, &buf, result)
	assert.True(t, didFatal, "expected Fatalf to be called on struct sort field")
}

// ─────────────────────────────────────────────────────────────────────────────
// GenerateOutput — Filter + OrderBy combined
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateOutput_FilterAndOrderBy(t *testing.T) {
	var buf bytes.Buffer
	items := []outputTestItem{
		{Name: "gamma", Version: "3.0"},
		{Name: "alpha", Version: "1.0"},
		{Name: "beta", Version: "2.0"},
	}
	result := &CommandResult{
		Format:   `table{{.Name}}\t{{.Version}}` + "\n",
		OutputAs: OUTPUT_JSON,
		Filter:   "Name~^(alpha|beta)$",
		OrderBy:  "+Name",
		Data:     items,
	}
	GenerateOutput(&buf, result)
	out := buf.String()
	assert.NotContains(t, out, "gamma")
	idxAlpha := strings.Index(out, "alpha")
	idxBeta := strings.Index(out, "beta")
	assert.Less(t, idxAlpha, idxBeta)
}
