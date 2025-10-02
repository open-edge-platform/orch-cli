// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"
	"sort"
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

const createScheduleExamples = `# Create a new repeated schedule, an osupdate , using days of week
orch-cli create schedules my-schedule --timezone GMT --frequency-type repeated --maintenance-type osupdate --target site-532d1d07 --frequency weekly --start-time "10:10" --day-of-week "1-3,5" --months "2,4,7-8" --duration 3600

# Create a new repeated schedule, a maintenance , using days of month
orch-cli create schedules my-schedule --timezone GMT  --frequency-type repeated  --maintenance-type maintenance --target site-532d1d07 --frequency monthly --start-time "10:10" --day-of-month "1,6,31" --months "2,4,7-12" --duration 3600

# Create a new single schedule, an osupdate
orch-cli create schedules my-schedule --timezone GMT --frequency-type single --maintenance-type osupdate --target region-65c0d433 --start-time "2026-12-01 20:20" --end-time "2027-12-01 20:20"
`

const deleteScheduleExamples = `# Delete a schedule resource using it's name
orch-cli delete schedule myschedule --project some-project`

var ScheduleHeader = fmt.Sprintf("\n%s\t%s\t%s", "Name", "Target", "Type")

const SINGLE = 0
const REPEATED = 1

// Prints SSH keys in tabular format
func printSchedules(writer io.Writer, singleSchedules []infra.SingleScheduleResource, repeatedSchedules []infra.RepeatedScheduleResource, verbose bool) {

	status := "Unspecified"
	var maintenanceType string

	target := "Unspecified"

	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%s\n", "Name", "Target", "Resource ID", "Type")
	}

	for _, schedule := range singleSchedules {

		if schedule.TargetHostId != nil && *schedule.TargetHostId != "" {
			target = *schedule.TargetHostId
		} else if schedule.TargetRegionId != nil && *schedule.TargetRegionId != "" {
			target = *schedule.TargetRegionId
		} else if schedule.TargetSiteId != nil && *schedule.TargetSiteId != "" {
			target = *schedule.TargetSiteId
		}
		if schedule.ScheduleStatus == infra.SCHEDULESTATUSMAINTENANCE {
			status = "Maintenance"
		} else if schedule.ScheduleStatus == infra.SCHEDULESTATUSOSUPDATE {
			status = "OS Update"
		}
		maintenanceType = "single"

		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", *schedule.Name, target, status)
		} else {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", *schedule.Name, target, *schedule.ResourceId, maintenanceType)
		}
	}
	for _, schedule := range repeatedSchedules {
		if schedule.TargetHostId != nil && *schedule.TargetHostId != "" {
			target = *schedule.TargetHostId
		} else if schedule.TargetRegionId != nil && *schedule.TargetRegionId != "" {
			target = *schedule.TargetRegionId
		} else if schedule.TargetSiteId != nil && *schedule.TargetSiteId != "" {
			target = *schedule.TargetSiteId
		}
		if schedule.ScheduleStatus == infra.SCHEDULESTATUSMAINTENANCE {
			status = "Maintenance"
		} else if schedule.ScheduleStatus == infra.SCHEDULESTATUSOSUPDATE {
			status = "OS Update"
		}
		maintenanceType = "repeated"

		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", *schedule.Name, target, status)
		} else {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", *schedule.Name, target, *schedule.ResourceId, maintenanceType)
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

		tHost := "Unspecified"
		tSite := "Unspecified"
		tRegion := "Unspecified"
		if singleSchedule.TargetHostId != nil && *singleSchedule.TargetHostId != "" {
			tHost = *singleSchedule.TargetHostId
		} else if singleSchedule.TargetRegionId != nil && *singleSchedule.TargetRegionId != "" {
			tRegion = *singleSchedule.TargetRegionId
		} else if singleSchedule.TargetSiteId != nil && *singleSchedule.TargetSiteId != "" {
			tSite = *singleSchedule.TargetSiteId
		}

		_, _ = fmt.Fprintf(writer, "Name: \t%s\n", *singleSchedule.Name)
		_, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *singleSchedule.ResourceId)
		_, _ = fmt.Fprintf(writer, "Target Host ID: \t%s\n", tHost)
		_, _ = fmt.Fprintf(writer, "Target Region ID: \t%s\n", tRegion)
		_, _ = fmt.Fprintf(writer, "Target Site ID: \t%s\n", tSite)
		_, _ = fmt.Fprintf(writer, "Schedule Status: \t%s\n", singleSchedule.ScheduleStatus)
		_, _ = fmt.Fprintf(writer, "Start Time: \t%s\n", startTime)
		_, _ = fmt.Fprintf(writer, "End Time: \t%s\n", endTime)
	}

	if repeatedSchedule.ResourceId != nil {

		tHost := "Unspecified"
		tSite := "Unspecified"
		tRegion := "Unspecified"
		if repeatedSchedule.TargetHostId != nil && *repeatedSchedule.TargetHostId != "" {
			tHost = *repeatedSchedule.TargetHostId
		} else if repeatedSchedule.TargetRegionId != nil && *repeatedSchedule.TargetRegionId != "" {
			tRegion = *repeatedSchedule.TargetRegionId
		} else if repeatedSchedule.TargetSiteId != nil && *repeatedSchedule.TargetSiteId != "" {
			tSite = *repeatedSchedule.TargetSiteId
		}
		_, _ = fmt.Fprintf(writer, "Name: \t%s\n", *repeatedSchedule.Name)
		_, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *repeatedSchedule.ResourceId)
		_, _ = fmt.Fprintf(writer, "Target Host ID: \t%s\n", tHost)
		_, _ = fmt.Fprintf(writer, "Target Region ID: \t%s\n", tRegion)
		_, _ = fmt.Fprintf(writer, "Target Site ID: \t%s\n", tSite)
		_, _ = fmt.Fprintf(writer, "Schedule Status: \t%s\n", repeatedSchedule.ScheduleStatus)
		_, _ = fmt.Fprintf(writer, "Month: \t%s\n", repeatedSchedule.CronMonth)
		_, _ = fmt.Fprintf(writer, "Month day: \t%s\n", repeatedSchedule.CronDayMonth)
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
	}
	return nil, nil, nil, fmt.Errorf("invalid target type '%s', must be one of: host, region, site", targetType)

}

