// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
)

const listSSHKeyExamples = `# List all SSH key resources
orch-cli list sshkey --project some-project
`

const getSSHKeyExamples = `# Get detailed information about specific SSH key resource using it's name
orch-cli get sshkey mysshkey --project some-project`

const createSSHKeyExamples = `# Create a new SSH key resource with a given name using a public key file as input
orch-cli create sshkey mysshkey /path/to/publickey.pub --project some-project`

const deleteSSHKeyExamples = `# Delete a SSH key resource using it's name
orch-cli delete sshkey mysshkey --project some-project`

var SSHKeyHeader = fmt.Sprintf("\n%s\t%s", "Remote User", "Resource ID")

// Prints SSH keys in tabular format
func printSSHKeys(writer io.Writer, SSHKeys []infra.LocalAccountResource, instances []infra.InstanceResource, verbose bool) {
	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\n", "Remote User", "Resource ID", "In use")
	}

	for _, sshKey := range SSHKeys {
		inUse := "No"
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\n", sshKey.Username, *sshKey.ResourceId)
		} else {
			for _, instance := range instances {
				if instance.Localaccount != nil && *instance.Localaccount.ResourceId == *sshKey.ResourceId {
					inUse = "Yes"
					break
				}
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\n", sshKey.Username, *sshKey.ResourceId, inUse)
		}
	}
}

// Prints output details of SSH key
func printSSHKey(writer io.Writer, SSHKey *infra.LocalAccountResource, instances []infra.InstanceResource) {

	useHosts := ""
	for _, instance := range instances {
		if instance.Localaccount != nil && *instance.Localaccount.ResourceId == *SSHKey.ResourceId {
			useHosts = useHosts + *instance.HostID + " "
		}
	}

	_, _ = fmt.Fprintf(writer, "Remote User Name: \t%s\n", SSHKey.Username)
	_, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *SSHKey.ResourceId)
	_, _ = fmt.Fprintf(writer, "Key: \t%s\n", SSHKey.SshKey)
	_, _ = fmt.Fprintf(writer, "In use by: \t%s\n\n", useHosts)
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

	writer, verbose := getOutputContext(cmd)
	ctx, sshKeyClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := sshKeyClient.LocalAccountServiceListLocalAccountsWithResponse(ctx, projectName,
		&infra.LocalAccountServiceListLocalAccountsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting SSH key configuration"); !proceed {
		return err
	}

	name := args[0]
	sshKey, err := filterSSHKeysByName(resp.JSON200.LocalAccounts, name)
	if err != nil {
		return err
	}

	if err := checkResponse(resp.HTTPResponse, "error while retrieving ssh key"); err != nil {
		return err
	}

	pageSize := 20
	instances := make([]infra.InstanceResource, 0)
	for offset := 0; ; offset += pageSize {
		iresp, err := sshKeyClient.InstanceServiceListInstancesWithResponse(ctx, projectName,
			&infra.InstanceServiceListInstancesParams{}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(iresp.HTTPResponse, "error while retrieving instances"); err != nil {
			return err
		}

		instances = append(instances, iresp.JSON200.Instances...)
		if !iresp.JSON200.HasNext {
			break // No more instances to process
		}
	}

	printSSHKey(writer, sshKey, instances)
	return writer.Flush()
}

// Lists all SSH keys - retrieves all keys and displays selected information in tabular format
func runListSSHKeyCommand(cmd *cobra.Command, _ []string) error {

	writer, verbose := getOutputContext(cmd)
	pageSize := 20

	ctx, sshKeyClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := sshKeyClient.LocalAccountServiceListLocalAccountsWithResponse(ctx, projectName,
		&infra.LocalAccountServiceListLocalAccountsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		SSHKeyHeader, "error getting SSH key configurations"); !proceed {
		return err
	}

	instances := make([]infra.InstanceResource, 0)
	if verbose {
		for offset := 0; ; offset += pageSize {
			iresp, err := sshKeyClient.InstanceServiceListInstancesWithResponse(ctx, projectName,
				&infra.InstanceServiceListInstancesParams{}, auth.AddAuthHeader)
			if err != nil {
				return processError(err)
			}
			if proceed, err := processResponse(resp.HTTPResponse, iresp.Body, writer, verbose,
				SSHKeyHeader, "error getting instances"); !proceed {
				return err
			}

			instances = append(instances, iresp.JSON200.Instances...)
			if !iresp.JSON200.HasNext {
				break // No more instances to process
			}
		}
	}

	printSSHKeys(writer, resp.JSON200.LocalAccounts, instances, verbose)

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
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating SSH key from %s", path))
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

	if err = checkResponse(gresp.HTTPResponse, "Error getting SSH keys"); err != nil {
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

	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting SSH key %s", name))
}
