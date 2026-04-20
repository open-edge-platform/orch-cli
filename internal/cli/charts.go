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
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
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
	cmd.Flags().StringP("output-type", "o", "table", "output type: table, json, yaml")
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
	var chartData interface{}
	if err := json.Unmarshal(data, &chartData); err != nil {
		return fmt.Errorf("failed to parse chart data: %w", err)
	}

	// Handle null response
	if chartData == nil {
		chartData = []interface{}{}
	}

	// Get output type
	outputType, err := cmd.Flags().GetString("output-type")
	if err != nil {
		outputType = "table"
	}

	// Output based on type
	switch outputType {
	case "json":
		jsonData, err := json.MarshalIndent(chartData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintf(writer, "%s\n", jsonData)
	case "yaml":
		yamlData, err := yaml.Marshal(chartData)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Fprintf(writer, "%s", yamlData)
	case "table":
		// For table output, format the data nicely
		if err := printChartsTable(writer, chartData, chartName); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown output type: %s (valid options: table, json, yaml)", outputType)
	}

	return writer.Flush()
}

func printChartsTable(writer io.Writer, data interface{}, chartName string) error {
	// If data is a slice, format as table
	if items, ok := data.([]interface{}); ok {
		if len(items) == 0 {
			if chartName != "" {
				fmt.Fprintf(writer, "No versions found for chart '%s'\n", chartName)
			} else {
				fmt.Fprintf(writer, "No charts found\n")
			}
			return nil
		}

		// Print header
		if chartName != "" {
			fmt.Fprintf(writer, "VERSION\n")
			// Print versions
			for _, item := range items {
				if str, ok := item.(string); ok {
					fmt.Fprintf(writer, "%s\n", str)
				} else {
					fmt.Fprintf(writer, "%v\n", item)
				}
			}
		} else {
			fmt.Fprintf(writer, "CHART NAME\n")
			// Print chart names
			for _, item := range items {
				if str, ok := item.(string); ok {
					fmt.Fprintf(writer, "%s\n", str)
				} else {
					fmt.Fprintf(writer, "%v\n", item)
				}
			}
		}
	} else {
		// Fallback: just print the data as-is
		fmt.Fprintf(writer, "%v\n", data)
	}
	return nil
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
