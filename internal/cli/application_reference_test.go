// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
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
		app1Version = "1.0.0"
		pubName     = "pubtest"
		pkgName     = "deployment-pkg"
		pkgVersion  = "1.0.0"
	)

	// create a test application and deployment package
	s.NoError(s.createTestRegistry(project))
	s.NoError(s.createTestApplication(project, app1))

	// Create app2 as a placeholder to create the deployment package
	// (deployment packages need at least one app reference)
	app2 := "app2"
	s.NoError(s.createTestApplication(project, app2))
	s.NoError(s.createTestDeploymentPackage(project, pkgName, pkgVersion, app2, app1Version))

	// add app1 reference to test the create application-reference command
	err := s.createApplicationReference(project, pkgName, pkgVersion, app1, "1.0.0")
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
			"Applications":             `[app1:1.0.0 app2:1.0.0]`,
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
	f.Add("project", "deployment-pkg", "1.0.0", "app1", "1.0.0") // valid
	f.Add("", "deployment-pkg", "1.0.0", "app1", "1.0.0")        // missing project
	f.Add("project", "", "1.0.0", "app1", "1.0.0")               // missing pkgName
	f.Add("project", "deployment-pkg", "", "app1", "1.0.0")      // missing pkgVersion
	f.Add("project", "deployment-pkg", "1.0.0", "", "1.0.0")     // missing applicationName
	f.Add("project", "deployment-pkg", "1.0.0", "app1", "")      // missing applicationVersion

	f.Fuzz(func(t *testing.T, project, pkgName, pkgVersion, applicationName, applicationVersion string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		// --- Create Application Reference ---
		err := testSuite.createApplicationReference(project, pkgName, pkgVersion, applicationName, applicationVersion)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete Application Reference ---
		err = testSuite.deleteApplicationReference(project, pkgName, pkgVersion, applicationName)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
