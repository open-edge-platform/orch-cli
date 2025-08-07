// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"strings"
	"testing"
)

func (s *CLITestSuite) createApplicationReference(project string, pkgName string,
	pkgVersion string, applicationName string, applicationVersion string) error {
	// create application-reference <deployment-package-name> <version> <application-name:version>
	commandString := addCommandArgs(
		commandArgs{},
		fmt.Sprintf(`create application-reference --project %s %s %s %s:%s`,
			project, pkgName,
			pkgVersion, applicationName, applicationVersion),
	)
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) deleteApplicationReference(project string, pkgName string,
	pkgVersion string, applicationName string) error {
	_, err := s.runCommand(fmt.Sprintf(`delete application-reference --project %s %s %s %s`, project, pkgName, pkgVersion, applicationName))
	return err
}

func (s *CLITestSuite) TestApplicationReference() {
	const (
		app1        = "app1"
		app1Version = "1.0"
		pubName     = "pubtest"
		pkgName     = "deployment-pkg"
		pkgVersion  = "1.0"
	)

	// create a test application and deployment package
	s.NoError(s.createTestRegistry(project))
	s.NoError(s.createTestApplication(project, app1))
	s.NoError(s.createTestDeploymentPackage(project, pkgName, pkgVersion, app1, app1Version))

	// add an app reference
	err := s.createApplicationReference(project, pkgName, pkgVersion, app1, "1.0")
	s.NoError(err)

	// verbose list deployment packages to make sure it was created properly
	listVerboseOutput, err := s.listDeploymentPackages(project, verboseOutput, "", "")
	s.NoError(err)

	parsedVerboseOutput := mapVerboseCliOutput(listVerboseOutput)
	expectedVerboseOutput := commandOutput{
		pkgName: {
			"Version":                  pkgVersion,
			"Create Time":              timestampRegex,
			"Update Time":              timestampRegex,
			"Name":                     pkgName,
			"Kind":                     "normal",
			"Display Name":             "deployment.package.display.name",
			"Description":              "",
			"Is Deployed":              "false",
			"Is Visible":               "true",
			"Applications":             `[app1:1.0 app2:1.0]`,
			"Application Dependencies": `[]`,
			"Profiles":                 ``,
			"Default Profile":          "",
			"Extensions":               "[]",
			"Artifacts":                "[]",
		},
	}

	s.compareOutput(expectedVerboseOutput, parsedVerboseOutput)

	// delete the application reference
	err = s.deleteApplicationReference(project, pkgName, pkgVersion, app1)
	s.NoError(err)

	// TODO not viable to mock
	// // Make sure application reference is gone
	// listVerboseAfterDeleteOutput, err := s.listDeploymentPackages(project, verboseOutput, "", "")
	// s.NoError(err)
	// parsedAfterDeleteOutput := mapVerboseCliOutput(listVerboseAfterDeleteOutput)
	// expectedVerboseOutput[pkgName]["Application Dependencies"] = `\[\]`
	// expectedVerboseOutput[pkgName]["Applications"] = `\[\]`
	// s.compareOutput(expectedVerboseOutput, parsedAfterDeleteOutput)
}

func FuzzApplicationReference(f *testing.F) {
	// Seed with valid and invalid input combinations
	f.Add("project", "deployment-pkg", "1.0", "app1", "1.0") // valid
	f.Add("", "deployment-pkg", "1.0", "app1", "1.0")        // missing project
	f.Add("project", "", "1.0", "app1", "1.0")               // missing pkgName
	f.Add("project", "deployment-pkg", "", "app1", "1.0")    // missing pkgVersion
	f.Add("project", "deployment-pkg", "1.0", "", "1.0")     // missing applicationName
	f.Add("project", "deployment-pkg", "1.0", "app1", "")    // missing applicationVersion

	f.Fuzz(func(t *testing.T, project, pkgName, pkgVersion, applicationName, applicationVersion string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		// --- Create Application Reference ---
		err := testSuite.createApplicationReference(project, pkgName, pkgVersion, applicationName, applicationVersion)
		if err != nil && (strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "accepts 3 arg(s), received 0") ||
			strings.Contains(err.Error(), "accepts 3 arg(s), received 1") ||
			strings.Contains(err.Error(), "accepts 3 arg(s), received 2") ||
			strings.Contains(err.Error(), "accepts 3 arg(s), received 4") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "application reference must be in the form name:version") ||
			strings.Contains(err.Error(), "unknown flag") ||
			strings.Contains(err.Error(), "no such file or directory")) {
			// Acceptable error for invalid reference
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid application reference create: %v", err)
		}

		// --- Delete Application Reference ---
		err = testSuite.deleteApplicationReference(project, pkgName, pkgVersion, applicationName)
		if err != nil && (strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "accepts 3 arg(s), received 0") ||
			strings.Contains(err.Error(), "accepts 3 arg(s), received 1") ||
			strings.Contains(err.Error(), "accepts 3 arg(s), received 2") ||
			strings.Contains(err.Error(), "accepts 3 arg(s), received 4") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "unknown flag") ||
			strings.Contains(err.Error(), "no such file or directory")) {
			// Acceptable error for invalid reference
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid application reference delete: %v", err)
		}
	})
}
