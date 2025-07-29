// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) createDeploymentProfile(pubName string, pkgName string, pkgVersion string, pkgProfileName string, args commandArgs) error {
	commandString := addCommandArgs(args,
		fmt.Sprintf(`create deployment-package-profile --project %s %s %s %s`,
			pubName, pkgName, pkgVersion, pkgProfileName))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) listDeploymentProfiles(pubName string, pkgName string, pkgVersion string, verbose bool) (string, error) {
	args := fmt.Sprintf(`get deployment-package-profiles --project %s %s %s`,
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
		pkgProfileName               = "deployment-package-profile"
		deploymentProfileDisplayName = "deployment.profile.display.name"
		deploymentProfileDescription = "Profile.for.testing"
	)

	// create test application and a deployment package
	s.NoError(s.createTestRegistry(pubName))
	s.NoError(s.createTestApplication(pubName, app1))
	s.NoError(s.createTestDeploymentPackage(pubName, pkgName, pkgVersion, app1, pkgVersion))

	// create a deployment profile
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

	// check that the deployment profile was updated
	_, err = s.getDeploymentProfile(pubName, pkgName, pkgVersion, pkgProfileName)
	s.NoError(err)
	// TODOCommenting out not viable to mock at this moment
	// parsedGetOutput := mapCliOutput(getCmdOutput)
	// expectedOutput[pkgProfileName]["Display Name"] = `new.display-name`
	// s.compareOutput(expectedOutput, parsedGetOutput)

	// delete the deployment profile
	err = s.deleteDeploymentProfile(pubName, pkgName, pkgVersion, pkgProfileName)
	s.NoError(err)

	// /Commenting out fot now not viable to mock
	// // Make sure deployment profile is gone
	// _, err = s.getDeploymentProfile(pubName, pkgName, pkgVersion, pkgProfileName)
	// s.Error(err)
	// s.Contains(err.Error(), ` not found`)
}
