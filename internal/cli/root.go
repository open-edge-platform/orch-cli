// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/atomix/dazl"
	clilib "github.com/open-edge-platform/orch-library/go/pkg/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
)

var log = dazl.GetLogger()

// Default value for the catalog service REST end-point.
const (
	CLIName = "orch-cli"

	apiEndpoint  = "api-endpoint"
	debugHeaders = "debug-headers"
	project      = "project"

	// Default for dev deployment
	apiDefaultEndpoint = "https://api.kind.internal/"

	// Time format constants
	HumanReadableTimeFormat = "2006-01-02 15:04:05 UTC"
	ISO8601Format           = "2006-01-02T15:04:05Z"
	DateOnlyFormat          = "2006-01-02"
	TimeOnlyFormat          = "15:04:05"
)

// TimestampFields lists all API response fields that contain epoch timestamps
// These fields are found in resources like Timestamps, Schedules, etc.
var TimestampFields = []string{
	// Common timestamp fields from Timestamps schema
	"createdAt",
	"created_at",
	"updatedAt",
	"updated_at",
	"deletedAt",
	"deleted_at",

	// Schedule-related timestamp fields
	"startTime",
	"start_time",
	"endTime",
	"end_time",
	"startSeconds",
	"start_seconds",
	"endSeconds",
	"end_seconds",
	"scheduleTime",
	"schedule_time",
	"nextRunTime",
	"next_run_time",
	"lastRunTime",
	"last_run_time",

	// Host/Instance related
	"onboardedAt",
	"onboarded_at",
	"registeredAt",
	"registered_at",
	"provisionedAt",
	"provisioned_at",
	"lastSeenAt",
	"last_seen_at",
	"lastHeartbeat",
	"last_heartbeat",

	// OS Update related
	"scheduledAt",
	"scheduled_at",
	"completedAt",
	"completed_at",
	"startedAt",
	"started_at",

	// Telemetry related
	"timestamp",
	"collectedAt",
	"collected_at",

	// Lease/expiration
	"expiresAt",
	"expires_at",
	"leaseExpiry",
	"lease_expiry",

	// Vpro power operation timestamps
	"powerStatusTimestamp",
	"power_status_timestamp",
	"powerOnTime",
	"power_on_time",
	"amtStatusTimestamp",
	"amt_status_timestamp",

	// Host status timestamps
	"hostStatusTimestamp",
	"host_status_timestamp",
	"onboardingStatusTimestamp",
	"onboarding_status_timestamp",
	"registrationStatusTimestamp",
	"registration_status_timestamp",
	"instanceStatusTimestamp",
	"instance_status_timestamp",
	"provisioningStatusTimestamp",
	"provisioning_status_timestamp",
	"updateStatusTimestamp",
	"update_status_timestamp",
	"trustedAttestationStatusTimestamp",
	"trusted_attestation_status_timestamp",
	"statusTimestamp",
	"status_timestamp",
}

// TimeDisplayFormat stores the user's preferred time display format
var TimeDisplayFormat = HumanReadableTimeFormat

// init initializes the command line
func init() {
	// Set the config directory relative path
	clilib.SetConfigDir("." + CLIName)

	// Initialize the config name
	clilib.InitConfig(CLIName)

	// Pre-create the config
	_ = clilib.CreateConfig(false)
}

// Init is a hook called after cobra initialization
func Init() {
	// noop for now
}

// Execute is the main entry point for the command-line execution.
func Execute() {
	rootCmd := getRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "orch-cli {create, get, set, list, delete, version} <resource> [flags]",
		Short:         "Orch-cli Command Line Interface",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Set some factory defaults as a fallback
	viper.SetDefault(apiEndpoint, apiDefaultEndpoint)
	viper.SetDefault(debugHeaders, false)
	viper.SetDefault("verbose", false)
	viper.SetDefault(project, "")
	viper.SetDefault("time-format", "human") // human, iso8601, epoch

	// Setup global persistent flags for endpoint addresses of various services
	rootCmd.PersistentFlags().String(apiEndpoint, viper.GetString(apiEndpoint), "API Service Endpoint")
	rootCmd.PersistentFlags().Bool(debugHeaders, viper.GetBool(debugHeaders), "emit debug-style headers separating columns via '|' character")
	rootCmd.PersistentFlags().StringP(project, "p", viper.GetString(project), "Active project name")

	// Setup global persistent flag for verbose output
	var Verbose bool
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", viper.GetBool("verbose"), "produce verbose output")
	var NoAuth bool
	rootCmd.PersistentFlags().BoolVarP(&NoAuth, "noauth", "n", viper.GetBool("noauth"), "use without authentication checks")

	// Time format flag
	rootCmd.PersistentFlags().String("time-format", viper.GetString("time-format"),
		"Time display format: human (2006-01-02 15:04:05 UTC), iso8601, epoch")

	rootCmd.AddCommand(
		clilib.GetConfigCommand(),
		getCreateCommand(),
		getListCommand(),
		getGetCommand(),
		getSetCommand(),
		getDeleteCommand(),
		getUpgradeCommand(),
		getUploadCommand(),
		getLoginCommand(),
		getLogoutCommand(),
		getExportCommand(),
		getDeauthorizeCommand(),
		getUpdateCommand(),
		getWipeProjectCommand(),
		versionCommand(),
		getImportCommand(),
		getGenerateCommand(),
	)
	return rootCmd
}

