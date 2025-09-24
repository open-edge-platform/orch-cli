// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
)

const listOSUpdateRunExamples = `# List all OS Update Policies
orch-cli list osupdaterun --project some-project
`

const getOSUpdateRunExamples = `# Get detailed information about specific OS Update Run using the run name
orch-cli get osupdaterun <resourceid> --project some-project`

const deleteOSUpdateRunExamples = `#Delete an OS Update Run  using it's name
orch-cli delete osupdaterun <resourceid> --project some-project`

var OSUpdateRunHeader = fmt.Sprintf("\n%s\t%s\t%s", "Name", "Resource ID", "Status")

// Prints OS Profiles in tabular format
func printOSUpdateRuns(writer io.Writer, OSUpdateRuns []infra.OSUpdateRun, verbose bool) {
	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%v\t%s\t%s\n", "Name", "Resource ID", "Status", "Applied Policy", "Start Time", "End Time")
	}

	for _, run := range OSUpdateRuns {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", *run.Name, *run.ResourceId, *run.Status)
		} else {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%v\t%s\t%s\n", *run.Name, *run.ResourceId, *run.Status, run.AppliedPolicy.Name, *run.StartTime, *run.EndTime)
		}
	}
}

// Prints output details of OS Profiles
func printOSUpdateRun(writer io.Writer, OSUpdateRun *infra.OSUpdateRun) {

	_, _ = fmt.Fprintf(writer, "Name: \t%s\n", *OSUpdateRun.Name)
	_, _ = fmt.Fprintf(writer, "ResourceID: \t%s\n", *OSUpdateRun.ResourceId)
	_, _ = fmt.Fprintf(writer, "Status: \t%s\n", *OSUpdateRun.Status)
	_, _ = fmt.Fprintf(writer, "Status Detail: \t%s\n", *OSUpdateRun.StatusDetails)
	_, _ = fmt.Fprintf(writer, "Applied Policy: \t%v\n", OSUpdateRun.AppliedPolicy.Name)
	_, _ = fmt.Fprintf(writer, "Description: \t%v\n", *OSUpdateRun.Description)
	_, _ = fmt.Fprintf(writer, "Start Time: \t%s\n", *OSUpdateRun.StartTime)
	_, _ = fmt.Fprintf(writer, "End Time: \t%s\n", *OSUpdateRun.StartTime)
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

	writer, verbose := getOutputContext(cmd)
	ctx, OSUpdateRunClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := OSUpdateRunClient.OSUpdateRunGetOSUpdateRunWithResponse(ctx, projectName,
		uprun, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		OSProfileHeaderGet, "error getting OS Update run"); !proceed {
		return err
	}

	printOSUpdateRun(writer, resp.JSON200)
	return writer.Flush()
}

// Lists all OS Update policies - retrieves all policies and displays selected information in tabular format
func runListOSUpdateRunCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	filtflag, _ := cmd.Flags().GetString("filter")
	filter := filterHelper(filtflag)

	ctx, OSUpdateRunClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	//TODO handle multiple pages
	resp, err := OSUpdateRunClient.OSUpdateRunListOSUpdateRunWithResponse(ctx, projectName,
		&infra.OSUpdateRunListOSUpdateRunParams{
			Filter: filter,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		OSUpdateRunHeader, "error getting OS Update Runs"); !proceed {
		return err
	}

	printOSUpdateRuns(writer, resp.JSON200.OsUpdateRuns, verbose)
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
