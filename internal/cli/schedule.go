// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
)

const listScheduleExamples = `# List all schedule resources
orch-cli list schedule --project some-project
`

const getScheduleExamples = `# Get detailed information about specific schedule resource using it's name
orch-cli get schedule myschedule --project some-project`

const createScheduleExamples = `# Create a new schedule resource 
orch-cli create schedule myschedule --project some-project`

const deleteScheduleExamples = `# Delete a schedule resource using it's name
orch-cli delete schedule myschedule --project some-project`

var ScheduleHeader = fmt.Sprintf("\n%s\t%s\t%s", "Name", "Host", "Type")

// Prints SSH keys in tabular format
func printSchedules(writer io.Writer, singleSchedules []infra.SingleScheduleResource, repeatedSchedules []infra.RepeatedScheduleResource, verbose bool) {

	status := "Unspecified"
	maintenanceType := "Unspecified"

	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%s\n", "Name", "Host", "Resource ID", "Type")
	}

	for _, schedule := range singleSchedules {

		if schedule.ScheduleStatus == infra.SCHEDULESTATUSMAINTENANCE {
			status = "Maintenance"
		} else if schedule.ScheduleStatus == infra.SCHEDULESTATUSOSUPDATE {
			status = "OS Update"
		}
		maintenanceType = "single"

		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", *schedule.Name, *schedule.TargetHostId, status)
		} else {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", *schedule.Name, *schedule.TargetHostId, *schedule.ResourceId, maintenanceType)
		}
	}
	for _, schedule := range repeatedSchedules {

		if schedule.ScheduleStatus == infra.SCHEDULESTATUSMAINTENANCE {
			status = "Maintenance"
		} else if schedule.ScheduleStatus == infra.SCHEDULESTATUSOSUPDATE {
			status = "OS Update"
		}
		maintenanceType = "repeated"

		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", *schedule.Name, *schedule.TargetHostId, status)
		} else {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", *schedule.Name, *schedule.TargetHostId, *schedule.ResourceId, maintenanceType)
		}
	}
}

// Prints output details of SSH key
func printSchedule(writer io.Writer, singleSchedules infra.SingleScheduleResource, repeatedSchedules infra.RepeatedScheduleResource) {

	_, _ = fmt.Fprintf(writer, "Name: \t%s\n", singleSchedules.Name)
	_, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *singleSchedules.ResourceId)
	// _, _ = fmt.Fprintf(writer, "Type: \t%s\n", singleSchedules.Type)
	// _, _ = fmt.Fprintf(writer, "Date: \t%s\n", singleSchedules.Date)
	// _, _ = fmt.Fprintf(writer, "Time: \t%s\n", singleSchedules.Time)
}

// Filters list of schedules to find one with specific name
func findScheduleType(singleSchedules []infra.SingleScheduleResource, repeatedSchedules []infra.RepeatedScheduleResource, id string) (string, error) {
	for _, schedule := range singleSchedules {
		if schedule.ResourceId == id {
			return "single", nil
		}
	}
	for _, schedule := range repeatedSchedules {
		if schedule.ResourceId == id {
			return "repeated", nil
		}
	}
	return "", errors.New("no schedule matches the given id")
}

// // Helper function to verify that the input file exists and is of right format
// func verifySSHUserName(n string) error {

// 	pattern := `^[a-z][a-z0-9-]{0,31}$`

// 	// Compile the regular expression
// 	re := regexp.MustCompile(pattern)

// 	// Match the input string against the pattern
// 	if re.MatchString(n) {
// 		return nil
// 	}
// 	return errors.New("input is not a valid SSH username")
// }

func getGetScheduleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedule <name> [flags]",
		Short:   "Get a schedule configuration",
		Example: getScheduleExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: scheduleAliases,
		RunE:    runGetScheduleCommand,
	}
	return cmd
}

func getListScheduleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedule [flags]",
		Short:   "List all schedule configurations",
		Example: listScheduleExamples,
		Aliases: scheduleAliases,
		RunE:    runListScheduleCommand,
	}
	return cmd
}

func getCreateScheduleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedule [flags]",
		Short:   "Creates a schedule configuration",
		Example: createScheduleExamples,
		Args:    cobra.ExactArgs(2),
		Aliases: scheduleAliases,
		RunE:    runCreateScheduleCommand,
	}
	return cmd
}

func getDeleteScheduleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedule <name> [flags]",
		Short:   "Delete a schedule configuration",
		Example: deleteScheduleExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: scheduleAliases,
		RunE:    runDeleteScheduleCommand,
	}
	return cmd
}

// Gets specific schedule configuration by resource ID
func runGetScheduleCommand(cmd *cobra.Command, args []string) error {

	writer, verbose := getOutputContext(cmd)
	ctx, scheduleClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := scheduleClient.ScheduleServiceListSchedulesWithResponse(ctx, projectName,
		&infra.ScheduleServiceListSchedulesParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting schedules"); !proceed {
		return err
	}

	id := args[0]
	scheduleType, err := findScheduleType(resp.JSON200.SingleSchedules, resp.JSON200.RepeatedSchedules, id)
	if err != nil {
		return err
	}

	if scheduleType != "" && scheduleType == "single" {

	} else if scheduleType != "" && scheduleType == "repeated" {

	}

	if err := checkResponse(resp.HTTPResponse, "error while retrieving schedule"); err != nil {
		return err
	}

	// pageSize := 20
	// instances := make([]infra.InstanceResource, 0)
	// for offset := 0; ; offset += pageSize {
	// 	iresp, err := sshKeyClient.InstanceServiceListInstancesWithResponse(ctx, projectName,
	// 		&infra.InstanceServiceListInstancesParams{}, auth.AddAuthHeader)
	// 	if err != nil {
	// 		return processError(err)
	// 	}
	// 	if err := checkResponse(iresp.HTTPResponse, "error while retrieving instances"); err != nil {
	// 		return err
	// 	}

	// 	instances = append(instances, iresp.JSON200.Instances...)
	// 	if !iresp.JSON200.HasNext {
	// 		break // No more instances to process
	// 	}
	// }

	printSchedule(writer, *schedule, infra.RepeatedScheduleResource{})
	return writer.Flush()
}

// Lists all schedules - retrieves all schedules and displays selected information in tabular format
func runListScheduleCommand(cmd *cobra.Command, _ []string) error {

	writer, verbose := getOutputContext(cmd)

	ctx, scheduleClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := scheduleClient.ScheduleServiceListSchedulesWithResponse(ctx, projectName,
		&infra.ScheduleServiceListSchedulesParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		ScheduleHeader, "error getting schedules"); !proceed {
		return err
	}

	printSchedules(writer, resp.JSON200.SingleSchedules, resp.JSON200.RepeatedSchedules, verbose)

	return writer.Flush()
}

// Creates SSH key configuration
func runCreateScheduleCommand(cmd *cobra.Command, args []string) error {
	name := args[0]

	// err := verifySSHUserName(name)
	// if err != nil {
	// 	return err
	// }

	ctx, sshKeyClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := sshKeyClient.ScheduleServiceCreateSingleScheduleWithResponse(ctx, projectName,
		infra.ScheduleServiceCreateSingleScheduleJSONRequestBody{
			// Name: name,
			// TargetHost: name,

		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating schedule %s", name))
}

// Deletes SSH Key - checks if a key already exists and then deletes it if it does
func runDeleteScheduleCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	ctx, sshKeyClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	gresp, err := sshKeyClient.ScheduleServiceListSchedulesWithResponse(ctx, projectName,
		&infra.ScheduleServiceListSchedulesParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err = checkResponse(gresp.HTTPResponse, "Error getting schedules"); err != nil {
		return err
	}

	schedule, err := findSchedulesByName(gresp.JSON200.SingleSchedules, gresp.JSON200.RepeatedSchedules, name)
	if err != nil {
		return err
	}

	resp, err := sshKeyClient.ScheduleServiceDeleteSingleScheduleWithResponse(ctx, projectName,
		*schedule.ResourceId, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting schedule %s", name))
}
