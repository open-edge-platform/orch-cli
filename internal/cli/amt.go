// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/open-edge-platform/cli/pkg/format"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/rps"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

const listAmtProfileExamples = `
# List all AMT domain profiles
orch-cli list amtprofile --project some-project
`

const createAmtProfileExamples = `
# Create AMT domain profile information using it's name
orch-cli create amtprofile name --project some-project --cert ./path/to/cert.pfx --cert-pass password --cert-format string --domain-suffix example.com

--cert - Mandatory path to PFX certificate: --cert ./path/to/cert.pfx
--cert-pass - Mandatory password used ot decode the provided certificate: --cert-pass mypass
--cert-format - Mandatory field defining how the cert is stored, accepted value "string" or "raw": --cert-format string
--domain-suffix - Mandatory field defining the domain suffix for which the cert is created: --domain-suffix example.com
`

const getAmtProfileExamples = `
# Get an AMT domain profile
orch-cli get  amtprofile name --project some-project
`
const deleteAmtProfileExamples = `
# Delete an AMT domain profile
orch-cli delete amtprofile name --project some-project
`

const (
	DEFAULT_AMTPROFILE_FORMAT         = "table{{.ProfileName}}\t{{.DomainSuffix}}"
	DEFAULT_AMTPROFILE_VERBOSE_FORMAT = "table{{.ProfileName}}\t{{.DomainSuffix}}\t{{.Version}}\t{{.ProvisioningCertStorageFormat}}\t{{formatTime .ExpirationDate}}"
	DEFAULT_AMTPROFILE_INSPECT_FORMAT = "Name: \t{{.ProfileName}}\nDomain Suffix: \t{{.DomainSuffix}}\nVersion: \t{{.Version}}\nTenant ID: \t{{.TenantId}}\nCert Format: \t{{.ProvisioningCertStorageFormat}}\nExpiration Date: \t{{formatTime .ExpirationDate}}\n"
	AMTPROFILE_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_AMTPROFILE_OUTPUT_TEMPLATE"
)

func getListAmtProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "amtprofile [flags]",
		Short:   "List all amptprofiles",
		Example: listAmtProfileExamples,
		Aliases: amtAliases,
		RunE:    runListAmtProfileCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "amtprofile")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getGetAmtProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "amtprofile <resourceid> [flags]",
		Short:   "Get an AMT profile",
		Example: getAmtProfileExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: amtAliases,
		RunE:    runGetAmtProfileCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getCreateAmtProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "amtprofile name [flags]",
		Short:   "Create an AMT profile",
		Example: createAmtProfileExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: amtAliases,
		RunE:    runCreateAmtProfileCommand,
	}
	cmd.PersistentFlags().StringP("domain-suffix", "d", viper.GetString("domain-suffix"), "Mandatory domain name suffix")
	cmd.PersistentFlags().StringP("cert", "c", viper.GetString("cert"), "Mandatory path to SSL certificate")
	cmd.PersistentFlags().StringP("cert-pass", "s", viper.GetString("cert-pass"), "Mandatory password for SSL certificate")
	cmd.PersistentFlags().StringP("cert-format", "f", viper.GetString("cert-format"), "Mandatory format of SSL certificate")

	return cmd
}

func getDeleteAmtProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "amtprofile <resourceid> [flags]",
		Short:   "Delete an AMT profile",
		Example: deleteAmtProfileExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: amtAliases,
		RunE:    runDeleteAmtProfileCommand,
	}
	return cmd
}

// Lists all AMT profiles
func runListAmtProfileCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	count := true
	ctx, rpsClient, projectName, err := RpsFactory(cmd)
	if err != nil {
		return err
	}
	// Validate order-by flag and determine API vs client-side ordering
	outputType, _ := cmd.Flags().GetString("output-type")
	var validatedOrderBy *string
	raw, _ := cmd.Flags().GetString("order-by")
	if outputType == "table" {
		var err error
		validatedOrderBy, err = normalizeOrderByForClientSorting(raw, rps.DomainResponse{})
		if err != nil {
			return err
		}
	} else {
		// The rps API does not support server-side ordering. If the user
		// requested JSON/YAML output with an --order-by, surface a helpful error
		// rather than silently ignoring it.
		if strings.TrimSpace(raw) != "" {
			return fmt.Errorf("server does not support server-side --order-by for AMT profiles; remove --order-by or use --output-type table for client-side sorting")
		}
	}

	// Prepare paging params
	pageSize32, _ := cmd.Flags().GetInt32("page-size")
	offset32, _ := cmd.Flags().GetInt32("offset")
	var top *int
	var skip *int
	if pageSize32 > 0 {
		v := int(pageSize32)
		top = &v
	}
	if offset32 > 0 {
		v := int(offset32)
		skip = &v
	}

	// API does not support server-side order-by for AMT; always fetch and sort client-side
	resp, err := rpsClient.GetAllDomainsWithResponse(ctx, projectName, &rps.GetAllDomainsParams{
		Top:   top,
		Skip:  skip,
		Count: &count,
	}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err := checkResponse(resp.HTTPResponse, resp.Body, "error while retrieving AMT profiles"); err != nil {
		return err
	}

	var countDomainResponse rps.CountDomainResponse

	// Unmarshal the JSON data into the CountDomainResponse struct
	// The autogenerated client returns a raw union for the 200 response; use the raw body here
	err = json.Unmarshal(resp.Body, &countDomainResponse)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return err
	}

	// Prepare data slice for output
	data := []rps.DomainResponse{}
	if countDomainResponse.Data != nil {
		data = *countDomainResponse.Data
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printAmtProfiles(cmd, writer, &data, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}

	return writer.Flush()
}

func runCreateAmtProfileCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	certpath, _ := cmd.Flags().GetString("cert")
	certpass, _ := cmd.Flags().GetString("cert-pass")
	certformat, _ := cmd.Flags().GetString("cert-format")
	domainsuffix, _ := cmd.Flags().GetString("domain-suffix")

	cert, err := readCert(certpath)
	if err != nil {
		return err
	}

	if certpass == "" {
		fmt.Print("Enter Certificate Password: ")
		bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		certpass = string(bytePassword)
	}

	if certpass == "" || strings.HasPrefix(certpass, "--") {
		return errors.New("certificate password must be provided with --cert-pass flag and cannot be empty")
	}
	if certformat == "" || certformat != "string" && certformat != "raw" {
		return errors.New("certificate format must be provided with --cert-format flag with accepted arguments `string|raw` ")
	}
	if domainsuffix == "" || strings.HasPrefix(domainsuffix, "--") {
		return errors.New("domain suffix format must be provided with --domain-suffix flag and cannot be empty")
	}

	ctx, rpsClient, projectName, err := RpsFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := rpsClient.CreateDomainWithResponse(ctx, projectName,
		rps.CreateDomainJSONRequestBody{
			DomainSuffix:                  domainsuffix,
			ProfileName:                   name,
			ProvisioningCert:              cert,
			ProvisioningCertPassword:      certpass,
			ProvisioningCertStorageFormat: certformat,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, "error while creating AMT")
}

func runGetAmtProfileCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, rpsClient, projectName, err := RpsFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]

	resp, err := rpsClient.GetDomainWithResponse(ctx, projectName,
		name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting AMT profile"); !proceed {
		return err
	}

	if err := printAmtProfile(cmd, writer, *resp.JSON200, verbose); err != nil {
		return err
	}
	return writer.Flush()

}

func runDeleteAmtProfileCommand(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, rpsClient, projectName, err := RpsFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := rpsClient.RemoveDomainWithResponse(ctx, projectName,
		name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	err = checkResponse(resp.HTTPResponse, resp.Body, "error while deleting AMT profile")
	if err != nil {
		if strings.Contains(string(resp.Body), `"Not Found"`) {
			return errors.New("AMT profile does not exist")
		}
	}
	return err
}

func printAmtProfiles(cmd *cobra.Command, writer io.Writer, amtprofiles *[]rps.DomainResponse, orderBy *string, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getAMTProfileOutputFormat(cmd, verbose, true)
	if err != nil {
		return err
	}

	sortSpec := ""
	filterSpec := ""
	data := interface{}(*amtprofiles)

	if outputType == "table" {
		if orderBy != nil {
			sortSpec = *orderBy
		}
		if outputFilter != nil && *outputFilter != "" {
			filterSpec = *outputFilter
		}
		// Header is generated by the output formatter; avoid printing manual headers
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    filterSpec,
		OrderBy:   sortSpec,
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      data,
	}
	GenerateOutput(writer, &result)
	return nil
}

// Prints output details of a single AMT profile
func printAmtProfile(cmd *cobra.Command, writer io.Writer, amtprofile rps.DomainResponse, verbose bool) error {
	outputFormat, err := getAMTProfileOutputFormat(cmd, verbose, false)
	if err != nil {
		return err
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    "",
		OrderBy:   "",
		OutputAs:  toOutputType("table"),
		NameLimit: -1,
		Data:      amtprofile,
	}

	GenerateOutput(writer, &result)
	return nil
}

func getAMTProfileOutputFormat(cmd *cobra.Command, verbose bool, forList bool) (string, error) {
	if verbose && forList {
		return DEFAULT_AMTPROFILE_VERBOSE_FORMAT, nil
	}
	if !forList {
		return DEFAULT_AMTPROFILE_INSPECT_FORMAT, nil
	}
	return resolveTableOutputTemplate(cmd, DEFAULT_AMTPROFILE_FORMAT, AMTPROFILE_OUTPUT_TEMPLATE_ENVVAR)
}

func getValidatedAMTProfileOrderBy(ctx interface{}, cmd *cobra.Command, rpsClient rps.ClientWithResponsesInterface, projectName string) (*string, error) {
	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return nil, err
	}

	// rps API does not expose an order-by parameter; use client-side sorting for all outputs
	return normalizeOrderByForClientSorting(raw, rps.DomainResponse{})
}

func readCert(certPath string) ([]byte, error) {

	if certPath == "" || strings.HasPrefix(certPath, "--") {
		return nil, errors.New("certificate path must be provided with --cert flag and cannot be empty")
	}

	if err := isSafePath(certPath); err != nil {
		return nil, err
	}

	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	return certData, nil
}
