// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"context"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const listOSUpdateRunExamples = `# List all OS Update Policies
orch-cli list osupdaterun --project some-project
`

const getOSUpdateRunExamples = `# Get detailed information about specific OS Update Run using the run name
orch-cli get osupdaterun <resourceid> --project some-project`

const deleteOSUpdateRunExamples = `#Delete an OS Update Run  using it's name
orch-cli delete osupdaterun <resourceid> --project some-project`

// Template-based output constants for standardization
const (
	DEFAULT_OSUPDATERUN_FORMAT = "table{{str .Name}}\t{{str .ResourceId}}\t{{str .Status}}\t{{str .AppliedPolicy.Name}}\t{{formatTime .StartTime}}\t{{formatTime .EndTime}}"
	// Verbose table: includes description and policy
	DEFAULT_OSUPDATERUN_VERBOSE_FORMAT = "table{{str .Name}}\t{{str .ResourceId}}\t{{str .Status}}\t{{str .AppliedPolicy.Name}}\t{{str .Description}}\t{{formatTime .StartTime}}\t{{formatTime .EndTime}}"
	// Detailed single-get format (multiline key: value)
	DEFAULT_OSUPDATERUN_GET_FORMAT     = "Name:\t{{str .Name}}\nResource ID:\t{{str .ResourceId}}\nStatus:\t{{str .Status}}\nStatus Detail:\t{{str .StatusDetails}}\nApplied Policy:\t{{str .AppliedPolicy.Name}}\nDescription:\t{{str .Description}}\nStart Time:\t{{formatTime .StartTime}}\nEnd Time:\t{{formatTime .EndTime}}\n"
	OSUPDATERUN_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_OSUPDATERUN_OUTPUT_TEMPLATE"
)

func getOSUpdateRunOutputFormat(cmd *cobra.Command, verbose bool, forList bool) (string, error) {
	if verbose && forList {
		return DEFAULT_OSUPDATERUN_VERBOSE_FORMAT, nil
	}
	if !forList {
		// For single-get, return the detailed get format but allow overrides via flags/env
		return resolveTableOutputTemplate(cmd, DEFAULT_OSUPDATERUN_GET_FORMAT, OSUPDATERUN_OUTPUT_TEMPLATE_ENVVAR)
	}
	return resolveTableOutputTemplate(cmd, DEFAULT_OSUPDATERUN_FORMAT, OSUPDATERUN_OUTPUT_TEMPLATE_ENVVAR)
}

func printOSUpdateRuns(cmd *cobra.Command, writer io.Writer, runs []infra.OSUpdateRun, orderBy *string, outputFilter *string, verbose bool) error {
	outputFormat, err := getOSUpdateRunOutputFormat(cmd, verbose, true)
	if err != nil {
		return err
	}
	outputType, _ := cmd.Flags().GetString("output-type")
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
		Data:      runs,
	}
	GenerateOutput(writer, &result)
	return nil
}

func printOSUpdateRun(cmd *cobra.Command, writer io.Writer, run *infra.OSUpdateRun) error {
	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getOSUpdateRunOutputFormat(cmd, false, false)
	if err != nil {
		return err
	}
	result := CommandResult{
		Format:    format.Format(outputFormat),
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      run,
	}
	GenerateOutput(writer, &result)
	return nil
}

func getGetOSUpdateRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdaterun <name> [flags]",
		Short:   "Get an OS Update run",
		Example: getOSUpdateRunExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: osUpdateRunAliases,
		RunE:    runGetOSUpdateRunCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getListOSUpdateRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdaterun [flags]",
		Short:   "List all OS Update policies",
		Example: listOSUpdateRunExamples,
		Aliases: osUpdateRunAliases,
		RunE:    runListOSUpdateRunCommand,
	}
	cmd.Flags().StringP("filter", "f", viper.GetString("filter"), "API filter (see https://google.aip.dev/160)")
	cmd.Flags().String("order-by", "", "order results by field (table output only)")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getDeleteOSUpdateRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdaterun <name> [flags]",
		Short:   "Delete an OS Update run",
		Example: deleteOSUpdateRunExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: osUpdateRunAliases,
		RunE:    runDeleteOSUpdateRunCommand,
	}
	return cmd
}

// Gets specific OSUpdateRun - retrieves list of policies and then filters and outputs
// specifc run by name
func runGetOSUpdateRunCommand(cmd *cobra.Command, args []string) error {
	uprun := args[0]
	writer, _ := getOutputContext(cmd)
	ctx, OSUpdateRunClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}
	resp, err := OSUpdateRunClient.OSUpdateRunGetOSUpdateRunWithResponse(ctx, projectName,
		uprun, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, "error getting OS Update run"); err != nil {
		return err
	}
	if err := printOSUpdateRun(cmd, writer, resp.JSON200); err != nil {
		return err
	}
	return writer.Flush()
}

// Lists all OS Update policies - retrieves all policies and displays selected information in tabular format
func runListOSUpdateRunCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	// filter helper not needed; validation uses API probe
	ctx, OSUpdateRunClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}
	//TODO handle multiple pages
	validatedFilter, err := getValidatedOSUpdateRunFilter(ctx, cmd, OSUpdateRunClient, projectName)
	if err != nil {
		return err
	}

	resp, err := OSUpdateRunClient.OSUpdateRunListOSUpdateRunWithResponse(ctx, projectName,
		&infra.OSUpdateRunListOSUpdateRunParams{
			Filter: validatedFilter,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting OS Update Runs"); !proceed {
		return err
	}
	validatedOrderBy, err := getValidatedOSUpdateRunOrderBy(ctx, cmd, OSUpdateRunClient, projectName)
	if err != nil {
		return err
	}
	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printOSUpdateRuns(cmd, writer, resp.JSON200.OsUpdateRuns, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}
	return writer.Flush()

}

// Deletes OS Update Run - checks if a run  already exists and then deletes it if it does
func runDeleteOSUpdateRunCommand(cmd *cobra.Command, args []string) error {
	osrun := args[0]

	ctx, OSUpdateRunClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := OSUpdateRunClient.OSUpdateRunDeleteOSUpdateRunWithResponse(ctx, projectName,
		osrun, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting OS Update run %s", osrun))
}

// Validates the order-by argument for OSUpdateRun and provides hints for valid fields
func getValidatedOSUpdateRunOrderBy(ctx interface{}, cmd *cobra.Command, OSUpdateRunClient infra.ClientWithResponsesInterface, projectName string) (*string, error) {
	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return nil, err
	}
	outputType, _ := cmd.Flags().GetString("output-type")
	// For table format (default), use client-side sorting which supports any field in the model
	if outputType == "table" {
		normalized, err := normalizeOrderByForClientSorting(raw, infra.OSUpdateRun{})
		if err != nil {
			return nil, err
		}
		return normalized, nil
	}
	// For JSON/YAML, use API ordering (only API-supported fields)
	return normalizeOrderByWithAPIProbe(raw, "os-update-runs", infra.OSUpdateRun{}, func(orderBy string) (bool, error) {
		pageSize := 1
		offset := 0
		resp, err := OSUpdateRunClient.OSUpdateRunListOSUpdateRunWithResponse(ctx.(context.Context), projectName,
			&infra.OSUpdateRunListOSUpdateRunParams{
				OrderBy:  &orderBy,
				Filter:   nil,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == 400 {
			return false, &api400Error{string(resp.Body)}
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating OS Update Run order-by"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func getValidatedOSUpdateRunFilter(
	ctx context.Context,
	cmd *cobra.Command,
	OSUpdateRunClient infra.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("filter")
	if err != nil {
		return nil, err
	}

	return normalizeFilterWithAPIProbe(raw, "os-update-runs", infra.OSUpdateRun{}, func(filter string) (bool, error) {
		pageSize := 1
		offset := 0
		resp, err := OSUpdateRunClient.OSUpdateRunListOSUpdateRunWithResponse(ctx, projectName,
			&infra.OSUpdateRunListOSUpdateRunParams{
				OrderBy:  nil,
				Filter:   &filter,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == 400 {
			return false, &api400Error{string(resp.Body)}
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating OS Update Run filter"); err != nil {
			return false, err
		}
		return true, nil
	})
}
