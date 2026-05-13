// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	promapi "github.com/prometheus/client_golang/api"
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
	rangeFlag           = "range"
	durationFlag        = "duration"
	startTimeFlag       = "start-time"
	endTimeFlag         = "end-time"
	timestampFlag       = "timestamp"

	defaultHostnameLabel         = "host"
	defaultMetricsTimeout        = 30 * time.Second
	prometheusQueryAPIPath       = "/api/v1/query"
	prometheusQueryRangeAPIPath  = "/api/v1/query_range"
	prometheusLabelValuesAPIPath = "/api/v1/label/__name__/values"

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

const listMetricNamesExamples = `# List metrics for the current project (org-id auto-derived from project UID) and current metrics endpoint
orch-cli list metrics
# List metrics for a different project and metrics endpoint
orch-cli list metrics --metrics-endpoint https://mimir.example.com/prometheus --project myproject
# List metrics with explicit org-id
orch-cli list metrics --metrics-endpoint https://mimir.example.com/prometheus --org-id 698fde6a-b721-447a-a7c2-7187d64393c1
# Filter metric names
orch-cli list metrics --filter node_cpu
`

const getMetricExamples = `# Configure metrics endpoint (once)
orch-cli config set metrics-endpoint http://<mimir-endpoint>/prometheus
# Query metric for a host in the current project (org-id auto-derived)
orch-cli get metric mem_used_percent --hostname host-fd7108f7
# Query metric for a host in another project (org-id auto-derived)
orch-cli get metric mem_used_percent --hostname host-fd7108f7 --project myproject
# Query with explicit org-id
orch-cli get metric mem_used_percent --hostname host-fd7108f7 --org-id 698fde6a-b721-447a-a7c2-7187d64393c1
# Query using a custom hostname label
orch-cli get metric up --hostname edge-node-01 --hostname-label instance --project myproject
# Query average metric over a time range (Unix timestamps)
orch-cli get metric mem_used_percent --hostname host-fd7108f7 --average --start-time 1704067200 --end-time 1704153600
# Query average metric over the last hour ending now
orch-cli get metric mem_used_percent --hostname host-fd7108f7 --average --duration 3600
# Query sum of metric over a specific time range
orch-cli get metric mem_used_percent --hostname host-fd7108f7 --sum --start-time 1704067200 --end-time 1704153600
# Query sum of metric over the last hour ending now
orch-cli get metric mem_used_percent --hostname host-fd7108f7 --sum --duration 3600
# Query metric range over the last hour ending now
orch-cli get metric mem_used_percent --hostname host-fd7108f7 --range --duration 3600
# Query metric range between two timestamps
orch-cli get metric mem_used_percent --hostname host-fd7108f7 --range --start-time 1704067200 --end-time 1704153600
# Query metric at a specific timestamp
orch-cli get metric mem_used_percent --hostname host-fd7108f7 --timestamp 1704153600
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

var PrometheusClientFactory = newPrometheusClient

func getListMetricNamesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "metrics",
		Short:   "List all metric names available at a Mimir (Prometheus-compatible) endpoint",
		Args:    cobra.NoArgs,
		Example: listMetricNamesExamples,
		RunE:    runListMetricNamesCommand,
	}

	cmd.Flags().String(metricsEndpointFlag, configuredMetricsEndpoint(), "Mimir (Prometheus-compatible) base URL")
	cmd.Flags().String(orgIDFlag, viper.GetString(orgIDFlag), "Mimir tenant ID sent as X-Scope-OrgID")
	cmd.Flags().String("filter", "", "Only show metric names containing this substring")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getListMetricsOutputFormat(cmd *cobra.Command) (string, error) {
	return resolveTableOutputTemplate(cmd, DEFAULT_LIST_METRICS_FORMAT, METRICS_OUTPUT_TEMPLATE_ENVVAR)
}

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

	body, err := executePrometheusGET(ctx, client, prometheusLabelValuesAPIPath, argID)
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

func getGetMetricCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "metric <metric-name>",
		Short:   "Query a metric from a Mimir (Prometheus-compatible) endpoint for a specific hostname",
		Args:    cobra.ExactArgs(1),
		Example: getMetricExamples,
		RunE:    runGetMetricCommand,
	}

	cmd.Aliases = []string{"metrics"}
	cmd.Flags().String(hostnameFlag, viper.GetString(hostnameFlag), "Hostname label value to match in Prometheus")
	cmd.Flags().String(hostnameLabelFlag, defaultHostnameLabel, "Prometheus label name used to match the hostname")
	cmd.Flags().String(metricsEndpointFlag, configuredMetricsEndpoint(), "Mimir (Prometheus-compatible) base URL")
	cmd.Flags().String(orgIDFlag, viper.GetString(orgIDFlag), "Mimir tenant ID sent as X-Scope-OrgID")
	cmd.Flags().Bool(averageFlag, false, "Calculate average of metric over time range (use either --duration or --start-time with --end-time)")
	cmd.Flags().Bool(sumFlag, false, "Calculate sum of metric over time range (use either --duration or --start-time with --end-time)")
	cmd.Flags().Bool(rangeFlag, false, "Retrieve metric range values over time (use either --duration or --start-time with --end-time)")
	cmd.Flags().Int64(durationFlag, 0, "Duration in seconds for --sum/--average/--range calculation ending now (e.g. 3600 for last hour)")
	cmd.Flags().String(startTimeFlag, "", "Start time for range query (Unix timestamp, e.g. 1704067200)")
	cmd.Flags().String(endTimeFlag, "", "End time for range query (Unix timestamp, e.g. 1704153600)")
	cmd.Flags().String(timestampFlag, "", "Evaluate metric at a specific Unix timestamp (instant query mode)")
	addStandardGetOutputFlags(cmd)
	_ = cmd.MarkFlagRequired(hostnameFlag)

	return cmd
}

func getMetricOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_GET_METRIC_INSPECT_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_GET_METRIC_FORMAT, METRIC_OUTPUT_TEMPLATE_ENVVAR)
}

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

func lastPrometheusSample(samples [][]interface{}) []interface{} {
	if len(samples) == 0 {
		return nil
	}
	return samples[len(samples)-1]
}

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
	average, err := cmd.Flags().GetBool(averageFlag)
	if err != nil {
		return err
	}
	sum, err := cmd.Flags().GetBool(sumFlag)
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

	if average && sum {
		return fmt.Errorf("--average and --sum cannot be used together")
	}
	if rangeQuery && (average || sum) {
		return fmt.Errorf("--range cannot be used with --sum or --average")
	}

	if strings.TrimSpace(timestampStr) != "" {
		if average || sum || rangeQuery {
			return fmt.Errorf("--timestamp cannot be used with --sum, --average, or --range")
		}
		if durationSec > 0 || strings.TrimSpace(startTimeStr) != "" || strings.TrimSpace(endTimeStr) != "" {
			return fmt.Errorf("--timestamp cannot be used with --duration, --start-time, or --end-time")
		}
	}

	if durationSec > 0 && !average && !sum && !rangeQuery {
		return fmt.Errorf("--duration requires either --sum, --average, or --range")
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

		body, err := executePrometheusRangeQuery(ctx, client, query, startTime, endTime, argID)
		if err != nil {
			return err
		}

		if err := printMetricRangeResult(cmd, writer, metricName, hostnameLabel, body, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	if sum {
		startTime, endTime, err := resolveSumWindow(startTimeStr, endTimeStr, durationSec, time.Now())
		if err != nil {
			return err
		}

		durationSec = endTime - startTime
		query, err := buildSumMetricQuery(metricName, hostnameLabel, hostname, durationSec)
		if err != nil {
			return err
		}

		body, err := executePrometheusQueryAt(ctx, client, query, endTime, argID)
		if err != nil {
			return err
		}

		if err := printMetricResult(cmd, writer, metricName, hostnameLabel, body, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	// If average flag is set, support either duration or explicit start/end window.
	if average {
		startTime, endTime, err := resolveAverageWindow(startTimeStr, endTimeStr, durationSec, time.Now())
		if err != nil {
			return err
		}

		durationSec := endTime - startTime
		query, err := buildAverageMetricQuery(metricName, hostnameLabel, hostname, durationSec)
		if err != nil {
			return err
		}

		body, err := executePrometheusQueryAt(ctx, client, query, endTime, argID)
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

	body, err := executePrometheusQueryAt(ctx, client, query, evalTime, argID)
	if err != nil {
		return err
	}

	if err := printMetricResult(cmd, writer, metricName, hostnameLabel, body, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

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

// buildAverageMetricQuery builds a PromQL query with avg_over_time for range queries
// The query fetches the metric and computes the average over the time range.
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

	// Build instant query evaluated at end-time over the full requested range.
	return fmt.Sprintf(`avg_over_time(%s{%s=%q}[%ds])`, metricName, hostnameLabel, hostname, durationSec), nil
}

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

func newPrometheusClient(cmd *cobra.Command) (promapi.Client, error) {
	endpoint, err := getMetricsEndpoint(cmd)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("metrics endpoint not configured. Set --%s or run 'orch-cli config set %s <url>'", metricsEndpointFlag, metricsEndpointFlag)
	}

	roundTripper := metricsAuthRoundTripper{base: promapi.DefaultRoundTripper}
	client, err := promapi.NewClient(promapi.Config{
		Address: endpoint,
		Client: &http.Client{
			Timeout:   defaultMetricsTimeout,
			Transport: roundTripper,
		},
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func configuredMetricsEndpoint() string {
	return strings.TrimSpace(viper.GetString(metricsEndpointFlag))
}

func getMetricsEndpoint(cmd *cobra.Command) (string, error) {
	endpoint, err := cmd.Flags().GetString(metricsEndpointFlag)
	if err != nil {
		return "", err
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint != "" {
		return endpoint, nil
	}

	return configuredMetricsEndpoint(), nil
}

// resolveOrgID returns the tenant/project UID for Mimir queries
// Precedence: explicit --org-id flag > project UID > empty string
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

// getProjectUID fetches the UID for a given project name
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

func executePrometheusQueryAt(ctx context.Context, client promapi.Client, query string, evalTime int64, orgID string) ([]byte, error) {
	values := url.Values{}
	values.Set("query", query)
	values.Set("timeout", defaultMetricsTimeout.String())
	if evalTime > 0 {
		values.Set("time", fmt.Sprintf("%d", evalTime))
	}

	u := client.URL(prometheusQueryAPIPath, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if orgID != "" {
		req.Header.Set("X-Scope-OrgID", orgID)
	}
	resp, body, err := client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("prometheus query failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}

func executePrometheusRangeQuery(ctx context.Context, client promapi.Client, query string, startTime int64, endTime int64, orgID string) ([]byte, error) {
	rangeSec := endTime - startTime
	if rangeSec < 1 {
		return nil, fmt.Errorf("time range must be at least 1 second")
	}

	stepSec := rangeSec / 100
	if stepSec < 1 {
		stepSec = 1
	}

	values := url.Values{}
	values.Set("query", query)
	values.Set("start", fmt.Sprintf("%d", startTime))
	values.Set("end", fmt.Sprintf("%d", endTime))
	values.Set("step", fmt.Sprintf("%d", stepSec))
	values.Set("timeout", defaultMetricsTimeout.String())

	u := client.URL(prometheusQueryRangeAPIPath, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if orgID != "" {
		req.Header.Set("X-Scope-OrgID", orgID)
	}
	resp, body, err := client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("prometheus range query failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}

func executePrometheusGET(ctx context.Context, client promapi.Client, path string, orgID string) ([]byte, error) {
	u := client.URL(path, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if orgID != "" {
		req.Header.Set("X-Scope-OrgID", orgID)
	}

	resp, body, err := client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("prometheus request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

type metricsAuthRoundTripper struct {
	base http.RoundTripper
}

func (rt metricsAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if err := auth.AddAuthHeader(clone.Context(), clone); err != nil {
		return nil, err
	}
	return rt.base.RoundTrip(clone)
}
