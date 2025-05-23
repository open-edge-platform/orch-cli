// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"io"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/spf13/cobra"
)

func getCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "create",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Create various catalog service entities",
		PersistentPreRunE: auth.CheckAuth,
	}

	cmd.AddCommand(
		getCreateRegistryCommand(),
		getCreateArtifactCommand(),
		getCreateApplicationCommand(),
		getCreateProfileCommand(),
		getCreateDeploymentPackageCommand(),
		getCreateDeploymentProfileCommand(),
		getCreateApplicationReferenceCommand(),
		getCreateNetworkCommand(),

		getCreateDeploymentCommand(),

		getCreateClusterCommand(),
	)
	return cmd
}

func getListCommand() *cobra.Command {
	catalogListRootCmd := &cobra.Command{
		Use:               "list",
		Aliases:           []string{"ls", "show"},
		Short:             "List various catalog service entities",
		PersistentPreRunE: auth.CheckAuth,
	}
	catalogListRootCmd.AddCommand(
		getListRegistriesCommand(),
		getListArtifactsCommand(),
		getListApplicationsCommand(),
		getListProfilesCommand(),
		getListDeploymentPackagesCommand(),
		getListDeploymentProfilesCommand(),
		getListNetworksCommand(),

		getListDeploymentsCommand(),

		getListClusterTemplatesCommand(),
	)
	return catalogListRootCmd
}

func getGetCommand() *cobra.Command {
	catalogGetRootCmd := &cobra.Command{
		Use:               "get",
		Short:             "Get various catalog service entities",
		PersistentPreRunE: auth.CheckAuth,
	}
	catalogGetRootCmd.AddCommand(
		getGetRegistryCommand(),
		getGetArtifactCommand(),
		getGetApplicationCommand(),
		getGetProfileCommand(),
		getGetDeploymentPackageCommand(),
		getGetDeploymentProfileCommand(),
		getGetNetworkCommand(),

		getGetDeploymentCommand(),

		// Add plurals here for consistency with kubectl
		getListRegistriesCommand(),
		getListArtifactsCommand(),
		getListApplicationsCommand(),
		getListProfilesCommand(),
		getListDeploymentPackagesCommand(),
		getListDeploymentProfilesCommand(),

		getListChartsCommand(),

		getListDeploymentsCommand(),

		getListNetworksCommand(),
	)
	return catalogGetRootCmd
}

func getSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "set",
		Aliases:           []string{"update"},
		Short:             "Create various catalog service entities",
		PersistentPreRunE: auth.CheckAuth,
	}
	cmd.AddCommand(
		getSetRegistryCommand(),
		getSetArtifactCommand(),
		getSetApplicationCommand(),
		getSetProfileCommand(),
		getSetDeploymentPackageCommand(),
		getSetDeploymentProfileCommand(),

		getSetDeploymentCommand(),

		getSetNetworkCommand(),
	)
	return cmd
}

func getDeleteCommand() *cobra.Command {
	catalogDeleteRootCmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete various catalog service entities",
		PersistentPreRunE: auth.CheckAuth,
	}
	catalogDeleteRootCmd.AddCommand(
		getDeleteRegistryCommand(),
		getDeleteArtifactCommand(),
		getDeleteApplicationCommand(),
		getDeleteProfileCommand(),
		getDeleteDeploymentPackageCommand(),
		getDeleteDeploymentProfileCommand(),
		getDeleteApplicationReferenceCommand(),

		getDeleteDeploymentCommand(),

		getDeleteNetworkCommand(),
	)
	return catalogDeleteRootCmd
}

func getWatchCommand() *cobra.Command {
	catalogWatchCmd := &cobra.Command{
		Use:               "watch {registries|artifacts|applications|packages}...",
		Short:             "Watch updates of various catalog service entities",
		PersistentPreRunE: auth.CheckAuth,
		Args:              cobra.MinimumNArgs(1),
		RunE:              runWatchAllCommand,
	}
	return catalogWatchCmd
}

var kindAliases = map[string]string{
	"registries":   "Registry",
	"artifacts":    "Artifact",
	"applications": "Application",
	"apps":         "Application",
	"packages":     "DeploymentPackage",
	"bundles":      "DeploymentPackage",
}

func runWatchAllCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 1 && args[0] == "all" {
		return runWatchCommand(cmd, printAllEvent,
			"Registry", "Artifact", "Application", "DeploymentPackage")
	}

	kinds := make([]string, 0, len(args))
	for _, arg := range args {
		kind, ok := kindAliases[arg]
		if !ok {
			return errors.NewInvalid("Unsupported kind: %s", arg)
		}
		kinds = append(kinds, kind)
	}
	return runWatchCommand(cmd, printAllEvent, kinds...)
}

func printAllEvent(writer io.Writer, kind string, payload []byte, verbose bool) error {
	switch kind {
	case "Registry":
		return printRegistryEvent(writer, kind, payload, verbose)
	case "Artifact":
		return printArtifactEvent(writer, kind, payload, verbose)
	case "Application":
		return printApplicationEvent(writer, kind, payload, verbose)
	case "DeploymentPackage":
		return printDeploymentPackageEvent(writer, kind, payload, verbose)
	}
	return nil
}

func shortenUUID(uuid string) string {
	f := strings.SplitN(uuid, "-", 5)
	if len(f) == 5 {
		return f[4]
	}
	return uuid
}
