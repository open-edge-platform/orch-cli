// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/spf13/cobra"
)

var (
	amtAliases               = []string{"amtprofile", "amtprofiles", "amt", "amts"}
	applicationAliases       = []string{"application", "applications", "app", "apps"}
	artifactAliases          = []string{"artifact", "artifacts", "art", "arts"}
	chartAliases             = []string{"chart", "charts"}
	clusterAliases           = []string{"cluster", "clusters", "cl", "cls"}
	clusterTemplateAliases   = []string{"clustertemplate", "clustertemplates", "template", "templates", "ctmpl", "ctmps"}
	customConfigAliases      = []string{"customconfig", "customconfigs", "cfg", "cfgs"}
	deploymentPackageAliases = []string{"deployment-package", "deployment-packages", "package", "packages", "bundle", "bundles", "pkg", "pkgs"}
	deploymentProfileAliases = []string{"deployment-package-profile", "deployment-package-profiles", "deployment-profile", "deployment-profiles", "package-profile", "bundle-profile"}
	deploymentAliases        = []string{"deployment", "deployments", "dep", "deps"}
	hostAliases              = []string{"host", "hosts", "hs"}
	networkAliases           = []string{"network", "networks", "net", "nets"}
	osProfileAliases         = []string{"osprofile", "osprofiles", "osp", "osps"}
	organizationAliases      = []string{"organization", "organizations", "org", "orgs"}
	osUpdatePolicyAliases    = []string{"osupdatepolicy", "osupdatepolicies", "oup", "oups"}
	osUpdateRunAliases       = []string{"osupdaterun", "osupdateruns", "our", "ours"}
	providerAliases          = []string{"provider", "providers", "prov", "provs"}
	profileAliases           = []string{"profile", "profiles", "prof", "profs"}
	projectAliases           = []string{"project", "projects", "proj", "projs"}
	registryAliases          = []string{"registry", "registries", "reg", "regs"}
	regionAliases            = []string{"region", "regions", "regn", "regns"}
	siteAliases              = []string{"site", "sites", "st", "sts"}
	scheduleAliases          = []string{"schedule", "schedules", "sch", "schs"}
	sshKeyAliases            = []string{"sshkey", "sshkeys", "ssh", "sshs"}
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
		getCreateScheduleCommand(),
		getCreateProjectCommand(),
		getCreateOrganizationCommand(),
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
		getListScheduleCommand(),
		getListProjectCommand(),
		getListOrganizationCommand(),
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
		getGetScheduleCommand(),
		getGetProjectCommand(),
		getGetOrganizationCommand(),
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
		getSetScheduleCommand(),
	)
	return cmd
}

func getUpgradeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "upgrade",
		Short:             "Upgrade deployment",
		PersistentPreRunE: auth.CheckAuth,
	}
	cmd.AddCommand(
		getUpgradeDeploymentCommand(),
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
		getDeleteScheduleCommand(),
		getDeleteProjectCommand(),
		getDeleteOrganizationCommand(),
	)
	return catalogDeleteRootCmd
}