// validateStartTimeFormat validates that the start time is in the correct format "YYYY-MM-DD HH:MM"
func validateStartTimeFormat(startTime string, m int) bool {
	const sTimeFormat = "2006-01-02 15:04"
	const rTimeFormat = "15:04"

	if m == 0 {
		_, err := time.Parse(sTimeFormat, startTime)
		return err == nil
	}
	if m == 1 {
		_, err := time.Parse(rTimeFormat, startTime)
		return err == nil
	}

	return false

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

// getTimeInCron converts a time string in "HH:MM" format and timezone to cron hour and minute strings
func getTimeInCron(timeStr, timezone string) (string, string) {
	const timeFormat = "15:04" // HH:MM format

	// Load the specified timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Fallback to UTC if timezone is invalid
		loc = time.UTC
	}

	// Parse the time in HH:MM format
	t, err := time.Parse(timeFormat, timeStr)
	if err != nil {
		// Return default values if parsing fails
		return "0", "0"
	}

	// Create a time in the specified timezone for today with the parsed hour/minute
	now := time.Now().In(loc)
	localTime := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, loc)

	// Convert to UTC for cron (since cron times are stored in UTC)
	utcTime := localTime.UTC()

	return fmt.Sprintf("%d", utcTime.Hour()), fmt.Sprintf("%d", utcTime.Minute())
}

// convertDayOfWeekToCron converts user-friendly day names to cron format (0-6)
// Supports individual days (mon,tue) and ranges (1-3,5)
func convertDayOfWeekToCron(dayOfWeek string) (string, error) {
	if dayOfWeek == "" {
		return "", nil // Empty is allowed for filters
	}

	// Map of day names to cron numbers (0=Sunday, 1=Monday, ..., 6=Saturday)
	dayMap := map[string]string{
		"sun": "0", "sunday": "0",
		"mon": "1", "monday": "1",
		"tue": "2", "tuesday": "2",
		"wed": "3", "wednesday": "3",
		"thu": "4", "thursday": "4",
		"fri": "5", "friday": "5",
		"sat": "6", "saturday": "6",
	}

	// Split comma-separated days/ranges
	parts := strings.Split(dayOfWeek, ",")
	daySet := make(map[int]bool) // To avoid duplicates

	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))

		// Check if it's a range (contains hyphen)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return "", fmt.Errorf("invalid range format '%s', expected format like '1-3'", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return "", fmt.Errorf("invalid start day '%s' in range '%s'", rangeParts[0], part)
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return "", fmt.Errorf("invalid end day '%s' in range '%s'", rangeParts[1], part)
			}

			// Validate range
			if start < 0 || start > 6 {
				return "", fmt.Errorf("invalid start day %d in range '%s', must be between 0-6", start, part)
			}
			if end < 0 || end > 6 {
				return "", fmt.Errorf("invalid end day %d in range '%s', must be between 0-6", end, part)
			}
			if start > end {
				return "", fmt.Errorf("invalid range '%s', start day must be less than or equal to end day", part)
			}

			// Add all days in range
			for day := start; day <= end; day++ {
				daySet[day] = true
			}
		} else {
			// Single day - check if it's already a cron number (0-6)
			if len(part) == 1 && part >= "0" && part <= "6" {
				dayNum, _ := strconv.Atoi(part)
				daySet[dayNum] = true
				continue
			}

			// Convert from day name to cron number
			if cronNum, exists := dayMap[part]; exists {
				dayNum, _ := strconv.Atoi(cronNum)
				daySet[dayNum] = true
			} else {
				return "", fmt.Errorf("invalid day '%s', must be one of: sun,mon,tue,wed,thu,fri,sat or 0-6", part)
			}
		}
	}

	// Convert set to sorted slice
	var dayList []int
	for day := range daySet {
		dayList = append(dayList, day)
	}

	// Sort the days
	sort.Ints(dayList)

	// Convert to strings
	var cronDays []string
	for _, day := range dayList {
		cronDays = append(cronDays, strconv.Itoa(day))
	}

	return strings.Join(cronDays, ","), nil
}

