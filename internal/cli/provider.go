// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

const (
	DEFAULT_PROVIDER_FORMAT              = "table{{.Name}}\t{{str .ResourceId}}\t{{.ProviderKind}}\t{{.ProviderVendor}}"
	DEFAULT_PROVIDER_LIST_VERBOSE_FORMAT = "table{{.Name}}\t{{str .ResourceId}}\t{{.ProviderKind}}\t{{.ProviderVendor}}\t{{.ApiEndpoint}}\t{{.Timestamps.CreatedAt}}\t{{.Timestamps.UpdatedAt}}"
	DEFAULT_PROVIDER_GET_FORMAT          = "Name: \t{{.Name}}\nResource ID: \t{{str .ResourceId}}\nKind: \t{{.ProviderKind}}\nVendor: \t{{.ProviderVendor}}\nAPI Endpoint: \t{{.ApiEndpoint}}\nConfig: \t{{str .Config}}\nCreation Timestamp: \t{{.Timestamps.CreatedAt}}\nUpdated Timestamp: \t{{.Timestamps.UpdatedAt}}\n"
	PROVIDER_OUTPUT_TEMPLATE_ENVVAR      = "ORCH_CLI_PROVIDER_OUTPUT_TEMPLATE"
)

const listProviderExamples = `# List all providers
orch-cli list provider --project some-project`

const getProviderExamples = `# Get specific provider information using resource ID
orch-cli get provider provider-aaaa1111 --project some-project`

const createProviderExamples = `# Create specific provider
# Create a provider by providing name, kind, and empty API endpoint
orch-cli create provider myprovider "PROVIDER_KIND_BAREMETAL" "" --vendor "PROVIDER_VENDOR_UNSPECIFIED" --config ""defaultOs":"","autoProvision":false,"defaultLocalAccount":"","osSecurityFeatureEnable":false" --project some-project`

const deleteProviderExamples = `# Delete specific provider
orch-cli delete provider provider-aaaa1111 --project some-project`

func printProviders(cmd *cobra.Command, writer io.Writer, providers *[]infra.ProviderResource, orderBy *string, outputFilter *string, verbose bool, forList bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	outputFormat, err := getProviderOutputFormat(cmd, verbose, forList)
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
		Data:      *providers,
	}

	GenerateOutput(writer, &result)
	return nil
}

func getProviderOutputFormat(cmd *cobra.Command, verbose bool, forList bool) (string, error) {
	if verbose && forList {
		return DEFAULT_PROVIDER_LIST_VERBOSE_FORMAT, nil
	}
	if !forList {
		// Get command always shows full details
		return DEFAULT_PROVIDER_GET_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_PROVIDER_FORMAT, PROVIDER_OUTPUT_TEMPLATE_ENVVAR)
}

func getListProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "provider [flags]",
		Short:   "List all providers",
		Example: listProviderExamples,
		Aliases: providerAliases,
		RunE:    runListProviderCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "provider")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getGetProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "provider <resourceid> [flags]",
		Short:   "Get a provider",
		Example: getProviderExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: providerAliases,
		RunE:    runGetProviderCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getCreateProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "provider name <vendor> <apiendpoint> [flags]",
		Short:   "Create a provider",
		Example: createProviderExamples,
		Args:    cobra.ExactArgs(3),
		Aliases: providerAliases,
		RunE:    runCreateProviderCommand,
	}
	cmd.PersistentFlags().BoolP("apicredentials", "a", viper.GetBool("apicredentials"), "Flag to accept API credentials for the provider: --apicredentials")
	cmd.PersistentFlags().StringP("config", "c", viper.GetString("config"), "Optional flag to provide config: --config <config>")
	cmd.PersistentFlags().StringP("vendor", "x", viper.GetString("vendor"), "Optional flag to provide vendor: --vendor <vendor>")
	return cmd
}

func getDeleteProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "provider <resourceid> [flags]",
		Short:   "Delete a provider",
		Example: deleteProviderExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: providerAliases,
		RunE:    runDeleteProviderCommand,
	}
	return cmd
}

