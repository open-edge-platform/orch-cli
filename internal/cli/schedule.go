// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time" // Add this import

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
func printSchedule(writer io.Writer, singleSchedule infra.SingleScheduleResource, repeatedSchedule infra.RepeatedScheduleResource, loc *time.Location) {
	// Load the timezone

	if singleSchedule.ResourceId != nil {
		// Convert Unix timestamps to readable format in specified timezone
		startTime := ""
		endTime := ""
		if singleSchedule.StartSeconds != 0 {
			startTime = time.Unix(int64(singleSchedule.StartSeconds), 0).In(loc).Format("2006-01-02 15:04:05 MST")
		}
		if singleSchedule.EndSeconds != nil && *singleSchedule.EndSeconds != 0 {
			endTime = time.Unix(int64(*singleSchedule.EndSeconds), 0).In(loc).Format("2006-01-02 15:04:05 MST")
		}

		_, _ = fmt.Fprintf(writer, "Name: \t%s\n", *singleSchedule.Name)
		_, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *singleSchedule.ResourceId)
		_, _ = fmt.Fprintf(writer, "Target Host ID: \t%s\n", *singleSchedule.TargetHostId)
		_, _ = fmt.Fprintf(writer, "Target Region ID: \t%s\n", *singleSchedule.TargetRegionId)
		_, _ = fmt.Fprintf(writer, "Target Site ID: \t%s\n", *singleSchedule.TargetSiteId)
		_, _ = fmt.Fprintf(writer, "Schedule Status: \t%s\n", singleSchedule.ScheduleStatus)
		_, _ = fmt.Fprintf(writer, "Start Time: \t%s\n", startTime)
		_, _ = fmt.Fprintf(writer, "End Time: \t%s\n", endTime)
	}

	if repeatedSchedule.ResourceId != nil {
		_, _ = fmt.Fprintf(writer, "Name: \t%s\n", *repeatedSchedule.Name)
		_, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *repeatedSchedule.ResourceId)
		_, _ = fmt.Fprintf(writer, "Target Host ID: \t%s\n", *repeatedSchedule.TargetHostId)
		_, _ = fmt.Fprintf(writer, "Target Region ID: \t%s\n", *repeatedSchedule.TargetRegionId)
		_, _ = fmt.Fprintf(writer, "Target Site ID: \t%s\n", *repeatedSchedule.TargetSiteId)
		_, _ = fmt.Fprintf(writer, "Schedule Status: \t%s\n", repeatedSchedule.ScheduleStatus)
		_, _ = fmt.Fprintf(writer, "Month: \t%s\n", repeatedSchedule.CronDayMonth)
		_, _ = fmt.Fprintf(writer, "Day: \t%s\n", repeatedSchedule.CronDayMonth)
		_, _ = fmt.Fprintf(writer, "Weekday: \t%s\n", repeatedSchedule.CronDayWeek)
		// Show both UTC and converted times
		_, _ = fmt.Fprintf(writer, "Hour (UTC): \t%s\n", repeatedSchedule.CronHours)
		_, _ = fmt.Fprintf(writer, "Minute (UTC): \t%s\n", repeatedSchedule.CronMinutes)

		// Convert and show timezone-adjusted time
		if loc.String() != "UTC" {
			convertedHour, convertedMinute, err := convertCronTimeToTimezone(repeatedSchedule.CronHours, repeatedSchedule.CronMinutes, loc)
			if err == nil {
				_, _ = fmt.Fprintf(writer, "Hour (%s): \t%s\n", loc.String(), convertedHour)
				_, _ = fmt.Fprintf(writer, "Minute (%s): \t%s\n", loc.String(), convertedMinute)
				_, _ = fmt.Fprintf(writer, "Local Time: \t%s:%s %s\n", convertedHour, convertedMinute, loc.String())
			}
		}
		_, _ = fmt.Fprintf(writer, "Duration: \t%d seconds\n", repeatedSchedule.DurationSeconds)
	}
}

// Helper function to convert UTC cron time to target timezone
func convertCronTimeToTimezone(cronHour, cronMinute string, targetLoc *time.Location) (string, string, error) {
	// Parse the cron hour and minute
	hour, err := strconv.Atoi(cronHour)
	if err != nil {
		return cronHour, cronMinute, fmt.Errorf("invalid hour format: %s", cronHour)
	}

	minute, err := strconv.Atoi(cronMinute)
	if err != nil {
		return cronHour, cronMinute, fmt.Errorf("invalid minute format: %s", cronMinute)
	}

	// Create a time in UTC for today with the cron hour/minute
	now := time.Now().UTC()
	utcTime := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.UTC)

	// Convert to target timezone
	targetTime := utcTime.In(targetLoc)

	return fmt.Sprintf("%d", targetTime.Hour()), fmt.Sprintf("%d", targetTime.Minute()), nil
}

