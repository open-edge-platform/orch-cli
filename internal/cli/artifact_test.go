// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
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

	fmt.Printf(listOutput)
	fmt.Printf("Parsed: %+v\n", parsedOutput)
	fmt.Printf("Expected: %+v\n", expectedOutput)
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

	fmt.Printf(listVerboseOutput)
	fmt.Printf("Parsed: %+v\n", parsedVerboseOutput)
	fmt.Printf("Expected: %+v\n", expectedVerboseOutput)
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