// Lists all providers - retrieves all providers and displays selected information in tabular format
func runListProviderCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, providerClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return err
	}

	outputType, _ := cmd.Flags().GetString("output-type")

	var validatedOrderBy *string
	if outputType == "table" {
		validatedOrderBy, err = normalizeOrderByForClientSorting(raw, infra.ProviderResource{})
	} else {
		validatedOrderBy, err = normalizeOrderByWithAPIProbe(raw, "provider", infra.ProviderResource{}, func(orderBy string) (bool, error) {
			pageSize := 1
			resp, err := providerClient.ProviderServiceListProvidersWithResponse(ctx, projectName,
				&infra.ProviderServiceListProvidersParams{
					OrderBy:  &orderBy,
					PageSize: &pageSize,
				}, auth.AddAuthHeader)
			if err != nil {
				return false, processError(err)
			}
			if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
				return false, nil
			}
			if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating provider order-by"); err != nil {
				return false, err
			}
			return true, nil
		})
	}
	if err != nil {
		return err
	}

	apiOrderBy := validatedOrderBy
	if outputType == "table" {
		// Table output sorts locally via GenerateOutput(CommandResult.OrderBy).
		apiOrderBy = nil
	}

	validatedFilter, err := getValidatedProviderFilter(ctx, cmd, providerClient, projectName)
	if err != nil {
		return err
	}

	pageSize32, offset32, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	pageSize := int(pageSize32)
	offset := int(offset32)
	if pageSize <= 0 {
		pageSize = 100
	}

	// Preserve explicit pagination requests as single-page results.
	if cmd.Flags().Changed("page-size") || cmd.Flags().Changed("offset") {
		params := &infra.ProviderServiceListProvidersParams{
			OrderBy:  apiOrderBy,
			Filter:   validatedFilter,
			PageSize: &pageSize,
			Offset:   &offset,
		}

		resp, err := providerClient.ProviderServiceListProvidersWithResponse(ctx, projectName, params, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}

		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
			"", "error getting provider"); !proceed {
			return err
		}

		if resp.JSON200 == nil || resp.JSON200.Providers == nil {
			return fmt.Errorf("error listing providers: unexpected response format")
		}

		providers := resp.JSON200.Providers

		outputFilter, _ := cmd.Flags().GetString("output-filter")
		if err := printProviders(cmd, writer, &providers, validatedOrderBy, &outputFilter, verbose, true); err != nil {
			return err
		}
		return writer.Flush()
	}

	// Automatic pagination: fetch all pages
	allProviders := make([]infra.ProviderResource, 0)

	resp, err := providerClient.ProviderServiceListProvidersWithResponse(ctx, projectName,
		&infra.ProviderServiceListProvidersParams{
			OrderBy:  apiOrderBy,
			Filter:   validatedFilter,
			PageSize: &pageSize,
			Offset:   &offset,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", "error getting provider"); !proceed {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Providers == nil {
		return fmt.Errorf("error listing providers: unexpected response format")
	}

	allProviders = append(allProviders, resp.JSON200.Providers...)
	totalElements := int(resp.JSON200.TotalElements)

	// When page size is omitted (0), derive increment from the first page length.
	if pageSize <= 0 {
		pageSize = len(resp.JSON200.Providers)
	}

	for len(allProviders) < totalElements {
		if pageSize <= 0 {
			break
		}
		offset += pageSize
		resp, err := providerClient.ProviderServiceListProvidersWithResponse(ctx, projectName,
			&infra.ProviderServiceListProvidersParams{
				OrderBy:  apiOrderBy,
				Filter:   validatedFilter,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
			"", "error getting provider"); !proceed {
			return err
		}

		if resp.JSON200 == nil || resp.JSON200.Providers == nil {
			return fmt.Errorf("error listing providers: unexpected response format")
		}

		if len(resp.JSON200.Providers) == 0 {
			break
		}
		allProviders = append(allProviders, resp.JSON200.Providers...)
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printProviders(cmd, writer, &allProviders, validatedOrderBy, &outputFilter, verbose, true); err != nil {
		return err
	}

	return writer.Flush()
}

func runCreateProviderCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	kind := args[1]
	api := args[2]

	configFlag, _ := cmd.Flags().GetString("config")
	vendorFlag, _ := cmd.Flags().GetString("vendor")

	var apiCredentials string
	var err error
	isAPICerds, _ := cmd.Flags().GetBool("apicredentials")

	if isAPICerds {
		fmt.Print("Enter API credentials (comma-separated if multiple): ")
		apiByteCredentials, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		apiCredentials = strings.TrimSpace(string(apiByteCredentials))
		fmt.Println("API credentials set.")
	}

	var apiCredentialsPtr *[]string
	if apiCredentials != "" {
		creds := strings.Split(apiCredentials, ",")
		apiCredentialsPtr = &creds
	}

	var providerVendorPtr *infra.ProviderVendor
	if vendorFlag != "" {
		v := infra.ProviderVendor(vendorFlag)
		providerVendorPtr = &v
	}

	if api == "" || api == " " || api == "null" {
		fmt.Print("Warning: Setting API endpoint to 'null'.\n")
		api = "null"
	}

	if kind != "PROVIDER_KIND_UNSPECIFIED" && kind != "PROVIDER_KIND_BAREMETAL" {
		fmt.Print("Warning: It is recommended to use the default provider kind 'PROVIDER_KIND_BAREMETAL' unless you have a specific requirement for a different kind. Accepted values: \"PROVIDER_KIND_UNSPECIFIED\", \"PROVIDER_KIND_BAREMETAL\"\n")
		return errors.New("invalid provider kind. Accepted values: \"PROVIDER_KIND_UNSPECIFIED\", \"PROVIDER_KIND_BAREMETAL\"")
	}

	if vendorFlag != "" && vendorFlag != "PROVIDER_VENDOR_UNSPECIFIED" && vendorFlag != "PROVIDER_VENDOR_LENOVO_LXCA" && vendorFlag != "PROVIDER_VENDOR_LENOVO_LOCA" {
		return errors.New("invalid vendor. Accepted values: \"PROVIDER_VENDOR_UNSPECIFIED\", \"PROVIDER_VENDOR_LENOVO_LXCA\", \"PROVIDER_VENDOR_LENOVO_LOCA\"")
	}

	ctx, providerClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := providerClient.ProviderServiceCreateProviderWithResponse(ctx, projectName, infra.ProviderServiceCreateProviderJSONRequestBody{
		Name:           name,
		ProviderKind:   infra.ProviderKind(kind),
		ApiEndpoint:    api,
		Config:         &configFlag,
		ProviderVendor: providerVendorPtr,
		ApiCredentials: apiCredentialsPtr,
	}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, "error while creating provider")

}

func getValidatedProviderFilter(
	ctx context.Context,
	cmd *cobra.Command,
	providerClient infra.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("filter")
	if err != nil {
		return nil, err
	}

	return normalizeFilterWithAPIProbe(raw, "provider", infra.ProviderResource{}, func(filter string) (bool, error) {
		pageSize := 1
		resp, err := providerClient.ProviderServiceListProvidersWithResponse(ctx, projectName,
			&infra.ProviderServiceListProvidersParams{
				OrderBy:  nil,
				Filter:   &filter,
				PageSize: &pageSize,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
			return false, nil
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating provider filter"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func runGetProviderCommand(cmd *cobra.Command, args []string) error {
	writer, _ := getOutputContext(cmd)
	ctx, providerClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	id := args[0]

	resp, err := providerClient.ProviderServiceGetProviderWithResponse(ctx, projectName,
		id, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", "error getting provider"); !proceed {
		return err
	}

	providers := []infra.ProviderResource{*resp.JSON200}
	var emptyFilter string
	// Get command always shows full details (forList=false)
	if err := printProviders(cmd, writer, &providers, nil, &emptyFilter, false, false); err != nil {
		return err
	}

	return writer.Flush()
}

func runDeleteProviderCommand(cmd *cobra.Command, args []string) error {
	id := args[0]

	err := printWarning()
	if err != nil {
		return err
	}

	ctx, providerClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := providerClient.ProviderServiceDeleteProviderWithResponse(ctx, projectName,
		id, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	err = checkResponse(resp.HTTPResponse, resp.Body, "error while deleting provider")
	if err != nil {
		if strings.Contains(string(resp.Body), `"message":"provider_resource not found"`) {
			return errors.New("provider does not exist")
		}
	}
	return err
}

func printWarning() error {
	fmt.Println("Warning: Usage of the default provider is recommended. Deleting a provider in use by other resources may lead to resource misconfiguration. This action may have unintended consequences.")
	fmt.Println("Are you sure you want to proceed? (y/n)")
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil || (response != "y" && response != "Y") {
		return errors.New("operation cancelled by user")
	}
	fmt.Println("Proceeding with deletion...")
	return nil
}
