// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	promrest "github.com/open-edge-platform/cli/pkg/rest/prometheus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	metricsEndpointFlag = "metrics-endpoint"
	hostnameFlag        = "hostname"
	hostnameLabelFlag   = "hostname-label"
	orgIDFlag           = "org-id"
	averageFlag         = "average"
	sumFlag             = "sum"
	increaseFlag        = "increase"
	rangeFlag           = "range"
	durationFlag        = "duration"
	startTimeFlag       = "start-time"
	endTimeFlag         = "end-time"
	timestampFlag       = "timestamp"

	defaultHostnameLabel  = "host"
	defaultMetricsTimeout = 30 * time.Second

	DEFAULT_LIST_METRICS_FORMAT    = "table{{.Number}}\t{{str .Metric}}"
	METRICS_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_METRICS_OUTPUT_TEMPLATE"

	DEFAULT_GET_METRIC_FORMAT         = "table{{str .Metric}}\t{{str .Host}}\t{{str .Labels}}\t{{str .Value}}\t{{str .Timestamp}}"
	DEFAULT_GET_METRIC_RANGE_FORMAT   = "table{{.Row}}\t{{str .Metric}}\t{{str .Host}}\t{{str .Labels}}\t{{str .Value}}\t{{str .Timestamp}}"
	DEFAULT_GET_METRIC_INSPECT_FORMAT = `Metric: {{str .Metric}}
Host: {{str .Host}}
Host GUID: {{str .HostGUID}}
Project ID: {{str .ProjectID}}
Labels: {{str .Labels}}
Timestamp: {{str .Timestamp}}
Value: {{str .Value}}
`
	METRIC_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_METRIC_OUTPUT_TEMPLATE"
)

const configuredMetricsEndpointExample = `# Configure metrics endpoint (once)
orch-cli config set metrics-endpoint https://metrics-node-cli.<CLUSTER_FQDN>/prometheus
`

const listMetricNamesExamples = `# List metrics for the current project (org-id auto-derived from project UID) and current metrics endpoint
orch-cli list metrics
# List metrics for a different project and metrics endpoint
orch-cli list metrics --metrics-endpoint https://mimir.example.com/prometheus --project myproject
# List metrics with explicit org-id
orch-cli list metrics --metrics-endpoint https://mimir.example.com/prometheus --org-id 698fde6a-b721-447a-a7c2-7187d64393c1
# Filter metric names
orch-cli list metrics --filter node_cpu
`

const getMetricExamples = `# Query metric for a host in the current project (org-id auto-derived)
orch-cli get metric metric_example --hostname host-fd7108f7
# Query metric for a host in another project (org-id auto-derived)
orch-cli get metric metric_example --hostname host-fd7108f7 --project myproject
# Query with explicit org-id
orch-cli get metric metric_example --hostname host-fd7108f7 --org-id 698fde6a-b721-447a-a7c2-7187d64393c1
# Query using a custom hostname label
orch-cli get metric up --hostname edge-node-01 --hostname-label instance --project myproject
# Query average metric over a time range (Unix timestamps)
orch-cli get metric metric_example --hostname host-fd7108f7 --average --start-time 1704067200 --end-time 1704153600
# Query average metric over the last hour ending now
orch-cli get metric metric_example --hostname host-fd7108f7 --average --duration 3600
# Query sum of metric over a specific time range
orch-cli get metric metric_example --hostname host-fd7108f7 --sum --start-time 1704067200 --end-time 1704153600
# Query sum of metric over the last hour ending now
orch-cli get metric metric_example --hostname host-fd7108f7 --sum --duration 3600
# Query increase of metric over a specific time range (best for counters)
orch-cli get metric metric_example --hostname host-fd7108f7 --increase --start-time 1704067200 --end-time 1704153600
# Query increase of metric over the last hour ending now
orch-cli get metric metric_example --hostname host-fd7108f7 --increase --duration 3600
# Query metric range over the last hour ending now
orch-cli get metric metric_example --hostname host-fd7108f7 --range --duration 3600
# Query metric range between two timestamps
orch-cli get metric metric_example --hostname host-fd7108f7 --range --start-time 1704067200 --end-time 1704153600
# Query metric at a specific timestamp
orch-cli get metric metric_example --hostname host-fd7108f7 --timestamp 1704153600
`

