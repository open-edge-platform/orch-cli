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
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
)

const (
	DEFAULT_ARTIFACT_FORMAT         = "table{{.Name}}\t{{.DisplayName}}\t{{.Description}}"
	DEFAULT_ARTIFACT_INSPECT_FORMAT = `Name: {{.Name}}
Display Name: {{str .DisplayName}}
Description: {{str .Description}}
Mime Type: {{.MimeType}}
`
	ARTIFACT_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_ARTIFACT_OUTPUT_TEMPLATE"
)

func getCreateArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifact <name> [flags]",
		Short:   "Create an artifact",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli create artifact my-artifact --mime-type application/octet-stream --artifact /path/to/artifact --project some-project",
		Aliases: artifactAliases,
		RunE:    runCreateArtifactCommand,
	}
	addEntityFlags(cmd, "artifact")
	cmd.Flags().String("mime-type", "", "artifact MIME type (required)")
	_ = cmd.MarkFlagRequired("mime-type")
	cmd.Flags().String("artifact", "-", "path to the artifact file; - for stdin")
	return cmd
}

func getListArtifactsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifacts [flags]",
		Short:   "List all artifacts",
		Example: "orch-cli list artifacts --project some-project --order-by name",
		Aliases: artifactAliases,
		RunE:    runListArtifactsCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "artifact")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getGetArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifact <name> [flags]",
		Short:   "Get an artifact",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli get artifact my-artifact --project some-project",
		Aliases: artifactAliases,
		RunE:    runGetArtifactCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getSetArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifact <name> [flags]",
		Short:   "Update an artifact",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli set artifact my-artifact --mime-type application/octet-stream --artifact /path/to/artifact --project some-project",
		Aliases: artifactAliases,
		RunE:    runSetArtifactCommand,
	}
	addEntityFlags(cmd, "artifact")
	cmd.Flags().String("mime-type", "", "artifact MIME type")
	cmd.Flags().String("artifact", "", "path to the artifact file; - for stdin")
	return cmd
}

func getDeleteArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifact <name> [flags]",
		Short:   "Delete an artifact",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli delete artifact my-artifact --project some-project",
		Aliases: artifactAliases,
		RunE:    runDeleteArtifactCommand,
	}
	return cmd
}

func getArtifactOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_ARTIFACT_INSPECT_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_ARTIFACT_FORMAT, ARTIFACT_OUTPUT_TEMPLATE_ENVVAR)
}

func printArtifacts(cmd *cobra.Command, writer io.Writer, artifactList *[]catapi.CatalogV3Artifact, orderBy *string, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getArtifactOutputFormat(cmd, verbose)
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
		Data:      *artifactList,
	}

	GenerateOutput(writer, &result)
	return nil
}

func runCreateArtifactCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}
	displayName, description, err := getEntityFlags(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	binary, err := readInput(*getFlag(cmd, "artifact"))
	if err != nil {
		return fmt.Errorf("error reading artifact content: %w", err)
	}

	resp, err := catalogClient.CatalogServiceCreateArtifactWithResponse(ctx, projectName,
		catapi.CatalogServiceCreateArtifactJSONRequestBody{
			Name:        name,
			DisplayName: &displayName,
			Description: &description,
			MimeType:    *getFlag(cmd, "mime-type"),
			Artifact:    binary,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating artifact %s", name)); err != nil {
		return err
	}
	fmt.Printf("Artifact '%s' created successfully\n", name)
	return nil
}

func getValidatedArtifactOrderBy(
	ctx context.Context,
	cmd *cobra.Command,
	catalogClient catapi.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return nil, err
	}

	outputType, _ := cmd.Flags().GetString("output-type")

	// For table format (default), use client-side sorting which supports any field in the model
	if outputType == "table" {
		return normalizeOrderByForClientSorting(raw, catapi.CatalogV3Artifact{})
	}

	// For JSON/YAML, use API ordering (only API-supported fields)
	return normalizeOrderByWithAPIProbe(raw, "artifacts", catapi.CatalogV3Artifact{}, func(orderBy string) (bool, error) {
		pageSize := int32(1)
		offset := int32(0)
		// Validate ordering in isolation. Reusing the caller's --filter here can turn
		// filter errors into misleading "invalid --order-by field" errors.
		resp, err := catalogClient.CatalogServiceListArtifactsWithResponse(ctx, projectName,
			&catapi.CatalogServiceListArtifactsParams{
				OrderBy:  &orderBy,
				Filter:   nil,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
			return false, nil
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating artifact order-by"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func runListArtifactsCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	validatedOrderBy, err := getValidatedArtifactOrderBy(ctx, cmd, catalogClient, projectName)
	if err != nil {
		return err
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	apiOrderBy := validatedOrderBy
	if outputType == "table" {
		// Table output sorts locally via GenerateOutput(CommandResult.OrderBy).
		apiOrderBy = nil
	}

	pageSize, offset, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	// Preserve explicit pagination requests as single-page results.
	if cmd.Flags().Changed("page-size") || cmd.Flags().Changed("offset") {
		resp, err := catalogClient.CatalogServiceListArtifactsWithResponse(ctx, projectName,
			&catapi.CatalogServiceListArtifactsParams{
				OrderBy:  apiOrderBy,
				Filter:   getFlag(cmd, "filter"),
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing artifacts"); !proceed {
			return err
		}
		outputFilter, _ := cmd.Flags().GetString("output-filter")
		if err := printArtifacts(cmd, writer, &resp.JSON200.Artifacts, validatedOrderBy, &outputFilter, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	allArtifacts := make([]catapi.CatalogV3Artifact, 0)

	resp, err := catalogClient.CatalogServiceListArtifactsWithResponse(ctx, projectName,
		&catapi.CatalogServiceListArtifactsParams{
			OrderBy:  apiOrderBy,
			Filter:   getFlag(cmd, "filter"),
			PageSize: &pageSize,
			Offset:   &offset,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
		"error listing artifacts"); !proceed {
		return err
	}

	allArtifacts = append(allArtifacts, resp.JSON200.Artifacts...)
	totalElements := int(resp.JSON200.TotalElements)

	// When page size is omitted (0), derive increment from the first page length.
	if pageSize <= 0 {
		pageSize = int32(len(resp.JSON200.Artifacts))
	}

	for len(allArtifacts) < totalElements {
		if pageSize <= 0 {
			break
		}

		offset += pageSize
		resp, err = catalogClient.CatalogServiceListArtifactsWithResponse(ctx, projectName,
			&catapi.CatalogServiceListArtifactsParams{
				OrderBy:  apiOrderBy,
				Filter:   getFlag(cmd, "filter"),
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing artifacts"); !proceed {
			return err
		}

		if len(resp.JSON200.Artifacts) == 0 {
			break
		}
		allArtifacts = append(allArtifacts, resp.JSON200.Artifacts...)
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printArtifacts(cmd, writer, &allArtifacts, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

func runGetArtifactCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	resp, err := catalogClient.CatalogServiceGetArtifactWithResponse(ctx, projectName, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
		fmt.Sprintf("error getting artifact %s", name)); !proceed {
		return err
	}
	if err := printArtifacts(cmd, writer, &[]catapi.CatalogV3Artifact{resp.JSON200.Artifact}, nil, nil, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

func runSetArtifactCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	gresp, err := catalogClient.CatalogServiceGetArtifactWithResponse(ctx, projectName, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("artifact %s not found", name)); err != nil {
		return err
	}

	artifact := gresp.JSON200.Artifact
	binary := gresp.Body

	// If the artifact flag was given, fetch the new content to replace the existing one
	newArtifactPath := *getFlag(cmd, "artifact")
	if len(newArtifactPath) > 0 {
		binary, err = readInput(*getFlag(cmd, "artifact"))
		if err != nil {
			return fmt.Errorf("error reading artifact content: %w", err)
		}
	}

	resp, err := catalogClient.CatalogServiceUpdateArtifactWithResponse(ctx, projectName, name,
		catapi.CatalogServiceUpdateArtifactJSONRequestBody{
			Name:        name,
			DisplayName: getFlagOrDefault(cmd, "display-name", artifact.DisplayName),
			Description: getFlagOrDefault(cmd, "description", artifact.Description),
			MimeType:    *getFlagOrDefault(cmd, "mime-type", &artifact.MimeType),
			Artifact:    binary,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while updating artifact %s", name)); err != nil {
		return err
	}
	fmt.Printf("Artifact '%s' updated successfully\n", name)
	return nil
}

func runDeleteArtifactCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	gresp, err := catalogClient.CatalogServiceGetArtifactWithResponse(ctx, projectName, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("artifact %s not found", name)); err != nil {
		return err
	}

	resp, err := catalogClient.CatalogServiceDeleteArtifactWithResponse(ctx, projectName, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting artifact %s", name)); err != nil {
		return err
	}
	fmt.Printf("Artifact '%s' deleted successfully\n", name)
	return nil
}

func printArtifactEvent(writer io.Writer, _ string, payload []byte, verbose bool) error {
	var item catapi.CatalogV3Artifact
	if err := json.Unmarshal(payload, &item); err != nil {
		return err
	}
	if !verbose {
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", item.Name, valueOrNone(item.DisplayName), valueOrNone(item.Description))
	} else {
		_, _ = fmt.Fprintf(writer, "Name: %s\n", item.Name)
		_, _ = fmt.Fprintf(writer, "Display Name: %s\n", valueOrNone(item.DisplayName))
		_, _ = fmt.Fprintf(writer, "Description: %s\n", valueOrNone(item.Description))
		_, _ = fmt.Fprintf(writer, "Mime Type: %s\n\n", item.MimeType)
	}
	return nil
}
