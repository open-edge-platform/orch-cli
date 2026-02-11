// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/spf13/viper"
)

func (s *CLITestSuite) TestNegative() {

	// Set viper flag BEFORE SetupTest/login
	viper.Set("test_orchestrator_features_disabled", true)
	defer viper.Set("test_orchestrator_features_disabled", false) // Clean up

	// Now call SetupTest which will trigger login with disabled features
	s.logout()
	err := s.login("u", "p")
	s.NoError(err)

	name := "schedule"
	rresourceID := "repeatedsche-abcd1234"
	version := "0.1.0"
	project := "disabled-features"
	SArgs := map[string]string{}

	/////////////////////////////////////////////////////////////////////
	// Test commands disabled by edge-infrastructure-manager.day2 feature
	/////////////////////////////////////////////////////////////////////
	feature := "schedule"
	featureAlias := "sch"
	expectedOutput := "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput := "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	//fail to create schedule with edge-infrastructure-manager.day2 disabled
	output, err := s.createSchedule(project, name, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	//fail to list schedules with edge-infrastructure-manager.day2 disabled

	output, err = s.listSchedule(project, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	//fail to get schedules with edge-infrastructure-manager.day2 disabled
	output, err = s.getSchedule(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	//fail to delete schedules with edge-infrastructure-manager.day2 disabled
	output, err = s.deleteSchedule(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//SET
	//fail to set schedule with edge-infrastructure-manager.day2 disabled
	output, err = s.setSchedule(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("set " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test osupdatepolicy (also Day2Feature)
	feature = "osupdatepolicy"
	featureAlias = "oup"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	output, err = s.createOSUpdatePolicy(project, "test.yaml", SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listOSUpdatePolicy(project, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getOSUpdatePolicy(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	output, err = s.deleteOSUpdatePolicy(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	/////////////////////////////////////////////////////////////////////
	// Test commands disabled by edge-infrastructure-manager.oob feature
	/////////////////////////////////////////////////////////////////////
	feature = "amtprofile"
	featureAlias = "amt"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	//fail to create amtprofile with edge-infrastructure-manager.oob disabled
	output, err = s.createAMT(project, name, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	//fail to list amtprofiles with edge-infrastructure-manager.oob disabled

	output, err = s.listAMT(project, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	//fail to get amtprofiles with edge-infrastructure-manager.oob disabled
	output, err = s.getAMT(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	//fail to delete amtprofiles with edge-infrastructure-manager.oob disabled
	output, err = s.deleteAMT(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	/////////////////////////////////////////////////////////////////////
	// Test commands disabled by edge-infrastructure-manager.provisioning feature
	/////////////////////////////////////////////////////////////////////
	feature = "site"
	featureAlias = "st"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	//fail to create site with edge-infrastructure-manager.provisioning disabled
	output, err = s.createSite(project, name, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	//fail to list sites with edge-infrastructure-manager.provisioning disabled

	output, err = s.listSite(project, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	//fail to get sites with edge-infrastructure-manager.provisioning disabled
	output, err = s.getSite(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	//fail to delete sites with edge-infrastructure-manager.provisioning disabled
	output, err = s.deleteSite(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test region (also ProvisioningFeature)
	feature = "region"
	featureAlias = "regn"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	output, err = s.createRegion(project, name, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listRegion(project, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getRegion(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	output, err = s.deleteRegion(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test osprofile (also ProvisioningFeature)
	feature = "osprofile"
	featureAlias = "osp"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	output, err = s.createOSProfile(project, "test.yaml", SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listOSProfile(project, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getOSProfile(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test customconfig (also ProvisioningFeature)
	feature = "customconfig"
	featureAlias = "cfg"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	output, err = s.createCustomConfig(project, name, "test.yaml", SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listCustomConfig(project, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getCustomConfig(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	output, err = s.deleteCustomConfig(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test provider (also ProvisioningFeature)
	feature = "provider"
	featureAlias = "prov"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	output, err = s.createProvider(project, name, "kind", "api", SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listProvider(project, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getProvider(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test sshkey (also ProvisioningFeature)
	feature = "sshkey"
	featureAlias = "ssh"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	output, err = s.createSSHKey(project, name, "test.pub", SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listSSHKey(project, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getSSHKey(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	/////////////////////////////////////////////////////////////////////
	// Test commands disabled by edge-infrastructure-manager.omx-profile feature
	/////////////////////////////////////////////////////////////////////
	output, err = s.runCommand("generate standalone-config --project " + project)
	s.Error(err) // Cobra still returns an error for unknown commands
	s.Contains(output, "Error: command \"generate\" is disabled in the current Edge Orchestrator configuration")

	/////////////////////////////////////////////////////////////////////
	// Test commands disabled by cluster-orchestration feature
	/////////////////////////////////////////////////////////////////////
	feature = "cluster"
	featureAlias = "cl"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	//fail to create cluster with cluster-orchestration disabled
	output, err = s.createCluster(project, name, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	//fail to list clusters with cluster-orchestration disabled

	output, err = s.listCluster(project, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	//fail to get clusters with cluster-orchestration disabled
	output, err = s.getCluster(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	//fail to delete clusters with cluster-orchestration disabled
	output, err = s.deleteCluster(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	/////////////////////////////////////////////////////////////////////
	// Test commands disabled by multitenancy feature
	/////////////////////////////////////////////////////////////////////
	feature = "organization"
	featureAlias = "org"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	//fail to create organization with multitenancy disabled
	output, err = s.createOrganization(project, name, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	//fail to list organizations with multitenancy disabled

	output, err = s.listOrganization(project, SArgs)
	s.NoError(err)
	s.Contains(output, "Error: command \"organizations\" is disabled in the current Edge Orchestrator configuration")

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	//fail to get organizations with multitenancy disabled
	output, err = s.getOrganization(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	//fail to delete organizations with multitenancy disabled
	output, err = s.deleteOrganization(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test project (also MultitenancyFeature)
	feature = "project"
	featureAlias = "proj"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	output, err = s.createProject(project, name, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("create " + featureAlias)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listProject(project, SArgs)
	s.NoError(err)
	s.Contains(output, "Error: command \"projects\" is disabled in the current Edge Orchestrator configuration")

	//using alias
	output, err = s.runCommand("list " + featureAlias)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getProject(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + rresourceID)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	output, err = s.deleteProject(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " " + rresourceID)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	/////////////////////////////////////////////////////////////////////
	// Test commands disabled by  feature application-orchestration
	/////////////////////////////////////////////////////////////////////
	feature = "application"
	featureAlias = "app"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	//fail to create application with application-orchestration disabled
	err = s.createApplication(project, name, version, SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	//fail to list application with application-orchestration disabled

	output, err = s.listApplications(project, false, "", "", "")
	s.NoError(err)
	s.Contains(output, "Error: command \"applications\" is disabled in the current Edge Orchestrator configuration")

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	//fail to get application with application-orchestration disabled
	output, err = s.getApplication(project, name, version)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	//fail to delete application with application-orchestration disabled
	err = s.deleteApplication(project, name, version)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test artifact (also AppOrchFeature)
	feature = "artifact"
	featureAlias = "art"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	err = s.createArtifact(project, name, SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listArtifacts(project, false, "", "")
	s.NoError(err)
	s.Contains(output, "Error: command \"artifacts\" is disabled in the current Edge Orchestrator configuration")

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getArtifact(project, rresourceID)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//SET
	err = s.updateArtifact(project, rresourceID, SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("set " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	err = s.deleteArtifact(project, rresourceID)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test profile (also AppOrchFeature)
	feature = "profile"
	featureAlias = "prof"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	err = s.createProfile(project, name, version, "profile1", SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listProfiles(project, name, version, false)
	s.NoError(err)
	s.Contains(output, "Error: command \"profiles\" is disabled in the current Edge Orchestrator configuration")

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getProfile(project, name, version, "profile1")
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + name + " " + version + " profile1 --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//SET
	err = s.updateProfile(project, name, version, "profile1", SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("set " + featureAlias + " " + name + " " + version + " profile1 --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	err = s.deleteProfile(project, name, version, "profile1")
	s.NoError(err)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " " + name + " " + version + " profile1 --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test deployment-package (also AppOrchFeature)
	feature = "deployment-package"
	featureAlias = "pkg"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	err = s.createDeploymentPackage(project, name, version, SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listDeploymentPackages(project, false, "", "")
	s.NoError(err)
	s.Contains(output, "Error: command \"deployment-packages\" is disabled in the current Edge Orchestrator configuration")

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getDeploymentPackage(project, name, version)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + name + " " + version + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//SET
	err = s.updateDeploymentPackage(project, name, version, SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("set " + featureAlias + " " + name + " " + version + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	err = s.deleteDeploymentPackage(project, name, version)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " " + name + " " + version + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test deployment-profile (also AppOrchFeature)
	feature = "deployment-profile"
	featureAlias = "package-profile"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	err = s.createDeploymentProfile(project, name, version, "profile1", SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listDeploymentProfiles(project, name, version, false)
	s.NoError(err)
	s.Contains(output, "Error: command \"deployment-package-profiles\" is disabled in the current Edge Orchestrator configuration")

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getDeploymentProfile(project, name, version, "profile1")
	s.NoError(err)
	s.Contains(output, "Error: command \"deployment-package-profile\" is disabled in the current Edge Orchestrator configuration")

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + name + " " + version + " profile1 --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//SET
	err = s.updateDeploymentProfile(project, name, version, "profile1", SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("set " + featureAlias + " " + name + " " + version + " profile1 --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	err = s.deleteDeploymentProfile(project, name, version, "profile1")
	s.NoError(err)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " " + name + " " + version + " profile1 --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test application-reference (also AppOrchFeature)
	feature = "application-reference"
	featureAlias = "app-reference"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	err = s.createApplicationReference(project, name, version, name, version)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	err = s.deleteApplicationReference(project, name, version, name)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " " + name + " " + version + " " + name + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test chart (also AppOrchFeature - list only)
	feature = "charts"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"

	//LIST
	output, err = s.listCharts(project, "registry1", SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	// Test registry (also AppOrchFeature)
	feature = "registry"
	featureAlias = "reg"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	err = s.createRegistry(project, name, SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listRegistries(project, false, false, "", "")
	s.NoError(err)
	s.Contains(output, "Error: command \"registries\" is disabled in the current Edge Orchestrator configuration")

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getRegistry(project, rresourceID)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//SET
	err = s.updateRegistry(project, rresourceID, SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("set " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	err = s.deleteRegistry(project, rresourceID)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test deployment (also AppOrchFeature)
	feature = "deployment"
	featureAlias = "dep"
	expectedOutput = "Error: command \"" + feature + "\" is disabled in the current Edge Orchestrator configuration"
	expectedAliasOutput = "Error: command \"" + featureAlias + "\" is disabled in the current Edge Orchestrator configuration"

	//CREATE
	err = s.createDeployment(name, version, SArgs)
	s.NoError(err)

	//using alias
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//LIST
	output, err = s.listDeployment(project, SArgs)
	s.NoError(err)
	s.Contains(output, "Error: command \"deployments\" is disabled in the current Edge Orchestrator configuration")

	//using alias
	output, err = s.runCommand("list " + featureAlias + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//GET
	output, err = s.getDeployment(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("get " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//SET
	output, err = s.setDeployment(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("set " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	//DELETE
	output, err = s.deleteDeployment(project, rresourceID, SArgs)
	s.NoError(err)
	s.Contains(output, expectedOutput)

	//using alias
	output, err = s.runCommand("delete " + featureAlias + " " + rresourceID + " --project " + project)
	s.NoError(err)
	s.Contains(output, expectedAliasOutput)

	// Test top-level commands (also AppOrchFeature)
	output, err = s.runCommand("upgrade deployment " + rresourceID + " --project " + project)
	s.Error(err)
	s.Contains(output, "Error: command \"upgrade\" is disabled in the current Edge Orchestrator configuration")

	output, err = s.runCommand("import helm-chart oci:/path/to/chart:1.0.0 --project " + project)
	s.Error(err)
	s.Contains(output, "Error: command \"import\" is disabled in the current Edge Orchestrator configuration")

	output, err = s.runCommand("export deployment-package wordpress 0.1.1 --project " + project)
	s.Error(err)
	s.Contains(output, "Error: command \"export\" is disabled in the current Edge Orchestrator configuration")

	/////////////////////////////////////////////////////////////////////
	// Test non existing command
	/////////////////////////////////////////////////////////////////////
	featureAlias = "nonsense"

	//fail with non-existing sub command
	output, err = s.runCommand("create " + featureAlias + " --project " + project)
	s.NoError(err)

	expectedOutput = "Error: unknown command \"" + featureAlias + "\""

	s.Contains(output, expectedOutput)

	//fail with non-existing top command
	output, err = s.runCommand(featureAlias + " --project " + project)
	s.ErrorContains(err, "unknown command")
}

func (s *CLITestSuite) TestNegativeLegacyOrch() {
	// Set viper flag to simulate 404 response from orchestrator info endpoint
	viper.Set("test_orchestrator_404", true)
	defer viper.Set("test_orchestrator_404", false) // Clean up

	// Logout and attempt login
	s.logout()
	err := s.login("u", "p")

	// Expect login to return an error about orchestrator info not being available
	s.Error(err)
	s.Contains(err.Error(), "the Edge Orchestrator Component Status service info not available")
	s.Contains(err.Error(), "setting relevant features to enabled by default for backward compatibility")

	// Verify all feature flags are set to true (default/safe state)
	s.True(viper.GetBool(OobFeature), "OobFeature should be true")
	s.True(viper.GetBool(OnboardingFeature), "OnboardingFeature should be true")
	s.True(viper.GetBool(ProvisioningFeature), "ProvisioningFeature should be true")
	s.True(viper.GetBool(OxmFeature), "OxmFeature should be true")
	s.True(viper.GetBool(Day2Feature), "Day2Feature should be true")
	s.True(viper.GetBool(AppOrchFeature), "AppOrchFeature should be true")
	s.True(viper.GetBool(ClusterOrchFeature), "ClusterOrchFeature should be true")
	s.True(viper.GetBool(ObservabilityFeature), "ObservabilityFeature should be true")
	s.True(viper.GetBool(MultitenancyFeature), "MultitenancyFeature should be true")
	s.True(viper.GetBool(EIMFeature), "EIMFeature should be true")
}