var metricNamePattern = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)
var labelNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type prometheusVectorResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
			Values [][]interface{}   `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

type metricGetRow struct {
	Metric    *string `json:"metric"`
	Host      *string `json:"host"`
	HostGUID  *string `json:"hostGuid"`
	ProjectID *string `json:"projectId"`
	Labels    *string `json:"labels"`
	Timestamp *string `json:"timestamp"`
	Value     *string `json:"value"`
}

type metricListRow struct {
	Number int     `json:"number"`
	Metric *string `json:"metric"`
}

type metricRangeRow struct {
	Row       *int    `json:"row"`
	Metric    *string `json:"metric"`
	Host      *string `json:"host"`
	HostGUID  *string `json:"hostGuid"`
	ProjectID *string `json:"projectId"`
	Labels    *string `json:"labels"`
	Timestamp *string `json:"timestamp"`
	Value     *string `json:"value"`
}

// getListMetricNamesCommand builds the `list metrics` command.
func getListMetricNamesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "metrics",
		Short:   "List all metric names available at a Mimir (Prometheus-compatible) endpoint",
		Args:    cobra.NoArgs,
		Example: configuredMetricsEndpointExample + listMetricNamesExamples,
		RunE:    runListMetricNamesCommand,
	}

	cmd.Flags().String(metricsEndpointFlag, configuredMetricsEndpoint(), "Mimir (Prometheus-compatible) base URL")
	cmd.Flags().String(orgIDFlag, viper.GetString(orgIDFlag), "Mimir tenant ID sent as X-Scope-OrgID")
	cmd.Flags().String("filter", "", "Only show metric names containing this substring")
	addStandardListOutputFlags(cmd)
	return cmd
}

// getListMetricsOutputFormat resolves the output template for metric lists.
func getListMetricsOutputFormat(cmd *cobra.Command) (string, error) {
	return resolveTableOutputTemplate(cmd, DEFAULT_LIST_METRICS_FORMAT, METRICS_OUTPUT_TEMPLATE_ENVVAR)
}

// printMetricNames renders a list of metric names in the selected output format.
func printMetricNames(cmd *cobra.Command, writer *tabwriter.Writer, metricNames []string) error {
	rows := make([]metricListRow, 0, len(metricNames))
	for i, metricName := range metricNames {
		name := metricName
		rows = append(rows, metricListRow{Number: i + 1, Metric: &name})
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getListMetricsOutputFormat(cmd)
	if err != nil {
		return err
	}

	filterSpec := ""
	if outputType == "table" {
		outputFilter, _ := cmd.Flags().GetString("output-filter")
		filterSpec = outputFilter
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    filterSpec,
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      rows,
	}

	GenerateOutput(writer, &result)
	return nil
}

// runListMetricNamesCommand fetches and prints metric names from Prometheus.
func runListMetricNamesCommand(cmd *cobra.Command, _ []string) error {
	writer, _ := getOutputContext(cmd)
	filter, err := cmd.Flags().GetString("filter")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultMetricsTimeout)
	defer cancel()

	client, err := PrometheusClientFactory(cmd)
	if err != nil {
		return err
	}
	argID, err := resolveOrgID(cmd)
	if err != nil {
		return err
	}

	body, err := promrest.ExecuteGET(ctx, client, promrest.ListMetricsAPIPath, argID)
	if err != nil {
		return err
	}

	var result struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse Prometheus response: %w", err)
	}
	if result.Status != "success" {
		return fmt.Errorf("prometheus returned non-success status: %s", result.Status)
	}

	filteredNames := make([]string, 0, len(result.Data))
	for _, name := range result.Data {
		if filter == "" || strings.Contains(name, filter) {
			filteredNames = append(filteredNames, name)
		}
	}

	if err := printMetricNames(cmd, writer, filteredNames); err != nil {
		return err
	}
	return writer.Flush()
}

// getGetMetricCommand builds the `get metric` command.
func getGetMetricCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "metric <metric-name>",
		Short:   "Query a metric from a Mimir (Prometheus-compatible) endpoint for a specific hostname",
		Args:    cobra.ExactArgs(1),
		Example: configuredMetricsEndpointExample + getMetricExamples,
		RunE:    runGetMetricCommand,
	}

	cmd.Aliases = []string{"metrics"}
	cmd.Flags().String(hostnameFlag, viper.GetString(hostnameFlag), "Hostname label value to match in Prometheus")
	cmd.Flags().String(hostnameLabelFlag, defaultHostnameLabel, "Prometheus label name used to match the hostname")
	cmd.Flags().String(metricsEndpointFlag, configuredMetricsEndpoint(), "Mimir (Prometheus-compatible) base URL")
	cmd.Flags().String(orgIDFlag, viper.GetString(orgIDFlag), "Mimir tenant ID sent as X-Scope-OrgID")
	cmd.Flags().Bool(averageFlag, false, "Calculate average of metric over time range (use either --duration or --start-time with --end-time)")
	cmd.Flags().Bool(sumFlag, false, "Calculate sum of metric over time range (use either --duration or --start-time with --end-time)")
	cmd.Flags().Bool(increaseFlag, false, "Calculate increase of metric over time range (recommended for counters; use either --duration or --start-time with --end-time)")
	cmd.Flags().Bool(rangeFlag, false, "Retrieve metric range values over time (use either --duration or --start-time with --end-time)")
	cmd.Flags().Int64(durationFlag, 0, "Duration in seconds for --sum/--average/--increase/--range calculation ending now (e.g. 3600 for last hour)")
	cmd.Flags().String(startTimeFlag, "", "Start time for range query (Unix timestamp, e.g. 1704067200)")
	cmd.Flags().String(endTimeFlag, "", "End time for range query (Unix timestamp, e.g. 1704153600)")
	cmd.Flags().String(timestampFlag, "", "Evaluate metric at a specific Unix timestamp (instant query mode)")
	addStandardGetOutputFlags(cmd)
	_ = cmd.MarkFlagRequired(hostnameFlag)

	return cmd
}

// getMetricOutputFormat resolves the standard output template for metric queries.
func getMetricOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_GET_METRIC_INSPECT_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_GET_METRIC_FORMAT, METRIC_OUTPUT_TEMPLATE_ENVVAR)
}

// getMetricRangeOutputFormat resolves the output template for range queries.
func getMetricRangeOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_GET_METRIC_INSPECT_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_GET_METRIC_RANGE_FORMAT, METRIC_OUTPUT_TEMPLATE_ENVVAR)
}

// printMetricResult formats non-range get metric responses and keeps the last sample for matrix results.
func printMetricResult(cmd *cobra.Command, writer *tabwriter.Writer, metricName string, hostnameLabel string, body []byte, verbose bool) error {
	var resp prometheusVectorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse Prometheus response: %w", err)
	}
	if resp.Status != "success" {
		return fmt.Errorf("prometheus returned non-success status: %s", resp.Status)
	}
	if resp.Data.ResultType != "vector" && resp.Data.ResultType != "matrix" {
		return fmt.Errorf("unsupported prometheus result type %q, expected vector or matrix", resp.Data.ResultType)
	}

	rows := make([]metricGetRow, 0, len(resp.Data.Result))
	for _, item := range resp.Data.Result {
		host := item.Metric[hostnameLabel]
		if host == "" {
			host = item.Metric[defaultHostnameLabel]
		}

		hostGUID := item.Metric["hostGuid"]
		projectID := item.Metric["projectId"]

		// Extract additional labels (like cpu, disk, etc.) excluding standard ones
		additionalLabels := make([]string, 0)
		standardLabels := map[string]bool{
			hostnameLabel:        true,
			defaultHostnameLabel: true,
			"hostGuid":           true,
			"projectId":          true,
			"__name__":           true,
		}
		for k, v := range item.Metric {
			if !standardLabels[k] && v != "" {
				additionalLabels = append(additionalLabels, fmt.Sprintf("%s=%s", k, v))
			}
		}
		labelsStr := strings.Join(additionalLabels, ", ")

		timestamp, value := "", ""
		if resp.Data.ResultType == "matrix" {
			timestamp, value = formatPrometheusSample(lastPrometheusSample(item.Values))
		} else {
			timestamp, value = formatPrometheusSample(item.Value)
		}

		rows = append(rows, metricGetRow{
			Metric:    &metricName,
			Host:      &host,
			HostGUID:  &hostGUID,
			ProjectID: &projectID,
			Labels:    &labelsStr,
			Timestamp: &timestamp,
			Value:     &value,
		})
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getMetricOutputFormat(cmd, verbose)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		_, err = fmt.Fprintln(writer, "No metrics found")
		return err
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      rows,
	}

	GenerateOutput(writer, &result)
	return nil
}

// printMetricRangeResult formats range-query responses for `get metric --range`.
// For matrix results, each sample in `values` becomes a separate output row.
func printMetricRangeResult(cmd *cobra.Command, writer *tabwriter.Writer, metricName string, hostnameLabel string, body []byte, verbose bool) error {
	var resp prometheusVectorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse Prometheus response: %w", err)
	}
	if resp.Status != "success" {
		return fmt.Errorf("prometheus returned non-success status: %s", resp.Status)
	}
	if resp.Data.ResultType != "vector" && resp.Data.ResultType != "matrix" {
		return fmt.Errorf("unsupported prometheus result type %q, expected vector or matrix", resp.Data.ResultType)
	}

	rows := make([]metricRangeRow, 0)
	rowNumber := 0
	for _, item := range resp.Data.Result {
		host := item.Metric[hostnameLabel]
		if host == "" {
			host = item.Metric[defaultHostnameLabel]
		}

		hostGUID := item.Metric["hostGuid"]
		projectID := item.Metric["projectId"]

		additionalLabels := make([]string, 0)
		standardLabels := map[string]bool{
			hostnameLabel:        true,
			defaultHostnameLabel: true,
			"hostGuid":           true,
			"projectId":          true,
			"__name__":           true,
		}
		for k, v := range item.Metric {
			if !standardLabels[k] && v != "" {
				additionalLabels = append(additionalLabels, fmt.Sprintf("%s=%s", k, v))
			}
		}
		labelsStr := strings.Join(additionalLabels, ", ")

		appendSample := func(sample []interface{}) {
			timestamp, value := formatPrometheusSample(sample)
			rowNumber++
			row := rowNumber
			rows = append(rows, metricRangeRow{
				Row:       &row,
				Metric:    &metricName,
				Host:      &host,
				HostGUID:  &hostGUID,
				ProjectID: &projectID,
				Labels:    &labelsStr,
				Timestamp: &timestamp,
				Value:     &value,
			})
		}

		if resp.Data.ResultType == "matrix" {
			for _, sample := range item.Values {
				appendSample(sample)
			}
		} else {
			appendSample(item.Value)
		}
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getMetricRangeOutputFormat(cmd, verbose)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		_, err = fmt.Fprintln(writer, "No metrics found")
		return err
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      rows,
	}

	GenerateOutput(writer, &result)
	return nil
}

// formatPrometheusSample converts a Prometheus sample into timestamp and value strings.
func formatPrometheusSample(sample []interface{}) (string, string) {
	timestamp := ""
	value := ""
	if len(sample) >= 2 {
		if ts, ok := sample[0].(float64); ok {
			timestamp = strconv.FormatInt(int64(ts), 10)
		} else {
			timestamp = fmt.Sprintf("%v", sample[0])
		}
		value = fmt.Sprintf("%v", sample[1])
	}
	return timestamp, value
}

// lastPrometheusSample returns the most recent sample from a Prometheus matrix result.
func lastPrometheusSample(samples [][]interface{}) []interface{} {
	if len(samples) == 0 {
		return nil
	}
	return samples[len(samples)-1]
}

// runGetMetricCommand executes instant, average, sum, increase, or range metric queries.
func runGetMetricCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	metricName := args[0]
	hostname, err := cmd.Flags().GetString(hostnameFlag)
	if err != nil {
		return err
	}
	hostnameLabel, err := cmd.Flags().GetString(hostnameLabelFlag)
	if err != nil {
		return err
	}

	// Check if this is a range query
	averageQuery, err := cmd.Flags().GetBool(averageFlag)
	if err != nil {
		return err
	}
	sumQuery, err := cmd.Flags().GetBool(sumFlag)
	if err != nil {
		return err
	}
	increaseQuery, err := cmd.Flags().GetBool(increaseFlag)
	if err != nil {
		return err
	}
	rangeQuery, err := cmd.Flags().GetBool(rangeFlag)
	if err != nil {
		return err
	}
	durationSec, err := cmd.Flags().GetInt64(durationFlag)
	if err != nil {
		return err
	}
	startTimeStr, err := cmd.Flags().GetString(startTimeFlag)
	if err != nil {
		return err
	}
	endTimeStr, err := cmd.Flags().GetString(endTimeFlag)
	if err != nil {
		return err
	}
	timestampStr, err := cmd.Flags().GetString(timestampFlag)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultMetricsTimeout)
	defer cancel()

	client, err := PrometheusClientFactory(cmd)
	if err != nil {
		return err
	}
	argID, err := resolveOrgID(cmd)
	if err != nil {
		return err
	}

	// Check which aggregation mode is requested and validate flag combinations
	aggregationModeCount := 0
	if averageQuery {
		aggregationModeCount++
	}
	if sumQuery {
		aggregationModeCount++
	}
	if increaseQuery {
		aggregationModeCount++
	}
	if aggregationModeCount > 1 {
		return fmt.Errorf("--average, --sum, and --increase are mutually exclusive")
	}
	if rangeQuery && (averageQuery || sumQuery || increaseQuery) {
		return fmt.Errorf("--range cannot be used with --sum, --average, or --increase")
	}

	if strings.TrimSpace(timestampStr) != "" {
		if averageQuery || sumQuery || increaseQuery || rangeQuery {
			return fmt.Errorf("--timestamp cannot be used with --sum, --average, --increase, or --range")
		}
		if durationSec > 0 || strings.TrimSpace(startTimeStr) != "" || strings.TrimSpace(endTimeStr) != "" {
			return fmt.Errorf("--timestamp cannot be used with --duration, --start-time, or --end-time")
		}
	}

	if durationSec > 0 && !averageQuery && !sumQuery && !increaseQuery && !rangeQuery {
		return fmt.Errorf("--duration requires either --sum, --average, --increase, or --range")
	}

	if rangeQuery {
		startTime, endTime, err := resolveRangeWindow(startTimeStr, endTimeStr, durationSec, time.Now())
		if err != nil {
			return err
		}

		query, err := buildMetricQuery(metricName, hostnameLabel, hostname)
		if err != nil {
			return err
		}

		body, err := promrest.ExecuteRangeQuery(ctx, client, query, startTime, endTime, argID, defaultMetricsTimeout)
		if err != nil {
			return err
		}

		if err := printMetricRangeResult(cmd, writer, metricName, hostnameLabel, body, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	if sumQuery {
		startTime, endTime, err := resolveSumWindow(startTimeStr, endTimeStr, durationSec, time.Now())
		if err != nil {
			return err
		}

		durationSec = endTime - startTime
		query, err := buildSumMetricQuery(metricName, hostnameLabel, hostname, durationSec)
		if err != nil {
			return err
		}

		body, err := promrest.ExecuteQueryAt(ctx, client, query, endTime, argID, defaultMetricsTimeout)
		if err != nil {
			return err
		}

		if err := printMetricResult(cmd, writer, metricName, hostnameLabel, body, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	if increaseQuery {
		startTime, endTime, err := resolveIncreaseWindow(startTimeStr, endTimeStr, durationSec, time.Now())
		if err != nil {
			return err
		}

		durationSec = endTime - startTime
		query, err := buildIncreaseMetricQuery(metricName, hostnameLabel, hostname, durationSec)
		if err != nil {
			return err
		}

		body, err := promrest.ExecuteQueryAt(ctx, client, query, endTime, argID, defaultMetricsTimeout)
		if err != nil {
			return err
		}

		if err := printMetricResult(cmd, writer, metricName, hostnameLabel, body, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	// If average flag is set, support either duration or explicit start/end window.
	if averageQuery {
		startTime, endTime, err := resolveAverageWindow(startTimeStr, endTimeStr, durationSec, time.Now())
		if err != nil {
			return err
		}

		durationSec := endTime - startTime
		query, err := buildAverageMetricQuery(metricName, hostnameLabel, hostname, durationSec)
		if err != nil {
			return err
		}

		body, err := promrest.ExecuteQueryAt(ctx, client, query, endTime, argID, defaultMetricsTimeout)
		if err != nil {
			return err
		}

		if err := printMetricResult(cmd, writer, metricName, hostnameLabel, body, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	// Instant query mode.
	query, err := buildMetricQuery(metricName, hostnameLabel, hostname)
	if err != nil {
		return err
	}

	evalTime := int64(0)
	if strings.TrimSpace(timestampStr) != "" {
		evalTime, err = parseTimestamp(timestampStr)
		if err != nil {
			return fmt.Errorf("failed to parse --timestamp: %w", err)
		}
	}

	body, err := promrest.ExecuteQueryAt(ctx, client, query, evalTime, argID, defaultMetricsTimeout)
	if err != nil {
		return err
	}

	if err := printMetricResult(cmd, writer, metricName, hostnameLabel, body, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

// buildMetricQuery builds a selector for the requested metric and hostname.
func buildMetricQuery(metricName string, hostnameLabel string, hostname string) (string, error) {
	if !metricNamePattern.MatchString(metricName) {
		return "", fmt.Errorf("invalid metric name %q", metricName)
	}
	if !labelNamePattern.MatchString(hostnameLabel) {
		return "", fmt.Errorf("invalid hostname label %q", hostnameLabel)
	}
	if strings.TrimSpace(hostname) == "" {
		return "", fmt.Errorf("hostname cannot be empty")
	}

	return fmt.Sprintf(`%s{%s=%q}`, metricName, hostnameLabel, hostname), nil
}

// parseTimestamp parses a timestamp as Unix seconds (int64).
func parseTimestamp(ts string) (int64, error) {
	ts = strings.TrimSpace(ts)

	unixSec, err := strconv.ParseInt(ts, 10, 64)
	if err == nil {
		return unixSec, nil
	}
	return 0, fmt.Errorf("timestamp must be Unix seconds (e.g. 1704067200)")
}

// resolveSumWindow converts sum inputs into a concrete time window.
func resolveSumWindow(startTimeStr string, endTimeStr string, durationSec int64, now time.Time) (int64, int64, error) {
	hasStart := strings.TrimSpace(startTimeStr) != ""
	hasEnd := strings.TrimSpace(endTimeStr) != ""
	hasDuration := durationSec > 0

	if hasDuration && (hasStart || hasEnd) {
		return 0, 0, fmt.Errorf("--sum supports either --duration or --start-time with --end-time, not both")
	}

	if hasDuration {
		endTime := now.Unix()
		startTime := endTime - durationSec
		if startTime >= endTime {
			return 0, 0, fmt.Errorf("--duration must be greater than 0 seconds")
		}
		return startTime, endTime, nil
	}

	if hasStart || hasEnd {
		if !hasStart || !hasEnd {
			return 0, 0, fmt.Errorf("--sum requires both --start-time and --end-time to be set")
		}

		startTime, err := parseTimestamp(startTimeStr)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse --start-time: %w", err)
		}
		endTime, err := parseTimestamp(endTimeStr)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse --end-time: %w", err)
		}

		if startTime >= endTime {
			return 0, 0, fmt.Errorf("--start-time must be before --end-time")
		}

		return startTime, endTime, nil
	}

	return 0, 0, fmt.Errorf("--sum requires either --duration or both --start-time and --end-time")
}

// resolveAverageWindow converts average inputs into a concrete time window.
func resolveAverageWindow(startTimeStr string, endTimeStr string, durationSec int64, now time.Time) (int64, int64, error) {
	hasStart := strings.TrimSpace(startTimeStr) != ""
	hasEnd := strings.TrimSpace(endTimeStr) != ""
	hasDuration := durationSec > 0

	if hasDuration && (hasStart || hasEnd) {
		return 0, 0, fmt.Errorf("--average supports either --duration or --start-time with --end-time, not both")
	}

	if hasDuration {
		endTime := now.Unix()
		startTime := endTime - durationSec
		if startTime >= endTime {
			return 0, 0, fmt.Errorf("--duration must be greater than 0 seconds")
		}
		return startTime, endTime, nil
	}

	if hasStart || hasEnd {
		if !hasStart || !hasEnd {
			return 0, 0, fmt.Errorf("--average requires both --start-time and --end-time to be set")
		}

		startTime, err := parseTimestamp(startTimeStr)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse --start-time: %w", err)
		}
		endTime, err := parseTimestamp(endTimeStr)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse --end-time: %w", err)
		}

		if startTime >= endTime {
			return 0, 0, fmt.Errorf("--start-time must be before --end-time")
		}

		return startTime, endTime, nil
	}

	return 0, 0, fmt.Errorf("--average requires either --duration or both --start-time and --end-time")
}

// resolveRangeWindow converts range inputs into a concrete time window.
func resolveRangeWindow(startTimeStr string, endTimeStr string, durationSec int64, now time.Time) (int64, int64, error) {
	hasStart := strings.TrimSpace(startTimeStr) != ""
	hasEnd := strings.TrimSpace(endTimeStr) != ""
	hasDuration := durationSec > 0

	if hasDuration && (hasStart || hasEnd) {
		return 0, 0, fmt.Errorf("--range supports either --duration or --start-time with --end-time, not both")
	}

	if hasDuration {
		endTime := now.Unix()
		startTime := endTime - durationSec
		if startTime >= endTime {
			return 0, 0, fmt.Errorf("--duration must be greater than 0 seconds")
		}
		return startTime, endTime, nil
	}

	if hasStart || hasEnd {
		if !hasStart || !hasEnd {
			return 0, 0, fmt.Errorf("--range requires both --start-time and --end-time to be set")
		}

		startTime, err := parseTimestamp(startTimeStr)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse --start-time: %w", err)
		}
		endTime, err := parseTimestamp(endTimeStr)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse --end-time: %w", err)
		}

		if startTime >= endTime {
			return 0, 0, fmt.Errorf("--start-time must be before --end-time")
		}

		return startTime, endTime, nil
	}

	return 0, 0, fmt.Errorf("--range requires either --duration or both --start-time and --end-time")
}

// buildAverageMetricQuery builds an avg_over_time query for the requested window.
func buildAverageMetricQuery(metricName string, hostnameLabel string, hostname string, durationSec int64) (string, error) {
	if !metricNamePattern.MatchString(metricName) {
		return "", fmt.Errorf("invalid metric name %q", metricName)
	}
	if !labelNamePattern.MatchString(hostnameLabel) {
		return "", fmt.Errorf("invalid hostname label %q", hostnameLabel)
	}
	if strings.TrimSpace(hostname) == "" {
		return "", fmt.Errorf("hostname cannot be empty")
	}
	if durationSec <= 0 {
		return "", fmt.Errorf("duration must be greater than 0 seconds")
	}
	return fmt.Sprintf(`avg_over_time(%s{%s=%q}[%ds])`, metricName, hostnameLabel, hostname, durationSec), nil
}

// buildSumMetricQuery builds a sum_over_time query for the requested window.
func buildSumMetricQuery(metricName string, hostnameLabel string, hostname string, durationSec int64) (string, error) {
	if !metricNamePattern.MatchString(metricName) {
		return "", fmt.Errorf("invalid metric name %q", metricName)
	}
	if !labelNamePattern.MatchString(hostnameLabel) {
		return "", fmt.Errorf("invalid hostname label %q", hostnameLabel)
	}
	if strings.TrimSpace(hostname) == "" {
		return "", fmt.Errorf("hostname cannot be empty")
	}
	if durationSec <= 0 {
		return "", fmt.Errorf("duration must be greater than 0 seconds")
	}

	return fmt.Sprintf(`sum_over_time(%s{%s=%q}[%ds])`, metricName, hostnameLabel, hostname, durationSec), nil
}

// buildIncreaseMetricQuery builds an increase query for counter-like metrics.
func buildIncreaseMetricQuery(metricName string, hostnameLabel string, hostname string, durationSec int64) (string, error) {
	if !metricNamePattern.MatchString(metricName) {
		return "", fmt.Errorf("invalid metric name %q", metricName)
	}
	if !labelNamePattern.MatchString(hostnameLabel) {
		return "", fmt.Errorf("invalid hostname label %q", hostnameLabel)
	}
	if strings.TrimSpace(hostname) == "" {
		return "", fmt.Errorf("hostname cannot be empty")
	}
	if durationSec <= 0 {
		return "", fmt.Errorf("duration must be greater than 0 seconds")
	}

	return fmt.Sprintf(`increase(%s{%s=%q}[%ds])`, metricName, hostnameLabel, hostname, durationSec), nil
}

// resolveIncreaseWindow converts increase inputs into a concrete time window.
func resolveIncreaseWindow(startTimeStr string, endTimeStr string, durationSec int64, now time.Time) (int64, int64, error) {
	hasStart := strings.TrimSpace(startTimeStr) != ""
	hasEnd := strings.TrimSpace(endTimeStr) != ""
	hasDuration := durationSec > 0

	if hasDuration && (hasStart || hasEnd) {
		return 0, 0, fmt.Errorf("--increase supports either --duration or --start-time with --end-time, not both")
	}

	if hasDuration {
		endTime := now.Unix()
		startTime := endTime - durationSec
		if startTime >= endTime {
			return 0, 0, fmt.Errorf("--duration must be greater than 0 seconds")
		}
		return startTime, endTime, nil
	}

	if hasStart || hasEnd {
		if !hasStart || !hasEnd {
			return 0, 0, fmt.Errorf("--increase requires both --start-time and --end-time to be set")
		}

		startTime, err := parseTimestamp(startTimeStr)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse --start-time: %w", err)
		}
		endTime, err := parseTimestamp(endTimeStr)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse --end-time: %w", err)
		}

		if startTime >= endTime {
			return 0, 0, fmt.Errorf("--start-time must be before --end-time")
		}

		return startTime, endTime, nil
	}

	return 0, 0, fmt.Errorf("--increase requires either --duration or both --start-time and --end-time")
}

// configuredMetricsEndpoint reads the configured Prometheus endpoint.
func configuredMetricsEndpoint() string {
	return strings.TrimSpace(viper.GetString(metricsEndpointFlag))
}

// getMetricsEndpoint resolves the endpoint from flags, config, or api-endpoint.
func getMetricsEndpoint(cmd *cobra.Command) (string, error) {
	endpoint, err := cmd.Flags().GetString(metricsEndpointFlag)
	if err != nil {
		return "", err
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint != "" {
		return endpoint, nil
	}

	endpoint = configuredMetricsEndpoint()
	if endpoint != "" {
		return endpoint, nil
	}

	apiEp, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return "", err
	}
	apiEp = strings.TrimSpace(apiEp)
	if apiEp == "" {
		apiEp = strings.TrimSpace(viper.GetString(apiEndpoint))
	}
	if apiEp == "" {
		return "", nil
	}

	derivedMetricsEndpoint, err := deriveMetricsEndpointFromAPIEndpoint(apiEp)
	if err != nil {
		return "", fmt.Errorf(
			"failed to determine metrics endpoint from api endpoint %q: %w. Set --%s or run 'orch-cli config set %s <url>'",
			apiEp,
			err,
			metricsEndpointFlag,
			metricsEndpointFlag,
		)
	}

	fmt.Printf("Determined metrics endpoint from api endpoint: %s\n", derivedMetricsEndpoint)
	return derivedMetricsEndpoint, nil
}

// deriveMetricsEndpointFromAPIEndpoint maps the API endpoint to the metrics endpoint.
func deriveMetricsEndpointFromAPIEndpoint(apiEp string) (string, error) {
	u, err := url.Parse(apiEp)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid api endpoint %q", apiEp)
	}

	hostname := u.Hostname()
	parts := strings.SplitN(hostname, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("failed to determine metrics endpoint from api endpoint %q", apiEp)
	}

	return fmt.Sprintf("%s://metrics-node_cli.%s/prometheus", u.Scheme, parts[1]), nil
}

// resolveOrgID returns the tenant/project UID for Mimir queries
func resolveOrgID(cmd *cobra.Command) (string, error) {
	// First check if org-id is explicitly set
	argID, err := cmd.Flags().GetString(orgIDFlag)
	if err != nil {
		return "", err
	}
	argID = strings.TrimSpace(argID)
	if argID != "" {
		return argID, nil
	}

	// If not set, try to derive from project UID
	projectName, err := cmd.Flags().GetString("project")
	if err != nil {
		return "", err
	}
	projectName = strings.TrimSpace(projectName)

	if projectName != "" {
		projectUID, err := getProjectUID(cmd, projectName)
		if err != nil {
			// If project UID lookup fails, continue without org-id
			return "", nil
		}
		return projectUID, nil
	}

	return "", nil
}

// getProjectUID looks up the project UID for the provided project name.
func getProjectUID(cmd *cobra.Command, projectName string) (string, error) {
	ctx, projectClient, err := TenancyFactory(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to create project client: %w", err)
	}

	resp, err := projectClient.GETV1ProjectsProjectProjectWithResponse(ctx, projectName, auth.AddAuthHeader)
	if err != nil {
		return "", fmt.Errorf("failed to fetch project %s: %w", projectName, err)
	}

	if resp == nil || resp.JSON200 == nil {
		return "", fmt.Errorf("project %s not found", projectName)
	}

	if resp.JSON200.Status == nil || resp.JSON200.Status.ProjectStatus == nil || resp.JSON200.Status.ProjectStatus.UID == nil {
		return "", fmt.Errorf("project %s has no UID", projectName)
	}

	return *resp.JSON200.Status.ProjectStatus.UID, nil
}
