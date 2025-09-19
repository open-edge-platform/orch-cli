// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) createDeploymentProfile(pubName string, pkgName string, pkgVersion string, pkgProfileName string, args commandArgs) error {
	commandString := addCommandArgs(args,
		fmt.Sprintf(`create deployment-package-profile --project %s %s %s %s`,
			pubName, pkgName, pkgVersion, pkgProfileName))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) listDeploymentProfiles(pubName string, pkgName string, pkgVersion string, verbose bool) (string, error) {
	args := fmt.Sprintf(`list deployment-package-profiles --project %s %s %s`,
		pubName, pkgName, pkgVersion)
	if verbose {
		args = args + " -v"
	}
	getCmdOutput, err := s.runCommand(args)
	return getCmdOutput, err
}

func (s *CLITestSuite) getDeploymentProfile(pubName string, pkgName string, pkgVersion string, pkgProfileName string) (string, error) {
	getCmdOutput, err := s.runCommand(fmt.Sprintf(`get deployment-package-profile --project %s %s %s %s`, pubName, pkgName, pkgVersion, pkgProfileName))
	return getCmdOutput, err
}

func (s *CLITestSuite) deleteDeploymentProfile(pubName string, pkgName string, pkgVersion string, pkgProfileName string) error {
	_, err := s.runCommand(fmt.Sprintf(`delete deployment-package-profile --project %s %s %s %s`, pubName, pkgName, pkgVersion, pkgProfileName))
	return err
}

func (s *CLITestSuite) updateDeploymentProfile(pubName string, pkgName string, pkgVersion string, pkgProfileName string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`set deployment-package-profile --project %s %s %s %s`, pubName, pkgName, pkgVersion, pkgProfileName))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) TestDeploymentProfile() {
	const (
		app1                         = "app1"
		pubName                      = "pubtest"
		pkgName                      = "deployment-pkg"
		pkgVersion                   = "1.0"
		pkgProfileName               = "new-test-deployment-profile"
		deploymentProfileDisplayName = "test.deployment.profile.display.name"
		deploymentProfileDescription = "Test.Profile.for.testing"
	)

	// create test application and a deployment package
	s.NoError(s.createTestRegistry(pubName))
	s.NoError(s.createTestApplication(pubName, app1))
	s.NoError(s.createTestDeploymentPackage(pubName, pkgName, pkgVersion, app1, pkgVersion))

	// Test against existing deployment profile in mock data, then create a new one
	createArgs := map[string]string{
		"display-name": deploymentProfileDisplayName,
		"description":  deploymentProfileDescription,
	}
	err := s.createDeploymentProfile(pubName, pkgName, pkgVersion, pkgProfileName, createArgs)
	s.NoError(err)

	// list deployment profiles to make sure it was created properly
	listOutput, err := s.listDeploymentProfiles(pubName, pkgName, pkgVersion, simpleOutput)
	s.NoError(err)

	parsedOutput := mapCliOutput(listOutput)
	expectedOutput := commandOutput{
		pkgProfileName: {
			"Name":          pkgProfileName,
			"Display Name":  deploymentProfileDisplayName,
			"Description":   deploymentProfileDescription,
			"Profile Count": "0",
		},
	}

	s.compareOutput(expectedOutput, parsedOutput)

	// verbose list deployment profiles
	listVerboseOutput, err := s.listDeploymentProfiles(pubName, pkgName, pkgVersion, verboseOutput)
	s.NoError(err)

	parsedVerboseOutput := mapVerboseCliOutput(listVerboseOutput)
	expectedVerboseOutput := commandOutput{
		pkgProfileName: {
			"Create Time":  timestampRegex,
			"Update Time":  timestampRegex,
			"Name":         pkgProfileName,
			"Display Name": deploymentProfileDisplayName,
			"Description":  deploymentProfileDescription,
			"Profiles":     "map[]",
		},
	}

	s.compareOutput(expectedVerboseOutput, parsedVerboseOutput)

	// Update the deployment profile
	updateArgs := map[string]string{
		"display-name": "new.display-name",
	}
	err = s.updateDeploymentProfile(pubName, pkgName, pkgVersion, pkgProfileName, updateArgs)
	s.NoError(err)

	// check that the deployment profile exists
	_, err = s.getDeploymentProfile(pubName, pkgName, pkgVersion, pkgProfileName)
	s.NoError(err)

	// delete the deployment profile
	err = s.deleteDeploymentProfile(pubName, pkgName, pkgVersion, pkgProfileName)
	s.NoError(err)
}

func FuzzDeploymentProfile(f *testing.F) {
	// Seed with valid and invalid input combinations
	f.Add("pubtest", "deployment-pkg", "1.0", "fuzz-test-deployment-profile", "display.name", "desc")
	f.Add("", "deployment-pkg", "1.0", "fuzz-test-deployment-profile", "display.name", "desc")     // missing pubName
	f.Add("pubtest", "", "1.0", "fuzz-test-deployment-profile", "display.name", "desc")            // missing pkgName
	f.Add("pubtest", "deployment-pkg", "", "fuzz-test-deployment-profile", "display.name", "desc") // missing pkgVersion
	f.Add("pubtest", "deployment-pkg", "1.0", "", "display.name", "desc")                          // missing pkgProfileName

	f.Fuzz(func(t *testing.T, pubName, pkgName, pkgVersion, pkgProfileName, displayName, description string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		createArgs := map[string]string{
			"display-name": displayName,
			"description":  description,
		}

		// --- Create Deployment Profile ---
		err := testSuite.createDeploymentProfile(pubName, pkgName, pkgVersion, pkgProfileName, createArgs)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List Deployment Profiles ---
		_, err = testSuite.listDeploymentProfiles(pubName, pkgName, pkgVersion, false)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Get Deployment Profile ---
		_, err = testSuite.getDeploymentProfile(pubName, pkgName, pkgVersion, pkgProfileName)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Update Deployment Profile ---
		updateArgs := map[string]string{
			"display-name": "new.display.name",
		}
		err = testSuite.updateDeploymentProfile(pubName, pkgName, pkgVersion, pkgProfileName, updateArgs)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete Deployment Profile ---
		err = testSuite.deleteDeploymentProfile(pubName, pkgName, pkgVersion, pkgProfileName)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
