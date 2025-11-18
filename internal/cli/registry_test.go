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

const (
	registryImageName   = "registry-image"
	registryHelmName    = "registry-helm"
	registryRootURL     = "http://x.y.z"
	registryDisplayName = "registry-display-name"
	registryDescription = "Registry-Description"
	registryHelmType    = "HELM"
	registryImageType   = "IMAGE"
	registryImageParam  = "image"
	registryHelmParam   = "helm"
)

func (s *CLITestSuite) createRegistry(project string, name string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`create registry --project %s %s`, project, name))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) listRegistries(project string, verbose bool, showSensitive bool, orderBy string, filter string) (string, error) {
	args := `list registries --project ` + project
	if verbose {
		args = args + " -v"
		if showSensitive {
			args = args + " --show-sensitive-info"
		}
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

func (s *CLITestSuite) getRegistry(project string, regName string) (string, error) {
	getCmdOutput, err := s.runCommand(fmt.Sprintf(`get registry --project %s %s`, project, regName))
	return getCmdOutput, err
}

func (s *CLITestSuite) deleteRegistry(project string, regName string) error {
	_, err := s.runCommand(fmt.Sprintf(`delete registry --project %s %s`, project, regName))
	return err
}

func (s *CLITestSuite) updateRegistry(project string, regName string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`set registry --project %s %s`, project, regName))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) createTestRegistry(project string) error {
	createRegArgs := map[string]string{
		"root-url": "http://1.2.3.4",
	}
	return s.createRegistry(project, "reg", createRegArgs)
}

func (s *CLITestSuite) setupRegistry(registryType string, registryName string) {
	// create a registry for the new publisher
	createArgs := map[string]string{
		"root-url":     registryRootURL,
		"display-name": registryDisplayName,
		"description":  registryDescription,
		"username":     "user",
		"auth-token":   "token",
	}
	if registryType != "" {
		createArgs["registry-type"] = registryType
	}
	err := s.createRegistry(project, registryName, createArgs)
	s.NoError(err)
}

func (s *CLITestSuite) removeRegistry(registryName string) {
	// delete the registry
	err := s.deleteRegistry(project, registryName)
	s.NoError(err)

	// Commenting out not supported by mock test
	// // Make sure registry is gone
	// _, err = s.getRegistry(project, registryName)
	// s.Error(err)
	// s.Contains(err.Error(), `registry not found`)
}

func (s *CLITestSuite) registryTest(registryTypeCommand string, registryTypeValue string, registryName string) {
	s.setupRegistry(registryTypeCommand, registryName)

	// list registries to make sure it was created properly
	listOutput, err := s.listRegistries(project, simpleOutput, false, "name desc", "description="+registryDescription)
	s.NoError(err)

	parsedOutput := mapCliOutput(listOutput)
	expectedOutput := commandOutput{
		registryName: {
			"Name":         registryName,
			"Display Name": registryDisplayName,
			"Description":  registryDescription,
			"Type":         registryTypeValue,
			"Root URL":     registryRootURL,
		},
	}

	s.compareOutput(expectedOutput, parsedOutput)

	// verbose list registry (show sensitive)
	listVerboseOutput, err := s.listRegistries(project, verboseOutput, true, "", "")
	s.NoError(err)

	parsedVerboseOutput := mapVerboseCliOutput(listVerboseOutput)
	expectedVerboseOutput := commandOutput{
		registryName: {
			"Name":          registryName,
			"Display Name":  registryDisplayName,
			"Description":   registryDescription,
			"Root URL":      registryRootURL,
			"Inventory URL": "<none>",
			"Type":          registryTypeValue,
			"API Type":      "<none>",
			"Username":      "user",
			"AuthToken":     "token",
			"CA Certs":      "<none>",
			"Create Time":   timestampRegex,
			"Update Time":   timestampRegex,
		},
	}

	s.compareOutput(expectedVerboseOutput, parsedVerboseOutput)

	// verbose list registry (hide sensitive)
	listVerboseOutput, err = s.listRegistries(project, verboseOutput, false, "", "")
	s.NoError(err)

	parsedVerboseOutput = mapVerboseCliOutput(listVerboseOutput)
	expectedVerboseOutput = commandOutput{
		registryName: {
			"Name":          registryName,
			"Display Name":  registryDisplayName,
			"Description":   registryDescription,
			"Root URL":      registryRootURL,
			"Inventory URL": "<none>",
			"Type":          registryTypeValue,
			"API Type":      "<none>",
			"Username":      "<none>",
			"AuthToken":     "********",
			"CA Certs":      "<none>",
			"Create Time":   timestampRegex,
			"Update Time":   timestampRegex,
		},
	}

	s.compareOutput(expectedVerboseOutput, parsedVerboseOutput)

	// Update the registry
	updateArgs := map[string]string{
		"description": "new-description",
	}
	err = s.updateRegistry(project, registryName, updateArgs)
	s.NoError(err)

	// check that the registry was updated
	getCmdOutput, err := s.getRegistry(project, registryName)
	s.NoError(err)

	parsedGetOutput := mapCliOutput(getCmdOutput)
	expectedOutput[registryName]["Description"] = `new-description`
	s.compareOutput(expectedOutput, parsedGetOutput)

	s.removeRegistry(registryName)
}

func (s *CLITestSuite) TestHelmRegistry() {
	s.registryTest(registryHelmParam, registryHelmType, registryHelmName)
}

func (s *CLITestSuite) TestImageRegistry() {
	s.registryTest(registryImageParam, registryImageType, registryImageName)
}

func TestPrintRegistryEvent(t *testing.T) {
	reg := catapi.CatalogV3Registry{
		Name:        "test-registry",
		DisplayName: strPtr("Test Registry"),
		Description: strPtr("A test registry"),
		Type:        "HELM",
	}
	payload, err := json.Marshal(reg)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = printRegistryEvent(&buf, "Registry", payload, false)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "test-registry")
	assert.Contains(t, output, "Test Registry")
	assert.Contains(t, output, "A test registry")
}

func FuzzRegistry(f *testing.F) {
	// Seed with valid and invalid input combinations
	f.Add("project", "reg1", "HELM", "http://1.2.3.4", "Registry-Description", "user", "token")
	f.Add("project", "", "HELM", "http://1.2.3.4", "Registry-Description", "user", "token") // missing name
	f.Add("", "reg1", "HELM", "http://1.2.3.4", "Registry-Description", "user", "token")    // missing project
	f.Add("project", "reg1", "", "http://1.2.3.4", "Registry-Description", "user", "token") // missing type
	f.Add("project", "reg1", "HELM", "", "Registry-Description", "user", "token")           // missing root-url
	f.Add("project", "reg1", "HELM", "http://1.2.3.4", "", "user", "token")                 // missing description

	f.Fuzz(func(t *testing.T, project, name, regType, rootURL, description, username, token string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		createArgs := map[string]string{
			"registry-type": regType,
			"root-url":      rootURL,
			"description":   description,
			"username":      username,
			"auth-token":    token,
		}

		// --- Create ---
		err := testSuite.createRegistry(project, name, createArgs)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List ---
		_, err = testSuite.listRegistries(project, false, false, "", "")
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Get ---
		_, err = testSuite.getRegistry(project, name)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Update ---
		updateArgs := map[string]string{
			"description": "new-description",
		}
		err = testSuite.updateRegistry(project, name, updateArgs)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete ---
		err = testSuite.deleteRegistry(project, name)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
