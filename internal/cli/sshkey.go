// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
)

const (
	DEFAULT_SSHKEY_FORMAT              = "table{{.Username}}\t{{str .ResourceId}}"
	DEFAULT_SSHKEY_LIST_VERBOSE_FORMAT = "table{{.Username}}\t{{str .ResourceId}}\t{{.InUse}}"
	DEFAULT_SSHKEY_GET_FORMAT          = "Remote User Name: \t{{.Username}}\nResource ID: \t{{str .ResourceId}}\nKey: \t{{.SshKey}}\nIn use by: \t{{.UseHosts}}\n"
	SSHKEY_OUTPUT_TEMPLATE_ENVVAR      = "ORCH_CLI_SSHKEY_OUTPUT_TEMPLATE"
)

// SSHKeyWithUsage wraps LocalAccountResource with usage information
type SSHKeyWithUsage struct {
	infra.LocalAccountResource
	InUse    string
	UseHosts string
}

const listSSHKeyExamples = `# List all SSH key resources
orch-cli list sshkey --project some-project
`

const getSSHKeyExamples = `# Get detailed information about specific SSH key resource using it's name
orch-cli get sshkey mysshkey --project some-project`

const createSSHKeyExamples = `# Create a new SSH key resource with a given name using a public key file as input
orch-cli create sshkey mysshkey /path/to/publickey.pub --project some-project`

const deleteSSHKeyExamples = `# Delete a SSH key resource using it's name
orch-cli delete sshkey mysshkey --project some-project`

func printSSHKeys(cmd *cobra.Command, writer io.Writer, sshKeys *[]infra.LocalAccountResource, instances *[]infra.InstanceResource, orderBy *string, outputFilter *string, verbose bool, forList bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	outputFormat, err := getSSHKeyOutputFormat(cmd, verbose, forList)
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

	// Create wrapper with usage information if instances are provided
	var data interface{}
	if instances != nil && len(*instances) > 0 {
		keysWithUsage := make([]SSHKeyWithUsage, 0, len(*sshKeys))
		for _, sshKey := range *sshKeys {
			inUse := "No"
			useHosts := ""
			for _, instance := range *instances {
				if instance.Localaccount != nil && *instance.Localaccount.ResourceId == *sshKey.ResourceId {
					inUse = "Yes"
					if instance.HostID != nil {
						if useHosts != "" {
							useHosts += " "
						}
						useHosts += *instance.HostID
					}
				}
			}
			keysWithUsage = append(keysWithUsage, SSHKeyWithUsage{
				LocalAccountResource: sshKey,
				InUse:                inUse,
				UseHosts:             useHosts,
			})
		}
		data = keysWithUsage
	} else {
		data = *sshKeys
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

func getSSHKeyOutputFormat(cmd *cobra.Command, verbose bool, forList bool) (string, error) {
	if verbose && forList {
		return DEFAULT_SSHKEY_LIST_VERBOSE_FORMAT, nil
	}
	if !forList {
		// Get command always shows full details
		return DEFAULT_SSHKEY_GET_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_SSHKEY_FORMAT, SSHKEY_OUTPUT_TEMPLATE_ENVVAR)
}

// Filters list of SSH keys to find one with specific name
func filterSSHKeysByName(SSHKeys []infra.LocalAccountResource, name string) (*infra.LocalAccountResource, error) {
	for _, key := range SSHKeys {
		if key.Username == name {
			return &key, nil
		}
	}
	return nil, errors.New("no SSH key matches the given name")
}

// Helper function to verify that the input file exists and is of right format
func verifySSHUserName(n string) error {

	pattern := `^[a-z][a-z0-9-]{0,31}$`

	// Compile the regular expression
	re := regexp.MustCompile(pattern)

	// Match the input string against the pattern
	if re.MatchString(n) {
		return nil
	}
	return errors.New("input is not a valid SSH username")
}

func readSSHKeyFromFile(certPath string) (string, error) {

	// Check if path is safe (no path traversal)
	if err := isSafePath(certPath); err != nil {
		return "", err
	}

	// Read file
	sshKeyData, err := os.ReadFile(certPath)
	if err != nil {
		return "", fmt.Errorf("failed to read ssh key file: %w", err)
	}
	sshKeyString := strings.TrimSpace(string(sshKeyData))

	// Validate key length
	if len(sshKeyString) > 800 {
		return "", fmt.Errorf("ssh key exceeds maximum length of 800 characters")
	}

	// Validate key format
	pattern := `^(ssh-ed25519|ecdsa-sha2-nistp521) ([A-Za-z0-9+/=]+) ?(.*)?$`
	matched, err := regexp.MatchString(pattern, sshKeyString)
	if err != nil {
		return "", fmt.Errorf("failed to validate ssh key format: %w", err)
	}
	if !matched {
		return "", fmt.Errorf("invalid ssh key format: must be ssh-ed25519 or ecdsa-sha2-nistp521")
	}

	return sshKeyString, nil
}

func getGetSSHKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sshkey <name> [flags]",
		Short:   "Get a SSH Key remote user configuration",
		Example: getSSHKeyExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: sshKeyAliases,
		RunE:    runGetSSHKeyCommand,
	}
	return cmd
}

func getListSSHKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sshkey [flags]",
		Short:   "List all SSH Key remote user configurations",
		Example: listSSHKeyExamples,
		Aliases: sshKeyAliases,
		RunE:    runListSSHKeyCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "sshkey")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getCreateSSHKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sshkey [flags]",
		Short:   "Creates SSH Key remote user configuration",
		Example: createSSHKeyExamples,
		Args:    cobra.ExactArgs(2),
		Aliases: sshKeyAliases,
		RunE:    runCreateSSHKeyCommand,
	}
	return cmd
}

func getDeleteSSHKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sshkey <name> [flags]",
		Short:   "Delete a SSH Key remote user configuration",
		Example: deleteSSHKeyExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: sshKeyAliases,
		RunE:    runDeleteSSHKeyCommand,
	}
	return cmd
}

// Gets specific SSH key configuration by resource ID
func runGetSSHKeyCommand(cmd *cobra.Command, args []string) error {
	writer, _ := getOutputContext(cmd)
	ctx, sshKeyClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := sshKeyClient.LocalAccountServiceListLocalAccountsWithResponse(ctx, projectName,
		&infra.LocalAccountServiceListLocalAccountsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", "error getting SSH key configuration"); !proceed {
		return err
	}

	name := args[0]
	sshKey, err := filterSSHKeysByName(resp.JSON200.LocalAccounts, name)
	if err != nil {
		return err
	}

	// Fetch instances to determine SSH key usage (always for get command)
	var instances []infra.InstanceResource
	pageSize := 100
	for offset := 0; ; offset += pageSize {
		iresp, err := sshKeyClient.InstanceServiceListInstancesWithResponse(ctx, projectName,
			&infra.InstanceServiceListInstancesParams{
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if iresp.JSON200 != nil {
			instances = append(instances, iresp.JSON200.Instances...)
			if !iresp.JSON200.HasNext {
				break
			}
		} else {
			break
		}
	}

	sshKeys := []infra.LocalAccountResource{*sshKey}
	var emptyFilter string
	// Get command always shows full details (forList=false)
	if err := printSSHKeys(cmd, writer, &sshKeys, &instances, nil, &emptyFilter, false, false); err != nil {
		return err
	}
	return writer.Flush()
}

// Lists all SSH keys - retrieves all keys and displays selected information in tabular format
func runListSSHKeyCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, sshKeyClient, projectName, err := InfraFactory(cmd)
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
		validatedOrderBy, err = normalizeOrderByForClientSorting(raw, infra.LocalAccountResource{})
	} else {
		validatedOrderBy, err = normalizeOrderByWithAPIProbe(raw, "sshkey", infra.LocalAccountResource{}, func(orderBy string) (bool, error) {
			pageSize := 1
			resp, err := sshKeyClient.LocalAccountServiceListLocalAccountsWithResponse(ctx, projectName,
				&infra.LocalAccountServiceListLocalAccountsParams{
					OrderBy:  &orderBy,
					PageSize: &pageSize,
				}, auth.AddAuthHeader)
			if err != nil {
				return false, processError(err)
			}
			if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
				return false, nil
			}
			if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating sshkey order-by"); err != nil {
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

	pageSize32, offset32, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	params := &infra.LocalAccountServiceListLocalAccountsParams{
		OrderBy: apiOrderBy,
		Filter:  getNonEmptyFlag(cmd, "filter"),
	}
	if pageSize32 > 0 {
		pageSize := int(pageSize32)
		params.PageSize = &pageSize
	}
	if offset32 > 0 {
		offset := int(offset32)
		params.Offset = &offset
	}

	resp, err := sshKeyClient.LocalAccountServiceListLocalAccountsWithResponse(ctx, projectName, params, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", "error getting SSH key configurations"); !proceed {
		return err
	}

	// Fetch instances to determine SSH key usage if in verbose mode
	var instances []infra.InstanceResource
	if verbose {
		pageSize := 100
		for offset := 0; ; offset += pageSize {
			iresp, err := sshKeyClient.InstanceServiceListInstancesWithResponse(ctx, projectName,
				&infra.InstanceServiceListInstancesParams{
					PageSize: &pageSize,
					Offset:   &offset,
				}, auth.AddAuthHeader)
			if err != nil {
				return processError(err)
			}
			if iresp.JSON200 != nil {
				instances = append(instances, iresp.JSON200.Instances...)
				if !iresp.JSON200.HasNext {
					break
				}
			} else {
				break
			}
		}
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	// List command (forList=true)
	if err := printSSHKeys(cmd, writer, &resp.JSON200.LocalAccounts, &instances, validatedOrderBy, &outputFilter, verbose, true); err != nil {
		return err
	}

	return writer.Flush()
}

// Creates SSH key configuration
func runCreateSSHKeyCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	path := args[1]

	err := verifySSHUserName(name)
	if err != nil {
		return err
	}

	key, err := readSSHKeyFromFile(path)
	if err != nil {
		return err
	}

	ctx, sshKeyClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := sshKeyClient.LocalAccountServiceCreateLocalAccountWithResponse(ctx, projectName,
		infra.LocalAccountServiceCreateLocalAccountJSONRequestBody{
			Username: name,
			SshKey:   key,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating SSH key from %s", path))
}

// Deletes SSH Key - checks if a key already exists and then deletes it if it does
func runDeleteSSHKeyCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	ctx, sshKeyClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	gresp, err := sshKeyClient.LocalAccountServiceListLocalAccountsWithResponse(ctx, projectName,
		&infra.LocalAccountServiceListLocalAccountsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err = checkResponse(gresp.HTTPResponse, gresp.Body, "Error getting SSH keys"); err != nil {
		return err
	}

	sshKey, err := filterSSHKeysByName(gresp.JSON200.LocalAccounts, name)
	if err != nil {
		return err
	}

	resp, err := sshKeyClient.LocalAccountServiceDeleteLocalAccountWithResponse(ctx, projectName,
		*sshKey.ResourceId, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting SSH key %s", name))
}
