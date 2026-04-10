// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/rest/keycloak"
	"github.com/spf13/cobra"
)

const listGroupsExamples = `# List all groups
orch-cli list groups

# List all groups in a specific realm
orch-cli list groups --realm master
`

var GroupHeader = fmt.Sprintf("\n%s\t%s", "Name", "ID")

func printGroups(writer io.Writer, groups []keycloak.GroupRepresentation, verbose bool) {
	if len(groups) == 0 {
		fmt.Fprintf(writer, "No groups found\n")
		return
	}

	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\n", "Name", "ID", "Path")
	}

	for _, group := range groups {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\n", group.Name, group.ID)
		} else {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", group.Name, group.ID, group.Path)
		}
	}
}

func getListGroupsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "groups [flags]",
		Short:   "List all groups",
		Example: listGroupsExamples,
		Aliases: groupAliases,
		RunE:    runListGroupsCommand,
	}
	cmd.Flags().String("realm", "master", "Keycloak realm")
	return cmd
}

func runListGroupsCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, kcClient, realm, err := KeycloakAdminFactory(cmd)
	if err != nil {
		return err
	}

	groups, err := kcClient.ListGroups(ctx, realm)
	if err != nil {
		return fmt.Errorf("error listing groups: %w", err)
	}

	if !verbose {
		_, _ = fmt.Fprintf(writer, "%s\n", GroupHeader)
	}

	printGroups(writer, groups, verbose)
	return writer.Flush()
}
