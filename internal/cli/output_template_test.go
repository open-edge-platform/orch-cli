// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func newTemplateTestCommand(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	addTableOutputTemplateFlags(cmd)
	return cmd
}

func TestResolveTableOutputTemplate_DefaultFallback(t *testing.T) {
	cmd := newTemplateTestCommand(t)
	defaultTemplate := "table{{.Name}}\t{{.Version}}"

	got, err := resolveTableOutputTemplate(cmd, defaultTemplate, "")
	assert.NoError(t, err)
	assert.Equal(t, defaultTemplate, got)
}

func TestResolveTableOutputTemplate_FromFlagUnescapesControlChars(t *testing.T) {
	cmd := newTemplateTestCommand(t)
	err := cmd.Flags().Set("output-template", `table{{.Name}}\t{{.Version}}`)
	assert.NoError(t, err)

	got, err := resolveTableOutputTemplate(cmd, "table{{.Name}}", "")
	assert.NoError(t, err)
	assert.Contains(t, got, "\t")
	assert.NotContains(t, got, `\t`)
}

func TestResolveTableOutputTemplate_FromFileUnescapesControlChars(t *testing.T) {
	cmd := newTemplateTestCommand(t)
	tmpFile := filepath.Join(t.TempDir(), "dp-table.tmpl")
	err := os.WriteFile(tmpFile, []byte(`table{{.Name}}\t{{.Version}}`), 0600)
	assert.NoError(t, err)

	err = cmd.Flags().Set("output-template-file", tmpFile)
	assert.NoError(t, err)

	got, err := resolveTableOutputTemplate(cmd, "table{{.Name}}", "")
	assert.NoError(t, err)
	assert.Contains(t, got, "\t")
	assert.NotContains(t, got, `\t`)
}

func TestResolveTableOutputTemplate_FromEnvUnescapesControlChars(t *testing.T) {
	cmd := newTemplateTestCommand(t)
	t.Setenv("TEST_TABLE_TEMPLATE", `table{{.Name}}\t{{.Kind}}`)

	got, err := resolveTableOutputTemplate(cmd, "table{{.Name}}", "TEST_TABLE_TEMPLATE")
	assert.NoError(t, err)
	assert.Contains(t, got, "\t")
	assert.NotContains(t, got, `\t`)
}

func TestResolveTableOutputTemplate_FlagOverridesEnv(t *testing.T) {
	cmd := newTemplateTestCommand(t)
	t.Setenv("TEST_TABLE_TEMPLATE", `table{{.Name}}\t{{.Kind}}`)
	err := cmd.Flags().Set("output-template", `table{{.Name}}\t{{.Version}}`)
	assert.NoError(t, err)

	got, err := resolveTableOutputTemplate(cmd, "table{{.Name}}", "TEST_TABLE_TEMPLATE")
	assert.NoError(t, err)
	assert.True(t, strings.Contains(got, "Version"))
	assert.False(t, strings.Contains(got, "Kind"))
}

func TestResolveTableOutputTemplate_DualFlagsError(t *testing.T) {
	cmd := newTemplateTestCommand(t)
	_ = cmd.Flags().Set("output-template", `table{{.Name}}`)
	_ = cmd.Flags().Set("output-template-file", "/tmp/invalid.tmpl")

	_, err := resolveTableOutputTemplate(cmd, "table{{.Name}}", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only one of")
}

func TestResolveTableOutputTemplate_MissingFileError(t *testing.T) {
	cmd := newTemplateTestCommand(t)
	_ = cmd.Flags().Set("output-template-file", "/nonexistent/path/template.tmpl")

	_, err := resolveTableOutputTemplate(cmd, "table{{.Name}}", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to read")
}

func TestGetDeploymentPackageOutputFormat_VerboseIgnoresTableOverrides(t *testing.T) {
	cmd := newTemplateTestCommand(t)
	t.Setenv(DEPLOYMENT_PACKAGE_OUTPUT_TEMPLATE_ENVVAR, `table{{.Name}}\t{{.Version}}`)
	_ = cmd.Flags().Set("output-template", `table{{.Name}}\t{{.Kind}}`)

	gotVerbose, err := getDeploymentPackageOutputFormat(cmd, true)
	assert.NoError(t, err)
	assert.Equal(t, DEFAULT_DEPLOYMENT_PACKAGE_INSPECT_FORMAT, gotVerbose)

	gotTable, err := getDeploymentPackageOutputFormat(cmd, false)
	assert.NoError(t, err)
	assert.Contains(t, gotTable, "\t")
	assert.NotContains(t, gotTable, `\t`)
	assert.Contains(t, gotTable, "Kind")
}
