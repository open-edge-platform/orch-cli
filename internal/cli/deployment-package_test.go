// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/stretchr/testify/assert"
)

func (s *CLITestSuite) createDeploymentPackage(project string, applicationName string, applicationVersion string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`create deployment-package --project %s %s %s`, project, applicationName, applicationVersion))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) listDeploymentPackages(project string, verbose bool, orderBy string, filter string) (string, error) {
	args := `get deployment-packages --project ` + project
	if verbose {
		args = args + " -v"
	}
	if orderBy != "" {
		args = args + " order-by=" + orderBy
	}
	if filter != "" {
		args = args + " filter=" + filter
	}
	getCmdOutput, err := s.runCommand(args)
	return getCmdOutput, err
}

func (s *CLITestSuite) getDeploymentPackage(project string, pkgName string, pkgVersion string) (string, error) {
	getCmdOutput, err := s.runCommand(fmt.Sprintf(`get deployment-package --project %s %s %s`, project, pkgName, pkgVersion))
	return getCmdOutput, err
}

func (s *CLITestSuite) deleteDeploymentPackage(project string, pkgName string, pkgVersion string) error {
	_, err := s.runCommand(fmt.Sprintf(`delete deployment-package --project %s %s %s`, project, pkgName, pkgVersion))
	return err
}

func (s *CLITestSuite) deleteDeploymentPackageNoVersion(project string, pkgName string) error {
	_, err := s.runCommand(fmt.Sprintf(`delete deployment-package --project %s %s`, project, pkgName))
	return err
}

func (s *CLITestSuite) updateDeploymentPackage(project string, pkgName string, pkgVersion string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`set deployment-package --project %s %s %s`, project, pkgName, pkgVersion))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) createTestDeploymentPackage(project string, pkgName string, pkgVersion string, appName string, appVersion string) error {
	createArgs := map[string]string{
		"application-reference": fmt.Sprintf("%s:%s:%s", appName, appVersion, project),
	}
	return s.createDeploymentPackage(project, pkgName, pkgVersion, createArgs)
}

func (s *CLITestSuite) exportDeploymentPackage(project string, pkgName string, pkgVersion string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`export deployment-package --project %s %s %s`, project, pkgName, pkgVersion))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) TestDeploymentPackage() {
	const (
		app1                         = "app1"
		app2                         = "app2"
		project                      = "pubtest"
		pkgName                      = "deployment-pkg"
		pkgVersion                   = "1.0"
		deploymentPackageDisplayName = "deployment.package.display.name"
		deploymentPackageDescription = "Publisher.for.testing"
	)

	// create several test applications
	s.NoError(s.createTestRegistry(project))
	s.NoError(s.createTestApplication(project, app1))
	s.NoError(s.createTestApplication(project, app2))

	// create a deployment package
	createArgs := map[string]string{
		"application-reference": fmt.Sprintf("%s:%s:%s,%s:%s:%s",
			app1, pkgVersion, project,
			app2, pkgVersion, project,
		),
		"display-name": deploymentPackageDisplayName,
		"description":  deploymentPackageDescription,
	}
	err := s.createDeploymentPackage(project, pkgName, pkgVersion, createArgs)
	s.NoError(err)

	// list deployment packages to make sure it was created properly
	listOutput, err := s.listDeploymentPackages(project, simpleOutput, "display_name", "display_name="+deploymentPackageDisplayName)
	s.NoError(err)

	parsedOutput := mapCliOutput(listOutput)
	expectedOutput := commandOutput{
		pkgName: {
			"Name":              pkgName,
			"Version":           pkgVersion,
			"Kind":              "normal",
			"Display Name":      deploymentPackageDisplayName,
			"Default Profile":   "default-profile",
			"Is Deployed":       "false",
			"Is Visible":        "true",
			"Application Count": "2",
		},
	}

	s.compareOutput(expectedOutput, parsedOutput)

	// verbose list deployment packages
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
			"Display Name":             deploymentPackageDisplayName,
			"Description":              deploymentPackageDescription,
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

	fmt.Println(listVerboseOutput)
	fmt.Printf("Parsed output:\n%v\n", parsedVerboseOutput)
	fmt.Printf("Expected output:\n%v\n", expectedVerboseOutput)
	s.compareOutput(expectedVerboseOutput, parsedVerboseOutput)

	// Update the deployment package
	updateArgs := map[string]string{
		"display-name": "new.display-name",
	}
	err = s.updateDeploymentPackage(project, pkgName, pkgVersion, updateArgs)
	s.NoError(err)

	// check that the deployment package was updated
	_, err = s.getDeploymentPackage(project, pkgName, pkgVersion)
	s.NoError(err)
	// TODO commended out not viable to test with mock
	// parsedGetOutput := mapCliOutput(getCmdOutput)
	// expectedOutput[pkgName]["Display Name"] = `new.display-name`
	// s.compareOutput(expectedOutput, parsedGetOutput)

	// delete a single app version from the deployment package
	err = s.deleteDeploymentPackage(project, pkgName, pkgVersion)
	s.NoError(err)

	//TODO not viable to mock
	// // Make sure deployment package is gone
	// _, err = s.getDeploymentPackage(project, pkgName, pkgVersion)
	// s.Error(err)
	// s.Contains(err.Error(), fmt.Sprintf("deployment-package %s:%s not found", pkgName, pkgVersion))

	err = s.deleteDeploymentPackageNoVersion(project, pkgName)
	s.NoError(err)
	// TODO not viable to mock// delete all versions from the deployment package. None left, so should fail
	// err = s.deleteDeploymentPackageNoVersion(project, pkgName)
	// s.Error(err)
	// s.Contains(err.Error(), fmt.Sprintf("deployment package versions %s: 404 Not Found", pkgName))
}

func TestPrintDeploymentPackageEvent(t *testing.T) {
	kind := catapi.DeploymentPackageKind("normal")
	dp := catapi.DeploymentPackage{
		Name:        "test-deployment-pkg",
		Version:     "1.0.0",
		DisplayName: strPtr("Test Deployment Package"),
		Description: strPtr("A test deployment package"),
		Kind:        &kind,
	}
	payload, err := json.Marshal(dp)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = printDeploymentPackageEvent(&buf, "DeploymentPackage", payload, false)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "test-deployment-pkg")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "Test Deployment Package")
}
