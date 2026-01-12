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
	featuresAliases          = []string{"feature", "features", "feat", "feats"}
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

	//cmd.AddCommand(	// Core commands)

	// App related commands
	addCommandIfFeatureEnabled(cmd, getCreateRegistryCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getCreateArtifactCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getCreateApplicationCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getCreateProfileCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getCreateDeploymentPackageCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getCreateDeploymentProfileCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getCreateApplicationReferenceCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getCreateNetworkCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getCreateDeploymentCommand(), AppOrchFeature)

	// Cluster related commands
	addCommandIfFeatureEnabled(cmd, getCreateClusterCommand(), ClusterOrchFeature)

	// Day2 related commands
	addCommandIfFeatureEnabled(cmd, getCreateOSUpdatePolicyCommand(), Day2Feature)
	addCommandIfFeatureEnabled(cmd, getCreateScheduleCommand(), Day2Feature)

	// Onboarding related commands
	addCommandIfFeatureEnabled(cmd, getCreateHostCommand(), OnboardingFeature)

	// Provisioning related commands
	addCommandIfFeatureEnabled(cmd, getCreateOSProfileCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(cmd, getCreateCustomConfigCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(cmd, getCreateRegionCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(cmd, getCreateSiteCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(cmd, getCreateProviderCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(cmd, getCreateSSHKeyCommand(), ProvisioningFeature)

	// Out of Band Management related commands
	addCommandIfFeatureEnabled(cmd, getCreateAmtProfileCommand(), OobFeature)

	// Multitenancy related commands
	addCommandIfFeatureEnabled(cmd, getCreateProjectCommand(), MultitenancyFeature)
	addCommandIfFeatureEnabled(cmd, getCreateOrganizationCommand(), MultitenancyFeature)
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
		getListFeaturesCommand(),
	)

	// App related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListRegistriesCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListArtifactsCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListApplicationsCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListProfilesCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListDeploymentPackagesCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListDeploymentProfilesCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListNetworksCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListChartsCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListDeploymentsCommand(), AppOrchFeature)

	// Cluster related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListClusterTemplatesCommand(), ClusterOrchFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListClusterCommand(), ClusterOrchFeature)

	// Day2 related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListScheduleCommand(), Day2Feature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListOSUpdateRunCommand(), Day2Feature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListOSUpdatePolicyCommand(), Day2Feature)

	// Onboarding related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListHostCommand(), OnboardingFeature)

	// Provisioning related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListOSProfileCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListCustomConfigCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListRegionCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListSiteCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListProviderCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListSSHKeyCommand(), ProvisioningFeature)

	// Out of Band Management related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListAmtProfileCommand(), OobFeature)

	// Multitenancy related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListProjectCommand(), MultitenancyFeature)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListOrganizationCommand(), MultitenancyFeature)

	return catalogListRootCmd
}

func getGetCommand() *cobra.Command {
	catalogGetRootCmd := &cobra.Command{
		Use:               "get",
		Short:             "Get various orchestrator service entities",
		PersistentPreRunE: auth.CheckAuth,
	}
	//catalogGetRootCmd.AddCommand()

	// App related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetRegistryCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetArtifactCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetApplicationCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetProfileCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetDeploymentPackageCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetDeploymentProfileCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetNetworkCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetDeploymentCommand(), AppOrchFeature)

	// Cluster related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetClusterCommand(), ClusterOrchFeature)

	// Day2 related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetScheduleCommand(), Day2Feature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetOSUpdateRunCommand(), Day2Feature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetOSUpdatePolicyCommand(), Day2Feature)

	// Onboarding related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetHostCommand(), OnboardingFeature)

	// Provisioning related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetOSProfileCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetCustomConfigCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetRegionCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetSiteCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetProviderCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetSSHKeyCommand(), ProvisioningFeature)

	// Out of Band Management related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetAmtProfileCommand(), OobFeature)

	// Multitenancy related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetProjectCommand(), MultitenancyFeature)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetOrganizationCommand(), MultitenancyFeature)

	return catalogGetRootCmd
}

func getSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "set",
		Aliases:           []string{"update"},
		Short:             "Update various orchestrator service entities",
		PersistentPreRunE: auth.CheckAuth,
	}
	//cmd.AddCommand()

	// App related commands
	addCommandIfFeatureEnabled(cmd, getSetRegistryCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getSetArtifactCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getSetApplicationCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getSetProfileCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getSetDeploymentPackageCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getSetDeploymentProfileCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getSetDeploymentCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(cmd, getSetNetworkCommand(), AppOrchFeature)

	// Onboarding related commands
	addCommandIfFeatureEnabled(cmd, getSetHostCommand(), OnboardingFeature)

	// Day2 related commands
	addCommandIfFeatureEnabled(cmd, getSetScheduleCommand(), Day2Feature)
	return cmd
}

func getUpgradeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "upgrade",
		Short:             "Upgrade deployment",
		PersistentPreRunE: auth.CheckAuth,
	}
	//cmd.AddCommand()
	// App related commands
	addCommandIfFeatureEnabled(cmd, getUpgradeDeploymentCommand(), AppOrchFeature)
	return cmd
}

func getDeleteCommand() *cobra.Command {
	catalogDeleteRootCmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete various orchestrator service entities",
		PersistentPreRunE: auth.CheckAuth,
	}
	//catalogDeleteRootCmd.AddCommand()

	// App related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteRegistryCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteArtifactCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteApplicationCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteProfileCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteDeploymentPackageCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteDeploymentProfileCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteApplicationReferenceCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteDeploymentCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteNetworkCommand(), AppOrchFeature)

	// Cluster related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteClusterCommand(), ClusterOrchFeature)

	// Day2 related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteScheduleCommand(), Day2Feature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteOSUpdateRunCommand(), Day2Feature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteOSUpdatePolicyCommand(), Day2Feature)

	// Onboarding related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteHostCommand(), OnboardingFeature)

	// Provisioning related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteCustomConfigCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteRegionCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteSiteCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteOSProfileCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteProviderCommand(), ProvisioningFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteSSHKeyCommand(), ProvisioningFeature)

	// Out of Band Management related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteAmtProfileCommand(), OobFeature)

	// Multitenancy related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteProjectCommand(), MultitenancyFeature)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteOrganizationCommand(), MultitenancyFeature)

	return catalogDeleteRootCmd
}
