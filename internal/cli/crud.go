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
	// Core commands
	)
	// App related commands
	addCommandIfFeatureEnabled(cmd, getCreateRegistryCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateArtifactCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateApplicationCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateProfileCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateDeploymentPackageCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateDeploymentProfileCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateApplicationReferenceCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateNetworkCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateDeploymentCommand(), APP_ORCH_FEATURE)

	// Cluster related commands
	addCommandIfFeatureEnabled(cmd, getCreateClusterCommand(), CLUSTER_ORCH_FEATURE)

	// Day2 related commands
	addCommandIfFeatureEnabled(cmd, getCreateOSUpdatePolicyCommand(), DAY2_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateScheduleCommand(), DAY2_FEATURE)

	// Onboarding related commands
	addCommandIfFeatureEnabled(cmd, getCreateHostCommand(), ONBOARDING_FEATURE)

	// Provisioning related commands
	addCommandIfFeatureEnabled(cmd, getCreateOSProfileCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateCustomConfigCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateRegionCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateSiteCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateProviderCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateSSHKeyCommand(), PROVISIONING_FEATURE)

	// Out of Band Management related commands
	addCommandIfFeatureEnabled(cmd, getCreateAmtProfileCommand(), OOB_FEATURE)

	// Multitenancy related commands
	addCommandIfFeatureEnabled(cmd, getCreateProjectCommand(), MULTITENANCY_FEATURE)
	addCommandIfFeatureEnabled(cmd, getCreateOrganizationCommand(), MULTITENANCY_FEATURE)
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
	addCommandIfFeatureEnabled(catalogListRootCmd, getListAmtProfileCommand(), OOB_FEATURE)
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
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetAmtProfileCommand(), OOB_FEATURE)
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
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteAmtProfileCommand(), OOB_FEATURE)
	return catalogDeleteRootCmd
}
