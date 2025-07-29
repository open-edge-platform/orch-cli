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

func (s *CLITestSuite) createApplication(project string, applicationName string, applicationVersion string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`create application --project %s %s %s`, project, applicationName, applicationVersion))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) listApplications(project string, verbose bool, orderBy string, filter string, kind string) (string, error) {
	args := `get applications --project ` + project
	if verbose {
		args = args + " -v"
	}
	if orderBy != "" {
		args = args + " order-by=" + orderBy
	}
	if filter != "" {
		args = args + " filter=" + filter
	}
	if kind != "" {
		args = args + " kind=" + kind
		fmt.Println("yoyoyoyoyoyowe-qor wefopkfk XXXXXXXXX:wq")
	}
	getCmdOutput, err := s.runCommand(args)
	return getCmdOutput, err
}

func (s *CLITestSuite) getApplication(pubName string, applicationName string, applicationVersion string) (string, error) {
	getCmdOutput, err := s.runCommand(fmt.Sprintf(`get application --project %s %s %s`, pubName, applicationName, applicationVersion))
	return getCmdOutput, err
}

func (s *CLITestSuite) deleteApplication(pubName string, applicationName string, applicationVersion string) error {
	_, err := s.runCommand(fmt.Sprintf(`delete application --project %s %s %s`, pubName, applicationName, applicationVersion))
	return err
}

func (s *CLITestSuite) updateApplication(pubName string, applicationName string, applicationVersion string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`set application --project %s %s %s`, pubName, applicationName, applicationVersion))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) createTestApplication(pubName string, applicationName string) error {
	createArgs := map[string]string{
		"chart-name":     "chart",
		"chart-registry": "reg",
		"chart-version":  "1.0",
	}
	return s.createApplication(pubName, applicationName, "1.0", createArgs)
}

func (s *CLITestSuite) TestApplication() {
	const (
		applicationName        = "new-application"
		applicationDisplayName = "application.display.name"
		applicationDescription = "Application.Description"
		registryName           = "test-registry"
		chartName              = "chart-name"
		chartVersion           = "22.33.44"
		applicationVersion     = "1.2.3"
	)

	// create a registry
	createRegArgs := map[string]string{
		"root-url": "http://1.2.3.4",
	}
	err := s.createRegistry(project, registryName, createRegArgs)
	s.NoError(err)

	// create an application
	createArgs := map[string]string{
		"chart-name":     chartName,
		"chart-registry": registryName,
		"chart-version":  chartVersion,
		"display-name":   applicationDisplayName,
		"description":    applicationDescription,
	}
	err = s.createApplication(project, applicationName, applicationVersion, createArgs)
	s.NoError(err)

	// list applications to make sure it was created properly
	listOutput, err := s.listApplications(project, simpleOutput, "version", "version="+applicationVersion, "")
	s.NoError(err)

	parsedOutput := mapCliOutput(listOutput)
	expectedOutput := commandOutput{
		applicationName: {
			"Name":               applicationName,
			"Version":            applicationVersion,
			"Kind":               "normal",
			"Display Name":       applicationDisplayName,
			"Chart Name":         chartName,
			"Chart Version":      chartVersion,
			"Helm Registry Name": registryName,
			"Default Profile":    "",
		},
	}
	s.compareOutput(expectedOutput, parsedOutput)

	// verbose list applications
	listVerboseOutput, err := s.listApplications(project, verboseOutput, "", "", "")
	s.NoError(err)

	parsedVerboseOutput := mapVerboseCliOutput(listVerboseOutput)
	expectedVerboseOutput := commandOutput{
		applicationName: {
			"Version":             applicationVersion,
			"Chart Name":          chartName,
			"Chart Version":       chartVersion,
			"Create Time":         timestampRegex,
			"Update Time":         timestampRegex,
			"Name":                applicationName,
			"Kind":                "normal",
			"Display Name":        applicationDisplayName,
			"Description":         applicationDescription,
			"Helm Registry Name":  registryName,
			"Image Registry Name": "<none>",
			"Profiles":            "[]",
			"Default Profile":     "",
		},
	}

	s.compareOutput(expectedVerboseOutput, parsedVerboseOutput)

	// List with kind
	_, err = s.listApplications(project, verboseOutput, "", "", "\"addon normal\"")
	s.NoError(err)

	// Update the application
	updateArgs := map[string]string{
		"display-name": "new.display-name",
	}
	err = s.updateApplication(project, applicationName, applicationVersion, updateArgs)
	s.NoError(err)

	// check that the application was updated
	_, err = s.getApplication(project, applicationName, applicationVersion)
	s.NoError(err)
	//TODO not viable to mock at this moment
	// parsedGetOutput := mapCliOutput(getCmdOutput)
	// expectedOutput[applicationName]["Display Name"] = `new.display-name`
	// s.compareOutput(expectedOutput, parsedGetOutput)

	//Check application with one argument
	_, err = s.getApplication(project, applicationName, "")
	s.NoError(err)

	// delete the application
	err = s.deleteApplication(project, applicationName, "applicationVersion")
	s.NoError(err)

	// delete the application without version
	err = s.deleteApplication(project, applicationName, "")
	s.NoError(err)

	//TODO not viable to mock at this moment
	// // Make sure application is gone
	// _, err = s.getApplication(project, applicationName, applicationVersion)
	// s.Error(err)
	// s.Contains(err.Error(), `application new-application:1.2.3 not found`)

	// try with invalid names
	err = s.deleteApplication("", "", applicationVersion)
	s.Error(err)
	s.Contains(err.Error(), `accepts between 1 and 2 arg(s), received 0`)

	err = s.deleteApplication("missing-pub", "missing-app", "1.0")
	s.Error(err)
	s.Contains(err.Error(), `application missing-app:1.0 not found`)

	err = s.deleteApplication(project, "missing-app", "1.0")
	s.Error(err)
	s.Contains(err.Error(), `application missing-app:1.0 not found`)

	//Additional tests
	// create applications with kind
	createArgs = map[string]string{
		"chart-name":     chartName,
		"chart-registry": registryName,
		"chart-version":  chartVersion,
		"display-name":   applicationDisplayName,
		"description":    applicationDescription,
		"kind":           "addon",
	}
	err = s.createApplication(project, applicationName, applicationVersion, createArgs)
	s.NoError(err)

	createArgs = map[string]string{
		"chart-name":     chartName,
		"chart-registry": registryName,
		"chart-version":  chartVersion,
		"display-name":   applicationDisplayName,
		"description":    applicationDescription,
		"kind":           "extension",
	}
	err = s.createApplication(project, applicationName, applicationVersion, createArgs)
	s.NoError(err)
}

func TestPrintApplicationEvent(t *testing.T) {
	kind := catapi.ApplicationKindKINDNORMAL
	app := catapi.Application{
		Name:               "test-app",
		Version:            "1.0.0",
		Kind:               &kind, // take address of variable, not constant
		DisplayName:        strPtr("Test App"),
		Description:        strPtr("A test application"),
		ChartName:          "test-chart",
		ChartVersion:       "0.1.0",
		HelmRegistryName:   "test-registry",
		Profiles:           &[]catapi.Profile{},
		DefaultProfileName: strPtr("default"),
	}
	payload, err := json.Marshal(app)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = printApplicationEvent(&buf, "Application", payload, false)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "test-app")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "test-chart")
	assert.Contains(t, output, "test-registry")
}

func strPtr(s string) *string { return &s }
