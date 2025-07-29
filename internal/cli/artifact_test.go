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

func (s *CLITestSuite) createArtifact(project string, artifactName string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`create artifact --project %s %s`, project, artifactName))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) listArtifacts(project string, verbose bool, orderBy string, filter string) (string, error) {
	args := `get artifacts --project ` + project
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

func (s *CLITestSuite) getArtifact(project string, artifactName string) (string, error) {
	getCmdOutput, err := s.runCommand(fmt.Sprintf(`get artifact --project %s %s`, project, artifactName))
	return getCmdOutput, err
}

func (s *CLITestSuite) deleteArtifact(project string, artifactName string) error {
	_, err := s.runCommand(fmt.Sprintf(`delete artifact --project %s %s`, project, artifactName))
	return err
}

func (s *CLITestSuite) updateArtifact(project string, artifactName string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`set artifact --project %s %s`, project, artifactName))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) TestArtifact() {
	const (
		artifactName        = "artifact"
		artifactFile        = "testdata/artifact.txt"
		textMimeType        = "text/plain"
		artifactDisplayName = "artifact-display-name"
		artifactDescription = "Artifact-Description"
	)

	// create an artifact
	createArgs := map[string]string{
		"artifact":     artifactFile,
		"display-name": artifactDisplayName,
		"description":  artifactDescription,
		"mime-type":    textMimeType,
	}
	err := s.createArtifact(project, artifactName, createArgs)
	s.NoError(err)

	// list artifacts to make sure it was created properly
	listOutput, err := s.listArtifacts(project, simpleOutput, "", "")
	s.NoError(err)

	parsedOutput := mapCliOutput(listOutput)
	expectedOutput := commandOutput{
		artifactName: {
			"Name":         artifactName,
			"Description":  artifactDescription,
			"Display Name": artifactName,
		},
	}

	s.compareOutput(expectedOutput, parsedOutput)

	// verbose list artifact
	listVerboseOutput, err := s.listArtifacts(project, verboseOutput, "name", "description="+artifactDescription)
	s.NoError(err)

	parsedVerboseOutput := mapVerboseCliOutput(listVerboseOutput)
	expectedVerboseOutput := commandOutput{
		artifactName: {
			"Name":         artifactName,
			"Display Name": artifactDisplayName,
			"Description":  artifactDescription,
			"Mime Type":    textMimeType,
		},
	}

	s.compareOutput(expectedVerboseOutput, parsedVerboseOutput)

	// Update the artifact
	updateArgs := map[string]string{
		"description": "new-description",
	}
	err = s.updateArtifact(project, artifactName, updateArgs)
	s.NoError(err)

	// check that the artifact was updated
	_, err = s.getArtifact(project, artifactName)
	s.NoError(err)

	// TODO not viable to test via mock
	// parsedGetOutput := mapCliOutput(getCmdOutput)
	// expectedOutput[artifactName]["Description"] = `new-description`
	// s.compareOutput(expectedOutput, parsedGetOutput)

	// delete the artifact
	err = s.deleteArtifact(project, artifactName)
	s.NoError(err)

	// Not viable to test via mock
	// // Make sure artifact is gone
	// _, err = s.getArtifact(project, artifactName)
	// s.Error(err)
	// s.Contains(err.Error(), `artifact not found`)
}

func TestPrintArtifactEvent(t *testing.T) {
	artifact := catapi.Artifact{
		Name:        "test-artifact",
		DisplayName: strPtr("Test Artifact"),
		Description: strPtr("A test artifact"),
		MimeType:    "application/octet-stream",
	}
	payload, err := json.Marshal(artifact)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = printArtifactEvent(&buf, "Artifact", payload, false)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "test-artifact")
	assert.Contains(t, output, "Test Artifact")
	assert.Contains(t, output, "A test artifact")
}
