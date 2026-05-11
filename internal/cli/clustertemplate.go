// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	coapi "github.com/open-edge-platform/cli/pkg/rest/cluster"
	"github.com/spf13/cobra"
)

const (
	DEFAULT_CLUSTER_TEMPLATE_FORMAT         = "table{{.Name}}\t{{str .Description}}\t{{.Version}}\t{{.KubernetesVersion}}"
	DEFAULT_CLUSTER_TEMPLATE_INSPECT_FORMAT = `Name: {{.Name}}
Description: {{str .Description}}
Version: {{.Version}}
Kubernetes Version: {{.KubernetesVersion}}{{if .Controlplaneprovidertype}}
Control Plane Provider: {{.Controlplaneprovidertype}}{{end}}{{if .Infraprovidertype}}
Infrastructure Provider: {{.Infraprovidertype}}{{end}}{{if .ClusterLabels}}
Cluster Labels:{{range $key, $value := deref .ClusterLabels}}
  {{$key}}: {{$value}}{{end}}{{end}}{{if .ClusterNetwork}}
Cluster Network:{{if .ClusterNetwork.Pods}}{{if .ClusterNetwork.Pods.CidrBlocks}}
  Pod CIDR: {{range .ClusterNetwork.Pods.CidrBlocks}}{{.}} {{end}}{{else}}
  Pod CIDR: <none>{{end}}{{else}}
  Pod CIDR: <none>{{end}}{{if .ClusterNetwork.Services}}{{if .ClusterNetwork.Services.CidrBlocks}}
  Service CIDR: {{range .ClusterNetwork.Services.CidrBlocks}}{{.}} {{end}}{{else}}
  Service CIDR: <none>{{end}}{{else}}
  Service CIDR: <none>{{end}}{{end}}
Configuration: Use --output-type yaml to view full cluster configuration
`
	CLUSTER_TEMPLATE_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_CLUSTER_TEMPLATE_OUTPUT_TEMPLATE"
)

func getListClusterTemplatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clustertemplates [flags]",
		Aliases: clusterTemplateAliases,
		Short:   "List all cluster templates",
		Example: "orch-cli list clustertemplates --project some-project",
		RunE:    runListClusterTemplatesCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "cluster template")
	addStandardListOutputFlags(cmd)
	return cmd
}

func runListClusterTemplatesCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, clusterTemplateClient, projectName, err := ClusterFactory(cmd)
	if err != nil {
		return err
	}

	validatedOrderBy, err := getValidatedClusterTemplateOrderBy(ctx, cmd, clusterTemplateClient, projectName)
	if err != nil {
		return err
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	apiOrderBy := validatedOrderBy
	if outputType == "table" {
		// Table output sorts locally via GenerateOutput(CommandResult.OrderBy).
		apiOrderBy = nil
	}

	validatedFilter, err := getValidatedClusterTemplateFilter(ctx, cmd, clusterTemplateClient, projectName)
	if err != nil {
		return err
	}

	pageSize, offset, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	// Convert int32 to int for cluster API
	// Only pass non-zero values (cluster API requires pageSize > 0)
	var pageSizePtr *int
	var offsetPtr *int
	if pageSize > 0 {
		pageSizeInt := int(pageSize)
		pageSizePtr = &pageSizeInt
	}
	if offset > 0 {
		offsetInt := int(offset)
		offsetPtr = &offsetInt
	}

	// Preserve explicit pagination requests as single-page results.
	if cmd.Flags().Changed("page-size") || cmd.Flags().Changed("offset") {
		resp, err := clusterTemplateClient.GetV2ProjectsProjectNameTemplatesWithResponse(ctx, projectName,
			&coapi.GetV2ProjectsProjectNameTemplatesParams{
				OrderBy:  apiOrderBy,
				Filter:   validatedFilter,
				PageSize: pageSizePtr,
				Offset:   offsetPtr,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing cluster templates"); !proceed {
			return err
		}
		if resp.JSON200 == nil || resp.JSON200.TemplateInfoList == nil {
			return fmt.Errorf("error listing cluster templates: unexpected response format")
		}
		outputFilter, _ := cmd.Flags().GetString("output-filter")
		if err := printClusterTemplates(cmd, writer, resp.JSON200.TemplateInfoList, validatedOrderBy, &outputFilter, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	allTemplates := make([]coapi.TemplateInfo, 0)

	resp, err := clusterTemplateClient.GetV2ProjectsProjectNameTemplatesWithResponse(ctx, projectName,
		&coapi.GetV2ProjectsProjectNameTemplatesParams{
			OrderBy:  apiOrderBy,
			Filter:   validatedFilter,
			PageSize: pageSizePtr,
			Offset:   offsetPtr,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
		"error listing cluster templates"); !proceed {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.TemplateInfoList == nil {
		return fmt.Errorf("error listing cluster templates: unexpected response format")
	}

	allTemplates = append(allTemplates, *resp.JSON200.TemplateInfoList...)
	var totalElements int
	if resp.JSON200.TotalElements != nil {
		totalElements = int(*resp.JSON200.TotalElements)
	}

	// When page size is omitted (0), derive increment from the first page length.
	if pageSize <= 0 {
		pageSize = int32(len(*resp.JSON200.TemplateInfoList))
		if pageSize > 0 {
			pageSizeInt := int(pageSize)
			pageSizePtr = &pageSizeInt
		}
	}

	for len(allTemplates) < totalElements {
		if pageSize <= 0 {
			break
		}
		offset += pageSize
		offsetInt := int(offset)
		offsetPtr = &offsetInt
		resp, err := clusterTemplateClient.GetV2ProjectsProjectNameTemplatesWithResponse(ctx, projectName,
			&coapi.GetV2ProjectsProjectNameTemplatesParams{
				OrderBy:  apiOrderBy,
				Filter:   validatedFilter,
				PageSize: pageSizePtr,
				Offset:   offsetPtr,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing cluster templates"); !proceed {
			return err
		}
		if resp.JSON200 == nil || resp.JSON200.TemplateInfoList == nil {
			break
		}
		allTemplates = append(allTemplates, *resp.JSON200.TemplateInfoList...)
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printClusterTemplates(cmd, writer, &allTemplates, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

func printClusterTemplates(cmd *cobra.Command, writer io.Writer, templates *[]coapi.TemplateInfo, orderBy *string, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getClusterTemplateOutputFormat(cmd, verbose)
	if err != nil {
		return err
	}

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
		Data:      *templates,
	}

	GenerateOutput(writer, &result)
	return nil
}

func getClusterTemplateOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_CLUSTER_TEMPLATE_INSPECT_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_CLUSTER_TEMPLATE_FORMAT, CLUSTER_TEMPLATE_OUTPUT_TEMPLATE_ENVVAR)
}

func getValidatedClusterTemplateOrderBy(
	ctx context.Context,
	cmd *cobra.Command,
	clusterClient coapi.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return nil, err
	}

	outputType, _ := cmd.Flags().GetString("output-type")

	// For table format (default), use client-side sorting which supports any field in the model
	if outputType == "table" {
		return normalizeOrderByForClientSorting(raw, coapi.TemplateInfo{})
	}

	// For JSON/YAML, use API ordering (only API-supported fields)
	return normalizeOrderByWithAPIProbe(raw, "cluster-templates", coapi.TemplateInfo{}, func(orderBy string) (bool, error) {
		pageSize := 1
		offset := 0
		// Validate ordering in isolation. Reusing the caller's --filter here can turn
		// filter errors into misleading "invalid --order-by field" errors.
		resp, err := clusterClient.GetV2ProjectsProjectNameTemplatesWithResponse(ctx, projectName,
			&coapi.GetV2ProjectsProjectNameTemplatesParams{
				OrderBy:  &orderBy,
				Filter:   nil,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
			return false, &api400Error{string(resp.Body)}
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating cluster template order-by"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func getValidatedClusterTemplateFilter(
	ctx context.Context,
	cmd *cobra.Command,
	clusterClient coapi.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("filter")
	if err != nil {
		return nil, err
	}

	return normalizeFilterWithAPIProbe(raw, "cluster-templates", coapi.TemplateInfo{}, func(filter string) (bool, error) {
		pageSize := 1
		offset := 0
		resp, err := clusterClient.GetV2ProjectsProjectNameTemplatesWithResponse(ctx, projectName,
			&coapi.GetV2ProjectsProjectNameTemplatesParams{
				OrderBy:  nil,
				Filter:   &filter,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
			return false, &api400Error{string(resp.Body)}
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating cluster template filter"); err != nil {
			return false, err
		}
		return true, nil
	})
}
