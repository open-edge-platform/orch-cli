// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/spf13/cobra"
)

func getCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "create",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Create various orchestrator service entities",
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

		getCreateOSUpdatePolicyCommand(),
		getCreateAmtProfileCommand(),
		getCreateCustomConfigCommand(),
		getCreateRegionCommand(),
		getCreateSiteCommand(),
		getCreateHostCommand(),
		getCreateOSProfileCommand(),
		getCreateProviderCommand(),
		getCreateSSHKeyCommand(),
	)
	return cmd
}

func getListCommand() *cobra.Command {
	catalogListRootCmd := &cobra.Command{
		Use:               "list",
		Aliases:           []string{"ls", "show"},
		Short:             "List various orchestrator service entities",
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
		getListChartsCommand(),
		getListDeploymentsCommand(),
		getListClusterCommand(),
		getListClusterTemplatesCommand(),

		getListOSUpdateRunCommand(),
		getListOSUpdatePolicyCommand(),
		getListAmtProfileCommand(),
		getListCustomConfigCommand(),
		getListSiteCommand(),
		getListRegionCommand(),
		getListOSProfileCommand(),
		getListHostCommand(),
		getListProviderCommand(),
		getListSSHKeyCommand(),
	)
	return catalogListRootCmd
}

func getGetCommand() *cobra.Command {
	catalogGetRootCmd := &cobra.Command{
		Use:               "get",
		Short:             "Get various orchestrator service entities",
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
		getGetClusterCommand(),

		getGetOSUpdateRunCommand(),
		getGetOSUpdatePolicyCommand(),
		getGetAmtProfileCommand(),
		getGetCustomConfigCommand(),
		getGetOSProfileCommand(),
		getGetRegionCommand(),
		getGetSiteCommand(),
		getGetHostCommand(),
		getGetProviderCommand(),
		getGetSSHKeyCommand(),
	)
	return catalogGetRootCmd
}

func getSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "set",
		Aliases:           []string{"update"},
		Short:             "Update various orchestrator service entities",
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

		getSetHostCommand(),
	)
	return cmd
}

func getDeleteCommand() *cobra.Command {
	catalogDeleteRootCmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete various orchestrator service entities",
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
		getDeleteClusterCommand(),
		getDeleteNetworkCommand(),

		getDeleteOSUpdateRunCommand(),
		getDeleteOSUpdatePolicyCommand(),
		getDeleteAmtProfileCommand(),
		getDeleteCustomConfigCommand(),
		getDeleteRegionCommand(),
		getDeleteSiteCommand(),
		getDeleteOSProfileCommand(),
		getDeleteHostCommand(),
		getDeleteProviderCommand(),
		getDeleteSSHKeyCommand(),
	)
	return catalogDeleteRootCmd
}