// Filters list of schedules to find one with specific name
func findSchedule(singleSchedules []infra.SingleScheduleResource, repeatedSchedules []infra.RepeatedScheduleResource, id string) (infra.SingleScheduleResource, infra.RepeatedScheduleResource, error) {
	for _, schedule := range singleSchedules {
		if *schedule.ResourceId == id {
			return schedule, infra.RepeatedScheduleResource{}, nil
		}
	}
	for _, schedule := range repeatedSchedules {
		if *schedule.ResourceId == id {
			return infra.SingleScheduleResource{}, schedule, nil
		}
	}
	return infra.SingleScheduleResource{}, infra.RepeatedScheduleResource{}, errors.New("no schedule matches the given id")
}

// parseTargetResource parses a target string in format "type-resourceid" and returns appropriate target pointers
func parseTargetResource(target string) (hostname, region, site *string, err error) {
	if target == "" {
		return nil, nil, nil, errors.New("target must be specified")
	}

	// Split target string at the first '-' character
	parts := strings.SplitN(target, "-", 2)
	if len(parts) != 2 {
		return nil, nil, nil, errors.New("target must be in format 'resource-id' (e.g., host-abcd1234)")
	}

	targetType := parts[0]

	if targetType == "host" {
		return &target, nil, nil, nil
	} else if targetType == "region" {
		return nil, &target, nil, nil
	} else if targetType == "site" {
		return nil, nil, &target, nil
	} else {
		return nil, nil, nil, fmt.Errorf("invalid target type '%s', must be one of: host, region, site", targetType)
	}
}

// validateStartTimeFormat validates that the start time is in the correct format "YYYY-MM-DD HH:MM"
func validateStartTimeFormat(startTime string) bool {
	// Expected format: "YYYY-MM-DD HH:MM"
	const timeFormat = "2006-01-02 15:04"

	_, err := time.Parse(timeFormat, startTime)
	return err == nil
}

// getTimeInSeconds converts a time string and timezone to Unix timestamp
func getTimeInSeconds(timeStr, timezone string) int64 {
	const timeFormat = "2006-01-02 15:04"

	// Load the specified timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Fallback to UTC if timezone is invalid
		loc = time.UTC
	}

	// Parse the time in the specified timezone
	t, err := time.ParseInLocation(timeFormat, timeStr, loc)
	if err != nil {
		return 0
	}

	return t.Unix()
}

func getGetScheduleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedule <name> [flags]",
		Short:   "Get a schedule configuration",
		Example: getScheduleExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: scheduleAliases,
		RunE:    runGetScheduleCommand,
	}
	cmd.PersistentFlags().StringP("timezone", "t", viper.GetString("timezone"), "Display time in particular timezone: --timezone Europe/Berlin")
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
		Args:    cobra.ExactArgs(1),
		Aliases: scheduleAliases,
		RunE:    runCreateScheduleCommand,
	}
	cmd.PersistentFlags().StringP("frequency-type", "F", viper.GetString("frequency-type"), "Frequency of the schedule: --frequency-type single|repeated")
	cmd.PersistentFlags().StringP("maintenance-type", "m", viper.GetString("maintenance-type"), "Type of maintenance: --maintenance-type maintenance|osupdate")
	cmd.PersistentFlags().StringP("timezone", "t", viper.GetString("timezone"), "Set time in particular timezone: --timezone Europe/Berlin")
	cmd.PersistentFlags().StringP("target", "T", viper.GetString("target"), "Target maintenance on a host|region|site using it's resource ID: --target host-abcd1234|region-abcd1234|site-abcd1234")
	cmd.PersistentFlags().StringP("start-time", "s", viper.GetString("start-time"), "Start time of the schedule: --start-time \"2025-12-15 12:00\"")
	cmd.PersistentFlags().StringP("end-time", "e", viper.GetString("end-time"), "End time of the schedule: --end-time \"2025-12-15 14:00\"")
	cmd.PersistentFlags().StringP("frequency", "f", viper.GetString("frequency"), "Frequency of the schedule: --frequency daily|weekly|monthly")
	cmd.PersistentFlags().StringP("day-of-week", "d", viper.GetString("day-of-week"), "Day of the week for repeated schedule: --day-of-week \"mon,tue,wed,thu,fri,sat,sun\"")
	cmd.PersistentFlags().StringP("day-of-month", "D", viper.GetString("day-of-month"), "Day of the month for repeated schedule: --day-of-month 1-31")
	cmd.PersistentFlags().StringP("hour", "H", viper.GetString("hour"), "Hour of the day for repeated schedule (0-23): --hour 2")
	cmd.PersistentFlags().StringP("minute", "M", viper.GetString("minute"), "Minute of the hour for repeated schedule (0-59): --minute 30")
	cmd.PersistentFlags().IntP("duration", "u", viper.GetInt("duration"), "Duration of the maintenance window in seconds: --duration 3600")

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
	timezone, _ := cmd.Flags().GetString("timezone")
	loc := time.UTC
	var err error

	if timezone != "" {
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return fmt.Errorf("invalid timezone '%s': %w", timezone, err)
		}
	}
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

	if err := checkResponse(resp.HTTPResponse, "error while retrieving schedule"); err != nil {
		return err
	}

	singleSchedule, repeatedSchedule, err := findSchedule(resp.JSON200.SingleSchedules, resp.JSON200.RepeatedSchedules, id)
	if err != nil {
		return err
	}

	printSchedule(writer, singleSchedule, repeatedSchedule, loc)
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

	timezone, _ := cmd.Flags().GetString("timezone")
	scheduleType, _ := cmd.Flags().GetString("frequency-type")
	maintenanceType, _ := cmd.Flags().GetString("maintenance-type")
	target, _ := cmd.Flags().GetString("target")

	// Parameters for single schedule
	startTime, _ := cmd.Flags().GetString("start-time")
	endTime, _ := cmd.Flags().GetString("end-time")

	// Parameters for repeated schedule
	// frequency, _ := cmd.Flags().GetString("frequency")
	// dayOfWeek, _ := cmd.Flags().GetString("day-of-week")
	// dayOfMonth, _ := cmd.Flags().GetString("day-of-month")
	// hour, _ := cmd.Flags().GetString("hour")
	// minute, _ := cmd.Flags().GetString("minute")
	// duration, _ := cmd.Flags().GetInt("duration")

	// Validate timezone
	if timezone == "" {
		return errors.New("timezone must be specified")
	}

	_, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone '%s': %w", timezone, err)
	}

	if scheduleType != "single" && scheduleType != "repeated" {
		return errors.New("invalid schedule type, must be 'single' or 'repeated'")
	}

	if maintenanceType != "maintenance" && maintenanceType != "osupdate" {
		return errors.New("invalid maintenance type, must be 'maintenance' or 'osupdate'")
	} else if maintenanceType == "osupdate" {
		maintenanceType = string(infra.SCHEDULESTATUSOSUPDATE)
	} else if maintenanceType == "maintenance" {
		maintenanceType = string(infra.SCHEDULESTATUSMAINTENANCE)
	}

	// Parse target resource
	hostname, region, site, err := parseTargetResource(target)
	if err != nil {
		return err
	}

	ctx, scheduleClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	// Repeated schedule logic
	if scheduleType == "repeated" {
		resp, err := scheduleClient.ScheduleServiceCreateRepeatedScheduleWithResponse(ctx, projectName,
			infra.ScheduleServiceCreateRepeatedScheduleJSONRequestBody{
				Name: &name,
				// TargetHost: name,

			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating schedule %s", name))
	}

	//Single schedule logic

	if scheduleType == "single" {

		if startTime == "" || !validateStartTimeFormat(startTime) {
			return errors.New("start-time must be specified in format \"YYYY-MM-DD HH:MM\"")
		}
		startSeconds := getTimeInSeconds(startTime, timezone)

		var endSeconds *int
		if endTime != "" {
			if !validateStartTimeFormat(endTime) {
				return errors.New("end-time must be in format \"YYYY-MM-DD HH:MM\"")
			}
			endSec := int(getTimeInSeconds(endTime, timezone))
			endSeconds = &endSec
		} else {
			fmt.Printf("End time not specified, maintenance window will be open ended\n")
		}

		resp, err := scheduleClient.ScheduleServiceCreateSingleScheduleWithResponse(ctx, projectName,
			infra.ScheduleServiceCreateSingleScheduleJSONRequestBody{
				Name:           &name,
				ScheduleStatus: infra.ScheduleStatus(maintenanceType),
				StartSeconds:   int(startSeconds),
				EndSeconds:     endSeconds,
				TargetHostId:   hostname,
				TargetRegionId: region,
				TargetSiteId:   site,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating schedule %s", name))
	}
	return errors.New("cannot create schedule")
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

	singleSchedule, repeatedSchedule, err := findSchedule(gresp.JSON200.SingleSchedules, gresp.JSON200.RepeatedSchedules, name)
	if err != nil {
		return err
	}

	if singleSchedule.ResourceId != nil {
		resp, err := sshKeyClient.ScheduleServiceDeleteSingleScheduleWithResponse(ctx, projectName,
			*singleSchedule.ResourceId, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting schedule %s", name))
	}

	if repeatedSchedule.ResourceId != nil {
		resp, err := sshKeyClient.ScheduleServiceDeleteRepeatedScheduleWithResponse(ctx, projectName,
			*repeatedSchedule.ResourceId, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting schedule %s", name))
	}
	return errors.New("no schedule matches the given id")
}