// convertDayOfMonthToCron converts day of month values to cron format (1-31)
// Supports individual days (1,15,31) and ranges (1-5,20-25)
func convertDayOfMonthToCron(dayOfMonth string) (string, error) {
	if dayOfMonth == "" {
		return "", nil // Empty is allowed for filters
	}

	// Split comma-separated days/ranges
	parts := strings.Split(dayOfMonth, ",")
	daySet := make(map[int]bool) // To avoid duplicates

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Check if it's a range (contains hyphen)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return "", fmt.Errorf("invalid range format '%s', expected format like '1-5'", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return "", fmt.Errorf("invalid start day '%s' in range '%s'", rangeParts[0], part)
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return "", fmt.Errorf("invalid end day '%s' in range '%s'", rangeParts[1], part)
			}

			// Validate range
			if start < 1 || start > 31 {
				return "", fmt.Errorf("invalid start day %d in range '%s', must be between 1-31", start, part)
			}
			if end < 1 || end > 31 {
				return "", fmt.Errorf("invalid end day %d in range '%s', must be between 1-31", end, part)
			}
			if start > end {
				return "", fmt.Errorf("invalid range '%s', start day must be less than or equal to end day", part)
			}

			// Add all days in range
			for day := start; day <= end; day++ {
				daySet[day] = true
			}
		} else {
			// Single day
			dayNum, err := strconv.Atoi(part)
			if err != nil {
				return "", fmt.Errorf("invalid day of month '%s', must be a number between 1-31", part)
			}

			// Validate range (1-31)
			if dayNum < 1 || dayNum > 31 {
				return "", fmt.Errorf("invalid day of month %d, must be between 1-31", dayNum)
			}

			daySet[dayNum] = true
		}
	}

	// Convert set to sorted slice
	var dayList []int
	for day := range daySet {
		dayList = append(dayList, day)
	}

	// Sort the days
	sort.Ints(dayList)

	// Convert to strings
	var cronDays []string
	for _, day := range dayList {
		cronDays = append(cronDays, strconv.Itoa(day))
	}

	return strings.Join(cronDays, ","), nil
}