// GenerateDocs generates markdown documentation for the suite of catalog service CLI commands.
func GenerateDocs() {
	cmd := getRootCmd()
	err := doc.GenMarkdownTree(cmd, "docs/cli")
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

// =====================================================
// Time Conversion Utilities
// =====================================================

// EpochToHumanReadable converts Unix epoch timestamp (seconds or milliseconds) to human-readable UTC format
func EpochToHumanReadable(epoch int64) string {
	if epoch == 0 {
		return "N/A"
	}

	var t time.Time

	// Detect if epoch is in seconds or milliseconds
	// Milliseconds will be > 10^12 (year 2001+), seconds will be < 10^12
	if epoch > 1e12 {
		// Milliseconds
		t = time.UnixMilli(epoch).UTC()
	} else {
		// Seconds
		t = time.Unix(epoch, 0).UTC()
	}

	return t.Format(TimeDisplayFormat)
}

// EpochToISO8601 converts Unix epoch timestamp to ISO8601 format
func EpochToISO8601(epoch int64) string {
	if epoch == 0 {
		return "N/A"
	}

	var t time.Time
	if epoch > 1e12 {
		t = time.UnixMilli(epoch).UTC()
	} else {
		t = time.Unix(epoch, 0).UTC()
	}

	return t.Format(ISO8601Format)
}

// FormatTimestamp converts an epoch timestamp based on the configured display format
func FormatTimestamp(epoch int64) string {
	format := viper.GetString("time-format")

	switch format {
	case "iso8601":
		return EpochToISO8601(epoch)
	case "epoch":
		return fmt.Sprintf("%d", epoch)
	case "human":
		fallthrough
	default:
		return EpochToHumanReadable(epoch)
	}
}

// FormatTimestampString converts a string epoch timestamp to human-readable format
func FormatTimestampString(epochStr string) string {
	if epochStr == "" || epochStr == "0" {
		return "N/A"
	}

	epoch, err := strconv.ParseInt(epochStr, 10, 64)
	if err != nil {
		// Try parsing as float (some APIs return float timestamps)
		epochFloat, err := strconv.ParseFloat(epochStr, 64)
		if err != nil {
			return epochStr // Return as-is if not parseable
		}
		epoch = int64(epochFloat)
	}

	return FormatTimestamp(epoch)
}

// ParseGoogleTimestamp parses google.protobuf.Timestamp format and returns human-readable string
func ParseGoogleTimestamp(timestamp string) string {
	if timestamp == "" {
		return "N/A"
	}

	// Try RFC3339 format (google.protobuf.Timestamp string representation)
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// Try RFC3339Nano
		t, err = time.Parse(time.RFC3339Nano, timestamp)
		if err != nil {
			// Try parsing as epoch
			return FormatTimestampString(timestamp)
		}
	}

	return t.UTC().Format(TimeDisplayFormat)
}

// ConvertTimestampsInMap recursively converts all timestamp fields in a map to human-readable format
func ConvertTimestampsInMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		if isTimestampField(key) {
			result[key] = convertTimestampValue(value)
		} else {
			switch v := value.(type) {
			case map[string]interface{}:
				result[key] = ConvertTimestampsInMap(v)
			case []interface{}:
				result[key] = convertTimestampsInSlice(v)
			default:
				result[key] = value
			}
		}
	}

	return result
}

// isTimestampField checks if a field name is a known timestamp field
func isTimestampField(fieldName string) bool {
	for _, tf := range TimestampFields {
		if fieldName == tf {
			return true
		}
	}
	return false
}

