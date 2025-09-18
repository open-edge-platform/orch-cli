// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
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

var ProviderHeader = fmt.Sprintf("\n%s\t%s\t%s\t%s", "Name", "Resource ID", "Kind", "Vendor")

func getListProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "provider [flags]",
		Short:   "List all providers",
		Example: listProviderExamples,
		Aliases: []string{"provider", "providers"},
		RunE:    runListProviderCommand,
	}
	return cmd
}

func getGetProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "provider <resourceid> [flags]",
		Short:   "Get a provider",
		Example: getProviderExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"provider", "providers"},
		RunE:    runGetProviderCommand,
	}
	return cmd
}

func getCreateProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "provider name <vendor> <apiendpoint> [flags]",
		Short:   "Create a provider",
		Example: createProviderExamples,
		Args:    cobra.ExactArgs(3),
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

	resp, err := providerClient.ProviderServiceListProvidersWithResponse(ctx, projectName,
		&infra.ProviderServiceListProvidersParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		ProviderHeader, "error getting provider"); !proceed {
		return err
	}
	printProviders(writer, resp.JSON200.Providers, verbose)

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
	return checkResponse(resp.HTTPResponse, "error while creating provider")

}

func runGetProviderCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
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

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting provider"); !proceed {
		return err
	}

	printProvider(writer, resp.JSON200)
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

	err = checkResponse(resp.HTTPResponse, "error while deleting provider")
	if err != nil {
		if strings.Contains(string(resp.Body), `"message":"provider_resource not found"`) {
			return errors.New("provider does not exist")
		}
	}
	return err
}

// Prints providers in tabular format
func printProviders(writer io.Writer, providers []infra.ProviderResource, verbose bool) {
	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "Name", "Resource ID", "Kind", "Vendor", "API Endpoint", "Created At", "Updated At")
	}
	for _, prov := range providers {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%v\n", prov.Name, *prov.ResourceId, prov.ProviderKind, *prov.ProviderVendor)
		} else {

			fmt.Fprintf(writer, "%s\t%s\t%s\t%v\t%s\t%s\t%s\n", prov.Name, *prov.ResourceId, prov.ProviderKind, *prov.ProviderVendor, prov.ApiEndpoint, prov.Timestamps.CreatedAt, prov.Timestamps.UpdatedAt)
		}
	}
}

// Prints output details of site
func printProvider(writer io.Writer, provider *infra.ProviderResource) {

	var config string
	if provider.Config != nil {
		config = *provider.Config
	}

	_, _ = fmt.Fprintf(writer, "Name: \t%s\n", provider.Name)
	_, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *provider.ResourceId)
	_, _ = fmt.Fprintf(writer, "Kind: \t%s\n", provider.ProviderKind)
	_, _ = fmt.Fprintf(writer, "Vendor: \t%v\n", *provider.ProviderVendor)
	_, _ = fmt.Fprintf(writer, "API Endpoint: \t%s\n", provider.ApiEndpoint)
	_, _ = fmt.Fprintf(writer, "Config: \t%s\n", config)
	_, _ = fmt.Fprintf(writer, "Created At: \t%s\n", provider.Timestamps.CreatedAt)
	_, _ = fmt.Fprintf(writer, "Updated At: \t%s\n", provider.Timestamps.UpdatedAt)

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
