// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) createProfile(pubName string, applicationName string, applicationVersion string, profileName string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`create profile --project %s %s %s %s`, pubName, applicationName, applicationVersion, profileName))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) listProfiles(pubName string, applicationName string, applicationVersion string, verbose bool) (string, error) {
	args := fmt.Sprintf(`list profiles --project %s %s %s`, pubName, applicationName, applicationVersion)
	if verbose {
		args = args + " -v"
	}
	getCmdOutput, err := s.runCommand(args)
	return getCmdOutput, err
}

func (s *CLITestSuite) getProfile(pubName string, applicationName, applicationVersion, profileName string) (string, error) {
	getCmdOutput, err := s.runCommand(fmt.Sprintf(`get profile --project %s %s %s %s`, pubName, applicationName, applicationVersion, profileName))
	return getCmdOutput, err
}

func (s *CLITestSuite) deleteProfile(pubName string, applicationName, applicationVersion, profileName string) error {
	_, err := s.runCommand(fmt.Sprintf(`delete profile --project %s %s %s %s`, pubName, applicationName, applicationVersion, profileName))
	return err
}

func (s *CLITestSuite) updateProfile(pubName string, applicationName, applicationVersion, profileName string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`set profile --project %s %s %s %s`, pubName, applicationName, applicationVersion, profileName))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) TestProfile() {
	const (
		pubName            = "testpub"
		profileName        = "new-profile"
		valueFile          = "testdata/values.yaml"
		profileDisplayName = "profile.display.name"
		profileDescription = "Profile.Description"

		registryName       = "myreg"
		chartName          = "chart-name"
		chartVersion       = "22.33.44"
		applicationVersion = "1.2.3"
		applicationName    = "myapp"
	)

	// create a registry
	createRegArgs := map[string]string{
		"root-url": "http://1.2.3.4",
	}
	err := s.createRegistry(pubName, registryName, createRegArgs)
	s.NoError(err)

	// create an application
	createAppArgs := map[string]string{
		"chart-name":     chartName,
		"chart-registry": registryName,
		"chart-version":  chartVersion,
	}
	err = s.createApplication(pubName, applicationName, applicationVersion, createAppArgs)
	s.NoError(err)

	// create a profile for the new publisher
	createArgs := map[string]string{
		"chart-values": valueFile,
		"display-name": profileDisplayName,
		"description":  profileDescription,
	}

	err = s.createProfile(pubName, applicationName, applicationVersion, profileName, createArgs)
	s.NoError(err)

	// list artifacts to make sure it was created properly
	listOutput, err := s.listProfiles(pubName, applicationName, applicationVersion, simpleOutput)
	s.NoError(err)

	parsedOutput := mapCliOutput(listOutput)
	expectedOutput := commandOutput{
		profileName: {
			"Name":         profileName,
			"Description":  profileDescription,
			"Display Name": profileDisplayName,
		},
	}
	s.compareOutput(expectedOutput, parsedOutput)

	// verbose list profiles
	listVerboseOutput, err := s.listProfiles(pubName, applicationName, applicationVersion, verboseOutput)
	s.NoError(err)

	parsedVerboseOutput := mapVerboseCliOutput(listVerboseOutput)
	expectedVerboseOutput := commandOutput{
		profileName: {
			"Name":         profileName,
			"Display Name": profileDisplayName,
			"Description":  profileDescription,
			"Create Time":  timestampRegex,
			"Update Time":  timestampRegex,
		},
	}
	s.compareOutput(expectedVerboseOutput, parsedVerboseOutput)

	// Update the profile
	updateArgs := map[string]string{
		"description": "new-description",
	}

	err = s.updateProfile(pubName, applicationName, applicationVersion, profileName, updateArgs)
	s.NoError(err)

	// check that the profile was updated
	getCmdOutput, err := s.getProfile(pubName, applicationName, applicationVersion, profileName)
	s.NoError(err)
	parsedGetOutput := mapCliOutput(getCmdOutput)
	expectedOutput[profileName]["Description"] = `new-description`
	s.compareOutput(expectedOutput, parsedGetOutput)

	// delete the profile
	err = s.deleteProfile(pubName, applicationName, applicationVersion, profileName)
	s.NoError(err)

	// Make sure profile is gone
	_, err = s.getProfile(pubName, applicationName, applicationVersion, profileName)
	s.Error(err)
	s.Contains(err.Error(), ` not found`)
}