// convertMonthToCron converts month values to cron format (1-12)
// Supports individual months (1,2,12) and ranges (1-3,5-8)
func convertMonthToCron(months string) (string, error) {
	if months == "" {
		return "", nil // Empty is allowed for filters
	}

	// Split comma-separated months/ranges
	parts := strings.Split(months, ",")
	var cronMonths []string
	monthSet := make(map[int]bool) // To avoid duplicates

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Check if it's a range (contains hyphen)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return "", fmt.Errorf("invalid range format '%s', expected format like '1-3'", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return "", fmt.Errorf("invalid start month '%s' in range '%s'", rangeParts[0], part)
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return "", fmt.Errorf("invalid end month '%s' in range '%s'", rangeParts[1], part)
			}

			// Validate range
			if start < 1 || start > 12 {
				return "", fmt.Errorf("invalid start month %d in range '%s', must be between 1-12", start, part)
			}
			if end < 1 || end > 12 {
				return "", fmt.Errorf("invalid end month %d in range '%s', must be between 1-12", end, part)
			}
			if start > end {
				return "", fmt.Errorf("invalid range '%s', start month must be less than or equal to end month", part)
			}

			// Add all months in range
			for month := start; month <= end; month++ {
				monthSet[month] = true
			}
		} else {
			// Single month
			month, err := strconv.Atoi(part)
			if err != nil {
				return "", fmt.Errorf("invalid month '%s', must be a number between 1-12", part)
			}

			// Validate range (1-12)
			if month < 1 || month > 12 {
				return "", fmt.Errorf("invalid month %d, must be between 1-12", month)
			}

			monthSet[month] = true
		}
	}

	// Convert set to sorted slice
	var monthList []int
	for month := range monthSet {
		monthList = append(monthList, month)
	}

	// Sort the months
	sort.Ints(monthList)

	// Convert to strings
	for _, month := range monthList {
		cronMonths = append(cronMonths, strconv.Itoa(month))
	}

	return strings.Join(cronMonths, ","), nil
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
	cmd.PersistentFlags().StringP("day-of-month", "D", viper.GetString("day-of-month"), "Day of the month for repeated schedule: --day-of-month \"1-4,31\"")
	cmd.PersistentFlags().StringP("months", "x", viper.GetString("months"), "The months in which the schedule should run --months \"1-2,12\"")
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

	if err := checkResponse(resp.HTTPResponse, resp.Body, "error while retrieving schedule"); err != nil {
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

	//Parameters for repeated schedule
	frequency, _ := cmd.Flags().GetString("frequency")
	dayOfWeek, _ := cmd.Flags().GetString("day-of-week")
	dayOfMonth, _ := cmd.Flags().GetString("day-of-month")
	months, _ := cmd.Flags().GetString("months")
	duration, _ := cmd.Flags().GetInt("duration")

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
		if startTime == "" || !validateStartTimeFormat(startTime, REPEATED) {
			return errors.New("repeated schedule --start-time must be specified in format \"HH:MM\"")
		}
		hour, minute := getTimeInCron(startTime, timezone)

		if frequency != "weekly" && frequency != "monthly" {
			return errors.New("invalid --frequency, must be 'weekly' or 'monthly'")
		}

		var cronDayOfMonth string
		var cronDayOfWeek string
		var cronMonth string
		if frequency == "weekly" && dayOfWeek == "" {
			return errors.New("--day-of-week must be specified for weekly frequency")
		} else if frequency == "weekly" {
			// Validate dayOfWeek values
			cronDayOfWeek, err = convertDayOfWeekToCron(dayOfWeek)
			if err != nil {
				return err
			}

			if dayOfMonth != "" {
				fmt.Println("--day-of-month should not be specified for weekly frequency - ignoring")
			}
			cronDayOfMonth = "*"
		}

		if frequency == "monthly" && dayOfMonth == "" {
			return errors.New("--day-of-month must be specified for monthly frequency")
		} else if frequency == "monthly" {
			// Validate dayOfMonth values
			cronDayOfMonth, err = convertDayOfMonthToCron(dayOfMonth)
			if err != nil {
				return err
			}

			if dayOfWeek != "" {
				fmt.Println("--day-of-week should not be specified for monthly frequency - ignoring")
			}
			cronDayOfWeek = "*"
		}

		if months == "" {
			return errors.New("months must be specified in format --months \"1,2,5-8\" ")
		}
		cronMonth, err = convertMonthToCron(months)
		if err != nil {
			return err
		}

		if duration <= 0 {
			return errors.New("duration must be a positive integer representing seconds")
		}

		// Set appropriate values based on frequency
		fmt.Printf("Day of Months is %s\n", cronDayOfMonth)

		fmt.Printf("Day of week is %s\n", cronDayOfWeek)
		resp, err := scheduleClient.ScheduleServiceCreateRepeatedScheduleWithResponse(ctx, projectName,
			infra.ScheduleServiceCreateRepeatedScheduleJSONRequestBody{
				Name:            &name,
				ScheduleStatus:  infra.ScheduleStatus(maintenanceType),
				CronDayWeek:     cronDayOfWeek,
				CronDayMonth:    cronDayOfMonth,
				CronMonth:       cronMonth,
				CronHours:       hour,
				CronMinutes:     minute,
				DurationSeconds: int32(duration),
				TargetHostId:    hostname,
				TargetRegionId:  region,
				TargetSiteId:    site,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating schedule %s", name))
	}

	//Single schedule logic

	if scheduleType == "single" {

		if startTime == "" || !validateStartTimeFormat(startTime, SINGLE) {
			return errors.New("single schedule --start-time must be specified in format \"YYYY-MM-DD HH:MM\"")
		}
		startSeconds := getTimeInSeconds(startTime, timezone)

		var endSeconds *int
		if endTime != "" {
			if !validateStartTimeFormat(endTime, SINGLE) {
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
		return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating schedule %s", name))
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

	if err = checkResponse(gresp.HTTPResponse, gresp.Body, "Error getting schedules"); err != nil {
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
		return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting schedule %s", name))
	}

	if repeatedSchedule.ResourceId != nil {
		resp, err := sshKeyClient.ScheduleServiceDeleteRepeatedScheduleWithResponse(ctx, projectName,
			*repeatedSchedule.ResourceId, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting schedule %s", name))
	}
	return errors.New("no schedule matches the given id")
}