// convertTimestampValue converts a timestamp value to human-readable format
func convertTimestampValue(value interface{}) string {
	switch v := value.(type) {
	case int64:
		return FormatTimestamp(v)
	case int:
		return FormatTimestamp(int64(v))
	case float64:
		return FormatTimestamp(int64(v))
	case string:
		// Could be epoch string or google.protobuf.Timestamp
		if _, err := strconv.ParseInt(v, 10, 64); err == nil {
			return FormatTimestampString(v)
		}
		return ParseGoogleTimestamp(v)
	case nil:
		return "N/A"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// convertTimestampsInSlice converts timestamps in a slice
func convertTimestampsInSlice(data []interface{}) []interface{} {
	result := make([]interface{}, len(data))

	for i, item := range data {
		switch v := item.(type) {
		case map[string]interface{}:
			result[i] = ConvertTimestampsInMap(v)
		default:
			result[i] = item
		}
	}

	return result
}

// =====================================================
// Response Formatting Utilities
// =====================================================

// FormatResourceTimestamps formats the standard timestamps in a resource response
type ResourceTimestamps struct {
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	DeletedAt string `json:"deletedAt,omitempty"`
}

// FormatResourceTimestampsFromEpoch converts epoch timestamps to human-readable format
func FormatResourceTimestampsFromEpoch(createdAt, updatedAt, deletedAt int64) ResourceTimestamps {
	return ResourceTimestamps{
		CreatedAt: FormatTimestamp(createdAt),
		UpdatedAt: FormatTimestamp(updatedAt),
		DeletedAt: FormatTimestamp(deletedAt),
	}
}

// ScheduleTimestamps represents schedule-related timestamps
type ScheduleTimestamps struct {
	StartTime   string `json:"startTime,omitempty"`
	EndTime     string `json:"endTime,omitempty"`
	NextRunTime string `json:"nextRunTime,omitempty"`
	LastRunTime string `json:"lastRunTime,omitempty"`
}

// FormatScheduleTimestamps converts schedule epoch timestamps to human-readable format
func FormatScheduleTimestamps(startTime, endTime, nextRunTime, lastRunTime int64) ScheduleTimestamps {
	return ScheduleTimestamps{
		StartTime:   FormatTimestamp(startTime),
		EndTime:     FormatTimestamp(endTime),
		NextRunTime: FormatTimestamp(nextRunTime),
		LastRunTime: FormatTimestamp(lastRunTime),
	}
}

// =====================================================
// Time Parsing Utilities (for input)
// =====================================================

// ParseHumanTimeToEpoch parses human-readable time string to Unix epoch seconds
func ParseHumanTimeToEpoch(timeStr string) (int64, error) {
	// Try various formats
	formats := []string{
		HumanReadableTimeFormat,
		ISO8601Format,
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"01/02/2006 15:04:05",
		"01/02/2006",
	}

	for _, format := range formats {
		t, err := time.Parse(format, timeStr)
		if err == nil {
			return t.UTC().Unix(), nil
		}
	}

	// Try parsing as epoch directly
	epoch, err := strconv.ParseInt(timeStr, 10, 64)
	if err == nil {
		return epoch, nil
	}

	return 0, fmt.Errorf("unable to parse time string: %s", timeStr)
}

// DurationToSeconds converts a duration string to seconds
func DurationToSeconds(durationStr string) (int64, error) {
	d, err := time.ParseDuration(durationStr)
	if err != nil {
		return 0, err
	}
	return int64(d.Seconds()), nil
}

// SecondsToHumanDuration converts seconds to human-readable duration
func SecondsToHumanDuration(seconds int64) string {
	d := time.Duration(seconds) * time.Second

	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	secs := d / time.Second

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, secs)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

// RelativeTime returns a human-readable relative time string
func RelativeTime(epoch int64) string {
	if epoch == 0 {
		return "N/A"
	}

	var t time.Time
	if epoch > 1e12 {
		t = time.UnixMilli(epoch).UTC()
	} else {
		t = time.Unix(epoch, 0).UTC()
	}

	now := time.Now().UTC()
	diff := now.Sub(t)

	if diff < 0 {
		// Future time
		diff = -diff
		return "in " + formatDuration(diff)
	}

	return formatDuration(diff) + " ago"
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	} else if d < 30*24*time.Hour {
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	} else if d < 365*24*time.Hour {
		return fmt.Sprintf("%d months", int(d.Hours()/(24*30)))
	}
	return fmt.Sprintf("%d years", int(d.Hours()/(24*365)))
}
