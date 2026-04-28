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
	DEFAULT_REGISTRY_FORMAT         = "table{{.Name}}\t{{str .DisplayName}}\t{{str .Description}}\t{{.Type}}\t{{.RootUrl}}"
	DEFAULT_REGISTRY_INSPECT_FORMAT = `Name: {{.Name}}
Display Name: {{str .DisplayName}}
Description: {{str .Description}}
Root URL: {{.RootUrl}}
Inventory URL: {{none .InventoryUrl}}
Type: {{.Type}}
API Type: {{none .ApiType}}
Username: {{none .Username}}{{if .AuthToken}}
AuthToken: ********{{else}}
AuthToken: <none>{{end}}{{if .Cacerts}}
CA Certs: ********{{else}}
CA Certs: <none>{{end}}
Create Time: {{fmttime .CreateTime}}
Update Time: {{fmttime .UpdateTime}}
`
	DEFAULT_REGISTRY_INSPECT_FORMAT_SENSITIVE = `Name: {{.Name}}
Display Name: {{str .DisplayName}}
Description: {{str .Description}}
Root URL: {{.RootUrl}}
Inventory URL: {{none .InventoryUrl}}
Type: {{.Type}}
API Type: {{none .ApiType}}
Username: {{none .Username}}
AuthToken: {{none .AuthToken}}
CA Certs: {{none .Cacerts}}
Create Time: {{fmttime .CreateTime}}
Update Time: {{fmttime .UpdateTime}}
`
	REGISTRY_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_REGISTRY_OUTPUT_TEMPLATE"
)

func getCreateRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry <name> [flags]",
		Short:   "Create a registry",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli create registry my-registry --root-url https://my-registry.example.com --username my-user --auth-token my-token --project some-project",
		Aliases: registryAliases,
		RunE:    runCreateRegistryCommand,
	}
	addEntityFlags(cmd, "registry")
	cmd.Flags().String("root-url", "", "root URL of the registry (required)")
	_ = cmd.MarkFlagRequired("root-url")
	cmd.Flags().String("username", "", "username for accessing the registry")
	cmd.Flags().String("auth-token", "", "authentication token for accessing the registry")
	cmd.Flags().String("ca-certs", "", "CA certs for accessing the registry")
	cmd.Flags().String("registry-type", "helm", "registry type (helm or image)")
	cmd.Flags().String("inventory-url", "", "inventory URL of the registry")
	cmd.Flags().String("api-type", "helm", "registry API type")
	return cmd
}

func getListRegistriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registries [flags]",
		Aliases: registryAliases,
		Short:   "List all registries",
		Example: "orch-cli list registries --project some-project",
		RunE:    runListRegistriesCommand,
	}
	// Help hint for client-side --output-filter
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	// annotations removed: dynamic header-derived hints will be used instead
	addListOrderingFilteringPaginationFlags(cmd, "registry")
	addStandardListOutputFlags(cmd)
	cmd.Flags().Bool("show-sensitive-info", false, "show sensitive info, e.g. auth-token, CA certs")
	return cmd
}

func getGetRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry <name> [flags]",
		Short:   "Get a registry",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli get registry my-registry --project some-project",
		Aliases: registryAliases,
		RunE:    runGetRegistryCommand,
	}
	cmd.Flags().Bool("show-sensitive-info", false, "show sensitive info, e.g. auth-token, CA certs")
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getSetRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry <name> [flags]",
		Short:   "Update a registry",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli set registry my-registry --root-url https://my-registry.example.com --username my-user --auth-token my-token --project some-project",
		Aliases: registryAliases,
		RunE:    runSetRegistryCommand,
	}
	addEntityFlags(cmd, "registry")
	cmd.Flags().String("root-url", "", "root URL of the registry")
	cmd.Flags().String("username", "", "username for accessing the registry")
	cmd.Flags().String("auth-token", "", "authentication token for accessing the registry")
	cmd.Flags().String("ca-certs", "", "CA certs for accessing the registry")
	cmd.Flags().String("registry-type", "helm", "registry type (helm or image)")
	cmd.Flags().String("inventory-url", "", "inventory URL of the registry")
	cmd.Flags().String("api-type", "helm", "registry API type")
	return cmd
}

func getDeleteRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry <name> [flags]",
		Short:   "Delete a registry",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli delete registry my-registry --project some-project",
		Aliases: registryAliases,
		RunE:    runDeleteRegistryCommand,
	}
	return cmd
}

func printRegistries(cmd *cobra.Command, writer io.Writer, registryList *[]catapi.CatalogV3Registry, orderBy *string, outputFilter *string, verbose bool, showSensitive bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	outputFormat, err := getRegistryOutputFormat(cmd, verbose, showSensitive)
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
		Data:      *registryList,
	}

	GenerateOutput(writer, &result)
	return nil
}

