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

	//cmd.AddCommand(	// Core commands)

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

	//catalogListRootCmd.AddCommand()

	// App related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListRegistriesCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListArtifactsCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListApplicationsCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListProfilesCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListDeploymentPackagesCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListDeploymentProfilesCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListNetworksCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListChartsCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListDeploymentsCommand(), APP_ORCH_FEATURE)

	// Cluster related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListClusterTemplatesCommand(), CLUSTER_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListClusterCommand(), CLUSTER_ORCH_FEATURE)

	// Day2 related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListScheduleCommand(), DAY2_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListOSUpdateRunCommand(), DAY2_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListOSUpdatePolicyCommand(), DAY2_FEATURE)

	// Onboarding related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListHostCommand(), ONBOARDING_FEATURE)

	// Provisioning related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListOSProfileCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListCustomConfigCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListRegionCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListSiteCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListProviderCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListSSHKeyCommand(), PROVISIONING_FEATURE)

	// Out of Band Management related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListAmtProfileCommand(), OOB_FEATURE)

	// Multitenancy related commands
	addCommandIfFeatureEnabled(catalogListRootCmd, getListProjectCommand(), MULTITENANCY_FEATURE)
	addCommandIfFeatureEnabled(catalogListRootCmd, getListOrganizationCommand(), MULTITENANCY_FEATURE)

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
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetRegistryCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetArtifactCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetApplicationCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetProfileCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetDeploymentPackageCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetDeploymentProfileCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetNetworkCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetDeploymentCommand(), APP_ORCH_FEATURE)

	// Cluster related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetClusterCommand(), CLUSTER_ORCH_FEATURE)

	// Day2 related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetScheduleCommand(), DAY2_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetOSUpdateRunCommand(), DAY2_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetOSUpdatePolicyCommand(), DAY2_FEATURE)

	// Onboarding related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetHostCommand(), ONBOARDING_FEATURE)

	// Provisioning related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetOSProfileCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetCustomConfigCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetRegionCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetSiteCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetProviderCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetSSHKeyCommand(), PROVISIONING_FEATURE)

	// Out of Band Management related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetAmtProfileCommand(), OOB_FEATURE)

	// Multitenancy related commands
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetProjectCommand(), MULTITENANCY_FEATURE)
	addCommandIfFeatureEnabled(catalogGetRootCmd, getGetOrganizationCommand(), MULTITENANCY_FEATURE)

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
	addCommandIfFeatureEnabled(cmd, getSetRegistryCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getSetArtifactCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getSetApplicationCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getSetProfileCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getSetDeploymentPackageCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getSetDeploymentProfileCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getSetDeploymentCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(cmd, getSetNetworkCommand(), APP_ORCH_FEATURE)

	// Onboarding related commands
	addCommandIfFeatureEnabled(cmd, getSetHostCommand(), ONBOARDING_FEATURE)

	// Day2 related commands
	addCommandIfFeatureEnabled(cmd, getSetScheduleCommand(), DAY2_FEATURE)
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
	addCommandIfFeatureEnabled(cmd, getUpgradeDeploymentCommand(), APP_ORCH_FEATURE)
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
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteRegistryCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteArtifactCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteApplicationCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteProfileCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteDeploymentPackageCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteDeploymentProfileCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteApplicationReferenceCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteDeploymentCommand(), APP_ORCH_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteNetworkCommand(), APP_ORCH_FEATURE)

	// Cluster related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteClusterCommand(), CLUSTER_ORCH_FEATURE)

	// Day2 related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteScheduleCommand(), DAY2_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteOSUpdateRunCommand(), DAY2_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteOSUpdatePolicyCommand(), DAY2_FEATURE)

	// Onboarding related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteHostCommand(), ONBOARDING_FEATURE)

	// Provisioning related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteCustomConfigCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteRegionCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteSiteCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteOSProfileCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteProviderCommand(), PROVISIONING_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteSSHKeyCommand(), PROVISIONING_FEATURE)

	// Out of Band Management related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteAmtProfileCommand(), OOB_FEATURE)

	// Multitenancy related commands
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteProjectCommand(), MULTITENANCY_FEATURE)
	addCommandIfFeatureEnabled(catalogDeleteRootCmd, getDeleteOrganizationCommand(), MULTITENANCY_FEATURE)

	return catalogDeleteRootCmd
}
