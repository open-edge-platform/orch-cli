// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) createApplication(project string, applicationName string, applicationVersion string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`create application --project %s %s %s`, project, applicationName, applicationVersion))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) listApplications(project string, verbose bool, orderBy string, filter string) (string, error) {
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
	listOutput, err := s.listApplications(project, simpleOutput, "version", "version="+applicationVersion)
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
	listVerboseOutput, err := s.listApplications(project, verboseOutput, "", "")
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
			"Image Registry Name": "\\<none\\>",
			"Profiles":            "\\[\\]",
			"Default Profile":     "",
		},
	}
	s.compareOutput(expectedVerboseOutput, parsedVerboseOutput)

	// Update the application
	updateArgs := map[string]string{
		"display-name": "new.display-name",
	}
	err = s.updateApplication(project, applicationName, applicationVersion, updateArgs)
	s.NoError(err)

	// check that the application was updated
	getCmdOutput, err := s.getApplication(project, applicationName, applicationVersion)
	s.NoError(err)
	parsedGetOutput := mapCliOutput(getCmdOutput)
	expectedOutput[applicationName]["Display Name"] = `new.display-name`
	s.compareOutput(expectedOutput, parsedGetOutput)

	// delete the application
	err = s.deleteApplication(project, applicationName, applicationVersion)
	s.NoError(err)

	// Make sure application is gone
	_, err = s.getApplication(project, applicationName, applicationVersion)
	s.Error(err)
	s.Contains(err.Error(), `application new-application:1.2.3 not found`)

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
}