func getRegistryOutputFormat(cmd *cobra.Command, verbose bool, showSensitive bool) (string, error) {
	if verbose {
		if showSensitive {
			return DEFAULT_REGISTRY_INSPECT_FORMAT_SENSITIVE, nil
		}
		return DEFAULT_REGISTRY_INSPECT_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_REGISTRY_FORMAT, REGISTRY_OUTPUT_TEMPLATE_ENVVAR)
}

func verifyRegistryType(cmd *cobra.Command) error {
	regType := *getFlag(cmd, "registry-type")
	if regType == "helm" || regType == "image" {
		return nil
	}
	return fmt.Errorf("invalid registry type %s", regType)
}

func getRegistryType(cmd *cobra.Command) string {
	typeFromCommand := *getFlag(cmd, "registry-type")
	switch typeFromCommand {
	case "helm":
		return "HELM"
	case "image":
		return "IMAGE"
	}
	return ""
}

func runCreateRegistryCommand(cmd *cobra.Command, args []string) error {
	err := verifyRegistryType(cmd)
	if err != nil {
		return err
	}
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}
	displayName, description, err := getEntityFlags(cmd)
	if err != nil {
		return err
	}
	name := args[0]
	registryType := getRegistryType(cmd)

	resp, err := catalogClient.CatalogServiceCreateRegistryWithResponse(ctx, projectName,
		catapi.CatalogServiceCreateRegistryJSONRequestBody{
			Name:         name,
			DisplayName:  &displayName,
			Description:  &description,
			RootUrl:      *getFlag(cmd, "root-url"),
			InventoryUrl: getFlag(cmd, "inventory-url"),
			Username:     getFlag(cmd, "username"),
			AuthToken:    getFlag(cmd, "auth-token"),
			Cacerts:      getFlag(cmd, "ca-certs"),
			Type:         registryType,
			ApiType:      getFlag(cmd, "api-type"),
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating registry %s", name)); err != nil {
		return err
	}
	fmt.Printf("Registry '%s' created successfully\n", name)
	return nil
}

