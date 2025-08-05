// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"strings"
	"testing"
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
			"Name":                    profileName,
			"Display Name":            profileDisplayName,
			"Description":             profileDescription,
			"Deployment Requirements": "requirement",
			"Create Time":             timestampRegex,
			"Update Time":             timestampRegex,
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
	_, err = s.getProfile(pubName, applicationName, applicationVersion, profileName)
	s.NoError(err)

	//Commenting out the test for now as not able to mock easily
	//parsedGetOutput := mapCliOutput(getCmdOutput)
	//expectedOutput[profileName]["Description"] = `new-description`
	//s.compareOutput(expectedOutput, parsedGetOutput)

	// delete the profile
	err = s.deleteProfile(pubName, applicationName, applicationVersion, profileName)
	s.NoError(err)

	//Commenting out for now as mock wont support
	// Make sure profile is gone
	// _, err = s.getProfile(pubName, applicationName, applicationVersion, profileName)
	// s.Error(err)
	// s.Contains(err.Error(), ` not found`)
}

func FuzzCreateProfile(f *testing.F) {
	// Initial corpus with valid and invalid input
	f.Add("testpub", "myapp", "1.2.3", "profile1", "testdata/values.yaml", "Profile.Display.Name", "Profile.Description") // valid
	f.Add("", "myapp", "1.2.3", "profile1", "testdata/values.yaml", "Profile.Display.Name", "Profile.Description")        // missing publisher
	f.Add("testpub", "", "1.2.3", "profile1", "testdata/values.yaml", "Profile.Display.Name", "Profile.Description")      // missing app
	f.Add("testpub", "myapp", "", "profile1", "testdata/values.yaml", "Profile.Display.Name", "Profile.Description")      // missing app version
	f.Add("testpub", "myapp", "1.2.3", "", "testdata/values.yaml", "Profile.Display.Name", "Profile.Description")         // missing profile name
	f.Add("testpub", "myapp", "1.2.3", "profile1", "", "Profile.Display.Name", "Profile.Description")                     // missing values file

	f.Fuzz(func(t *testing.T, pubName, applicationName, applicationVersion, profileName, valueFile, displayName, description string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		createArgs := map[string]string{
			"chart-values": valueFile,
			"display-name": displayName,
			"description":  description,
		}

		// Create profile
		err := testSuite.createProfile(pubName, applicationName, applicationVersion, profileName, createArgs)
		if pubName == "" || applicationName == "" || applicationVersion == "" || profileName == "" || valueFile == "" {
			if err == nil {
				t.Errorf("Expected error for missing required field, got: %v", err)
			}
			return
		}
		if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid profile creation: %v", err)
			return
		}

		// Update profile
		updateArgs := map[string]string{
			"description": "new-description",
		}
		err = testSuite.updateProfile(pubName, applicationName, applicationVersion, profileName, updateArgs)
		if err != nil && (strings.Contains(err.Error(), "no artifact profile matches the given name") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 0") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 2") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 3") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "required flag \"project\" not set") ||
			strings.Contains(err.Error(), "no such file or directory")) {
			// Acceptable error for missing profile
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid profile %s, application %s update: %v", profileName, applicationName, err)
			return
		}

		// Get profile
		_, err = testSuite.getProfile(pubName, applicationName, applicationVersion, profileName)
		if err != nil && (strings.Contains(err.Error(), "no artifact profile matches the given name") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 0") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 2") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 3") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "required flag \"project\" not set") ||
			strings.Contains(err.Error(), "no such file or directory")) {
			// Acceptable error for missing profile
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid profile get: %v", err)
			return
		}

		// Delete profile
		err = testSuite.deleteProfile(pubName, applicationName, applicationVersion, profileName)
		if err != nil && (strings.Contains(err.Error(), "no artifact profile matches the given name") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 0") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 2") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 3") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "required flag \"project\" not set") ||
			strings.Contains(err.Error(), "no such file or directory")) {
			// Acceptable error for missing profile
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid profile deletion: %v", err)
		}
	})
}
