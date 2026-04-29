// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/spf13/cobra"
)

// ChartInfo represents a chart entry in the API response
type ChartInfo struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

func getListChartsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "charts <registry-name> [<chart-name>] [flags]",
		Args:              cobra.MinimumNArgs(1),
		Short:             "List Helm charts or chart versions from a registry",
		PersistentPreRunE: auth.CheckAuth,
		Example: `# List all charts in a registry
	orch-cli list charts my-registry --project my-project

	# List versions for a specific chart
	orch-cli list charts my-registry kubevirt --project my-project

	# Output as JSON
	orch-cli list charts my-registry --project my-project --output-type json`,
		Aliases: chartAliases,
		RunE:    runListChartsCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "charts")
	addStandardListOutputFlags(cmd)
	return cmd
}

func runListChartsCommand(cmd *cobra.Command, args []string) error {
	writer, _ := getOutputContext(cmd)

	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return err
	}

	projectName, err := getProjectName(cmd)
	if err != nil {
		return err
	}

	registryName := args[0]
	var chartName string
	if len(args) > 1 {
		chartName = args[1]
	}

	url := fmt.Sprintf("%s/v3/projects/%s/catalog/charts?registry=%s", serverAddress, projectName, registryName)
	if chartName != "" {
		url = fmt.Sprintf("%s&chart=%s", url, chartName)
	}

	data, err := getRegistryContent(url)
	if err != nil {
		return err
	}

	// Parse the JSON response
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse chart data: %w", err)
	}
	if raw == nil {
		raw = []interface{}{}
	}

	// Normalize into []ChartInfo
	charts := make([]ChartInfo, 0)
	switch v := raw.(type) {
	case []interface{}:
		for _, it := range v {
			switch it2 := it.(type) {
			case string:
				if chartName != "" {
					charts = append(charts, ChartInfo{Version: it2})
				} else {
					charts = append(charts, ChartInfo{Name: it2})
				}
			case map[string]interface{}:
				ci := ChartInfo{}
				if n, ok := it2["name"].(string); ok {
					ci.Name = n
				}
				if ver, ok := it2["version"].(string); ok {
					ci.Version = ver
				}
				charts = append(charts, ci)
			default:
				charts = append(charts, ChartInfo{Name: fmt.Sprintf("%v", it)})
			}
		}
	default:
		charts = append(charts, ChartInfo{Name: fmt.Sprintf("%v", v)})
	}

	outputType, _ := cmd.Flags().GetString("output-type")

	sortSpec := ""
	filterSpec := ""
	if outputType == "table" {
		rawOrder, _ := cmd.Flags().GetString("order-by")
		if rawOrder != "" {
			sortSpec = rawOrder
		}
		of, _ := cmd.Flags().GetString("output-filter")
		if of != "" {
			filterSpec = of
		}
	}

	const DEFAULT_CHARTS_FORMAT = "table{{.Name}}"
	const DEFAULT_CHARTS_VERSION_FORMAT = "table{{.Version}}"
	const CHARTS_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_CHARTS_OUTPUT_TEMPLATE"

	if outputType == "json" || outputType == "yaml" {
		result := CommandResult{
			OutputAs: toOutputType(outputType),
			Data:     charts,
		}
		GenerateOutput(writer, &result)
		return writer.Flush()
	}

	outputFormat := DEFAULT_CHARTS_FORMAT
	if chartName != "" {
		outputFormat = DEFAULT_CHARTS_VERSION_FORMAT
	}

	// Resolve any template overrides
	tmpl, err := resolveTableOutputTemplate(cmd, outputFormat, CHARTS_OUTPUT_TEMPLATE_ENVVAR)
	if err != nil {
		return err
	}

	result := CommandResult{
		Format:    format.Format(tmpl),
		Filter:    filterSpec,
		OrderBy:   sortSpec,
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      charts,
	}
	GenerateOutput(writer, &result)
	return writer.Flush()
}

func getRegistryContent(url string) ([]byte, error) {
	ctx := context.Background()
	r, _ := http.NewRequest("GET", url, nil)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("CORS", "true")
	if err := auth.AddAuthHeader(ctx, r); err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return []byte("[]"), nil
	} else if resp.StatusCode != 200 {
		return nil, errors.NewInvalid("chart retrieval failed: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
