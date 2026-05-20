// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/cli/pkg/rest/keycloak"
	"github.com/spf13/cobra"
)

const (
	DEFAULT_GROUP_FORMAT         = "table{{.Name}}\t{{.ID}}"
	DEFAULT_GROUP_VERBOSE_FORMAT = "table{{.Name}}\t{{.ID}}\t{{.Path}}"
	GROUP_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_GROUP_OUTPUT_TEMPLATE"
)

const listGroupsExamples = `# List all groups
orch-cli list groups

# List all groups in a specific realm
orch-cli list groups --realm master
`

func getGroupOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_GROUP_VERBOSE_FORMAT, nil
	}
	return resolveTableOutputTemplate(cmd, DEFAULT_GROUP_FORMAT, GROUP_OUTPUT_TEMPLATE_ENVVAR)
}

func printGroups(cmd *cobra.Command, writer io.Writer, groups []keycloak.GroupRepresentation, orderBy *string, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	outputFormat, err := getGroupOutputFormat(cmd, verbose)
	if err != nil {
		return err
	}

	sortSpec := ""
	if outputType == "table" && orderBy != nil {
		sortSpec = *orderBy
	}

	filterSpec := ""
	if outputType == "table" && outputFilter != nil && *outputFilter != "" {
		filterSpec = *outputFilter
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    filterSpec,
		OrderBy:   sortSpec,
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      groups,
	}

	GenerateOutput(writer, &result)
	return nil
}

func getListGroupsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "groups [flags]",
		Short:   "List all groups",
		Example: listGroupsExamples,
		Aliases: groupAliases,
		RunE:    runListGroupsCommand,
	}
	cmd.Flags().String("realm", "master", "Keycloak realm")
	cmd.Flags().String("order-by", "", "order results by field (table output only)")
	addStandardListOutputFlags(cmd)
	return cmd
}

func runListGroupsCommand(cmd *cobra.Command, _ []string) error {
	writer, _ := getOutputContext(cmd)

	ctx, kcClient, realm, err := KeycloakAdminFactory(cmd)
	if err != nil {
		return err
	}

	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return err
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	verbose, _ := cmd.Flags().GetBool("verbose")

	var validatedOrderBy *string
	if outputType == "table" {
		validatedOrderBy, err = normalizeOrderByForClientSorting(raw, keycloak.GroupRepresentation{})
	} else {
		// JSON/YAML: no API support, but allow any field for consistency
		if raw != "" {
			validatedOrderBy = &raw
		}
	}
	if err != nil {
		return err
	}

	groups, err := kcClient.ListGroups(ctx, realm)
	if err != nil {
		return fmt.Errorf("error listing groups: %w", err)
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printGroups(cmd, writer, groups, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}

	return writer.Flush()
}
