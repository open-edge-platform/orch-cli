// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *CLITestSuite) createProfile(pubName string, applicationName string, applicationVersion string, profileName string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`create profile --project %s %s %s %s`, pubName, applicationName, applicationVersion, profileName))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) listProfiles(pubName string, applicationName string, applicationVersion string, verbose bool, outputFilter string, outputTemplate string, outputTemplateFile string) (string, error) {
	args := fmt.Sprintf(`list profiles --project %s %s %s`, pubName, applicationName, applicationVersion)
	if verbose {
		args = args + " -v"
	}
	if outputFilter != "" {
		args = args + " --output-filter " + outputFilter
	}
	if outputTemplate != "" {
		args = args + " --output-template " + outputTemplate
	}
	if outputTemplateFile != "" {
		args = args + " --output-template-file " + outputTemplateFile
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
	listOutput, err := s.listProfiles(pubName, applicationName, applicationVersion, simpleOutput, "", "", "")
	s.NoError(err)

	parsedOutput := mapCliOutput(listOutput)
	expectedOutput := commandOutput{
		profileName: {
			"NAME":         profileName,
			"DESCRIPTION":  profileDescription,
			"DISPLAY NAME": profileDisplayName,
		},
	}
	s.compareOutput(expectedOutput, parsedOutput)

	// verbose list profiles
	listVerboseOutput, err := s.listProfiles(pubName, applicationName, applicationVersion, verboseOutput, "", "", "")
	s.NoError(err)

	parsedVerboseOutput := mapVerboseCliOutput(listVerboseOutput)
	expectedVerboseOutput := commandOutput{
		profileName: {
			"Name":                    profileName,
			"Display Name":            profileDisplayName,
			"Description":             profileDescription,
			"Deployment Requirements": "requirement:1.2.3:Web server",
			"Create Time":             "2025-12-31T23:59:59",
			"Update Time":             "2025-12-31T23:59:59",
			"Parameter templates":     "Name: param1 Type: string Display Name: Parameter 1 Default: default-value Suggested values: value1,value2",
			"Chart Values":            "dmFsdWVzOiAxCnZhbDoy",
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

	// Test error handling for dual template flags (--output-template and --output-template-file both set)
	_, err = s.listProfiles(pubName, applicationName, applicationVersion, simpleOutput, "", "table{{.Name}}", "/tmp/invalid.tmpl")
	s.Error(err)
	s.Contains(err.Error(), "only one of")

	// Test error handling for missing template file
	_, err = s.listProfiles(pubName, applicationName, applicationVersion, simpleOutput, "", "", "/nonexistent/path/template.tmpl")
	s.Error(err)
	s.Contains(err.Error(), "unable to read")

	//Commenting out for now as mock wont support
	// Make sure profile is gone
	// _, err = s.getProfile(pubName, applicationName, applicationVersion, profileName)
	// s.Error(err)
	// s.Contains(err.Error(), ` not found`)
}

func FuzzProfile(f *testing.F) {
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
			"chart-values":       valueFile,
			"display-name":       displayName,
			"description":        description,
			"parameter-template": "name=type:name:value",
		}

		// Create profile
		err := testSuite.createProfile(pubName, applicationName, applicationVersion, profileName, createArgs)

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// Update profile
		updateArgs := map[string]string{
			"description": "new-description",
		}
		err = testSuite.updateProfile(pubName, applicationName, applicationVersion, profileName, updateArgs)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// Get profile
		_, err = testSuite.getProfile(pubName, applicationName, applicationVersion, profileName)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// Delete profile
		err = testSuite.deleteProfile(pubName, applicationName, applicationVersion, profileName)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// newCmdWithDeploymentRequirementFlag creates a minimal cobra.Command with the
// --deployment-requirement flag registered, for use in parseDeploymentRequirements tests.
func newCmdWithDeploymentRequirementFlag(values ...string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().StringSlice("deployment-requirement", values, "")
	return cmd
}

func TestParseDeploymentRequirements_NoFlag(t *testing.T) {
	cmd := newCmdWithDeploymentRequirementFlag()
	result, err := parseDeploymentRequirements(cmd)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestParseDeploymentRequirements_NameAndVersion(t *testing.T) {
	cmd := newCmdWithDeploymentRequirementFlag("cert-manager:0.2.1")
	result, err := parseDeploymentRequirements(cmd)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, *result, 1)
	assert.Equal(t, "cert-manager", (*result)[0].Name)
	assert.Equal(t, "0.2.1", (*result)[0].Version)
	assert.Nil(t, (*result)[0].DeploymentProfileName)
}

func TestParseDeploymentRequirements_NameVersionAndProfile(t *testing.T) {
	cmd := newCmdWithDeploymentRequirementFlag("cert-manager:0.2.1:default-profile")
	result, err := parseDeploymentRequirements(cmd)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, *result, 1)
	assert.Equal(t, "cert-manager", (*result)[0].Name)
	assert.Equal(t, "0.2.1", (*result)[0].Version)
	require.NotNil(t, (*result)[0].DeploymentProfileName)
	assert.Equal(t, "default-profile", *(*result)[0].DeploymentProfileName)
}

func TestParseDeploymentRequirements_ProfileNameEmpty(t *testing.T) {
	// Three parts but profile name is blank → DeploymentProfileName stays nil
	cmd := newCmdWithDeploymentRequirementFlag("cert-manager:0.2.1: ")
	result, err := parseDeploymentRequirements(cmd)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Nil(t, (*result)[0].DeploymentProfileName)
}

func TestParseDeploymentRequirements_MultipleRequirements(t *testing.T) {
	cmd := newCmdWithDeploymentRequirementFlag("pkg-a:1.0", "pkg-b:2.0:my-profile")
	result, err := parseDeploymentRequirements(cmd)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, *result, 2)
	assert.Equal(t, "pkg-a", (*result)[0].Name)
	assert.Equal(t, "pkg-b", (*result)[1].Name)
	assert.Equal(t, "my-profile", *(*result)[1].DeploymentProfileName)
}

func TestParseDeploymentRequirements_TooFewParts(t *testing.T) {
	cmd := newCmdWithDeploymentRequirementFlag("onlyone")
	_, err := parseDeploymentRequirements(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid deployment requirement format")
}

func TestParseDeploymentRequirements_TooManyParts(t *testing.T) {
	cmd := newCmdWithDeploymentRequirementFlag("a:b:c:d")
	_, err := parseDeploymentRequirements(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid deployment requirement format")
}

func TestParseDeploymentRequirements_EmptyPackageName(t *testing.T) {
	cmd := newCmdWithDeploymentRequirementFlag(":1.0")
	_, err := parseDeploymentRequirements(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package name and version cannot be empty")
}

func TestParseDeploymentRequirements_EmptyVersion(t *testing.T) {
	cmd := newCmdWithDeploymentRequirementFlag("pkg-a:")
	_, err := parseDeploymentRequirements(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package name and version cannot be empty")
}