func getValidatedRegistryOrderBy(
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
		return normalizeOrderByForClientSorting(raw, catapi.CatalogV3Registry{})
	}

	// For JSON/YAML, use API ordering (only API-supported fields)
	showSensitive, _ := cmd.Flags().GetBool("show-sensitive-info")
	return normalizeOrderByWithAPIProbe(raw, "registries", catapi.CatalogV3Registry{}, func(orderBy string) (bool, error) {
		pageSize := int32(1)
		offset := int32(0)
		// Validate ordering in isolation. Reusing the caller's --filter here can turn
		// filter errors into misleading "invalid --order-by field" errors.
		resp, err := catalogClient.CatalogServiceListRegistriesWithResponse(ctx, projectName,
			&catapi.CatalogServiceListRegistriesParams{
				OrderBy:           &orderBy,
				Filter:            nil,
				PageSize:          &pageSize,
				Offset:            &offset,
				ShowSensitiveInfo: &showSensitive,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
			return false, nil
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating registry order-by"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func runListRegistriesCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	validatedOrderBy, err := getValidatedRegistryOrderBy(ctx, cmd, catalogClient, projectName)
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

	showSensitive, _ := cmd.Flags().GetBool("show-sensitive-info")

	// Preserve explicit pagination requests as single-page results.
	if cmd.Flags().Changed("page-size") || cmd.Flags().Changed("offset") {
		resp, err := catalogClient.CatalogServiceListRegistriesWithResponse(ctx, projectName,
			&catapi.CatalogServiceListRegistriesParams{
				OrderBy:           apiOrderBy,
				Filter:            getFlag(cmd, "filter"),
				PageSize:          &pageSize,
				Offset:            &offset,
				ShowSensitiveInfo: &showSensitive,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing registries"); !proceed {
			return err
		}
		outputFilter, _ := cmd.Flags().GetString("output-filter")
		if err := printRegistries(cmd, writer, &resp.JSON200.Registries, validatedOrderBy, &outputFilter, verbose, showSensitive); err != nil {
			return err
		}
		return writer.Flush()
	}

	allRegistries := make([]catapi.CatalogV3Registry, 0)

	resp, err := catalogClient.CatalogServiceListRegistriesWithResponse(ctx, projectName,
		&catapi.CatalogServiceListRegistriesParams{
			OrderBy:           apiOrderBy,
			Filter:            getFlag(cmd, "filter"),
			PageSize:          &pageSize,
			Offset:            &offset,
			ShowSensitiveInfo: &showSensitive,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
		"error listing registries"); !proceed {
		return err
	}

	allRegistries = append(allRegistries, resp.JSON200.Registries...)
	totalElements := int(resp.JSON200.TotalElements)

	// When page size is omitted (0), derive increment from the first page length.
	if pageSize <= 0 {
		pageSize = int32(len(resp.JSON200.Registries))
	}

	for len(allRegistries) < totalElements {
		if pageSize <= 0 {
			break
		}

		offset += pageSize
		resp, err = catalogClient.CatalogServiceListRegistriesWithResponse(ctx, projectName,
			&catapi.CatalogServiceListRegistriesParams{
				OrderBy:           apiOrderBy,
				Filter:            getFlag(cmd, "filter"),
				PageSize:          &pageSize,
				Offset:            &offset,
				ShowSensitiveInfo: &showSensitive,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing registries"); !proceed {
			return err
		}

		if len(resp.JSON200.Registries) == 0 {
			break
		}
		allRegistries = append(allRegistries, resp.JSON200.Registries...)
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printRegistries(cmd, writer, &allRegistries, validatedOrderBy, &outputFilter, verbose, showSensitive); err != nil {
		return err
	}
	return writer.Flush()
}

func runGetRegistryCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	showSensitive, _ := cmd.Flags().GetBool("show-sensitive-info")

	resp, err := catalogClient.CatalogServiceGetRegistryWithResponse(ctx, projectName, name,
		&catapi.CatalogServiceGetRegistryParams{ShowSensitiveInfo: &showSensitive}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", fmt.Sprintf("error getting registry %s", name)); !proceed {
		return err
	}

	var emptyFilter string
	if err := printRegistries(cmd, writer, &[]catapi.CatalogV3Registry{resp.JSON200.Registry}, nil, &emptyFilter, verbose, showSensitive); err != nil {
		return err
	}
	return writer.Flush()
}

func runSetRegistryCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	showSensitive := true
	gresp, err := catalogClient.CatalogServiceGetRegistryWithResponse(ctx, projectName, name,
		&catapi.CatalogServiceGetRegistryParams{ShowSensitiveInfo: &showSensitive}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("registry %s not found", name)); err != nil {
		return err
	}

	registry := gresp.JSON200.Registry

	// Get registry type - use flag value if provided, otherwise keep existing
	registryType := registry.Type
	if cmd.Flags().Changed("registry-type") {
		registryType = getRegistryType(cmd)
	}

	resp, _ := catalogClient.CatalogServiceUpdateRegistryWithResponse(ctx, projectName, name,
		catapi.CatalogServiceUpdateRegistryJSONRequestBody{
			Name:         name,
			DisplayName:  getFlagOrDefault(cmd, "display-name", registry.DisplayName),
			Description:  getFlagOrDefault(cmd, "description", registry.Description),
			RootUrl:      *getFlagOrDefault(cmd, "root-url", &registry.RootUrl),
			InventoryUrl: getFlagOrDefault(cmd, "inventory-url", registry.InventoryUrl),
			Username:     getFlagOrDefault(cmd, "username", registry.Username),
			AuthToken:    getFlagOrDefault(cmd, "auth-token", registry.AuthToken),
			Cacerts:      getFlagOrDefault(cmd, "ca-certs", registry.Cacerts),
			Type:         registryType,
			ApiType:      getFlagOrDefault(cmd, "api-type", registry.ApiType),
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while updating registry %s", name)); err != nil {
		return err
	}
	fmt.Printf("Registry '%s' updated successfully\n", name)
	return nil
}

func runDeleteRegistryCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	gresp, err := catalogClient.CatalogServiceGetRegistryWithResponse(ctx, projectName, name,
		&catapi.CatalogServiceGetRegistryParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("registry %s not found", name)); err != nil {
		return err
	}

	resp, err := catalogClient.CatalogServiceDeleteRegistryWithResponse(ctx, projectName, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting registry %s", name)); err != nil {
		return err
	}
	fmt.Printf("Registry '%s' deleted successfully\n", name)
	return nil
}

func printRegistryEvent(writer io.Writer, _ string, payload []byte, verbose bool) error {
	var item catapi.CatalogV3Registry
	if err := json.Unmarshal(payload, &item); err != nil {
		return err
	}
	// Create a dummy command to pass to printRegistries (events don't support full output features)
	cmd := &cobra.Command{}
	cmd.Flags().String("output-type", "table", "")
	var emptyFilter string
	return printRegistries(cmd, writer, &[]catapi.CatalogV3Registry{item}, nil, &emptyFilter, verbose, false)
}
