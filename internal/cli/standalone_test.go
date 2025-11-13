// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) generateStandaloneConfig(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`generate standalone-config --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestStandalone() {
	filename := "./testdata/config-file"

	/////////////////////////////
	// Test Standalone Generation
	/////////////////////////////

	// Save original function and restore after test
	originalFunc := getPasswordFromUser
	defer func() { getPasswordFromUser = originalFunc }()

	// Mock the function to return test password
	getPasswordFromUser = func() (string, error) {
		return "pass", nil
	}

	// Test standalone config
	SArgs := map[string]string{
		"config-file": filename,
	}

	_, err := s.generateStandaloneConfig(project, SArgs)
	s.NoError(err)

	// Test standalone config with specific output
	SArgs = map[string]string{
		"config-file": filename,
		"output-file": "my-config.yaml",
	}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.NoError(err)

	//Preload with version
	SArgs = map[string]string{
		"config-file":       filename,
		"emts-repo-version": "standalone-node/3.1.0",
	}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.NoError(err)

	//Invalid config
	SArgs = map[string]string{
		"config-file": "nope",
	}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.EqualError(err, "failed to parse YAML block: open nope: no such file or directory")

	//No config
	SArgs = map[string]string{}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.EqualError(err, "required flag \"config-file\" not set")

	//Preload with wrong repo version
	SArgs = map[string]string{
		"config-file":       filename,
		"emts-repo-version": "sadasdasdsd",
	}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.EqualError(err, "bad status: 404 Not Found")

	// Test standalone config with specific invalid output
	SArgs = map[string]string{
		"config-file": filename,
		"output-file": "/",
	}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.EqualError(err, "failed to write cloud-init to path \"/\"")

}

func FuzzGenerateStandaloneConfig(f *testing.F) {
	// Initial corpus with valid and invalid input
	f.Add("project", "./testdata/standalone.env", "cloud-init.cfg", "standalone-node/3.1.0")
	f.Add("project", "", "cloud-init.cfg", "standalone-node/3.1.0")                       // missing config file
	f.Add("project", "./testdata/standalone.env", "", "standalone-node/3.1.0")            // missing output file
	f.Add("project", "./testdata/standalone.env", "cloud-init.cfg", "")                   // missing repo version
	f.Add("project", "./testdata/invalid.env", "cloud-init.cfg", "standalone-node/3.1.0") // invalid config file

	f.Fuzz(func(t *testing.T, project, configFile, outputFile, repoVersion string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		SArgs := map[string]string{
			"config-file":       configFile,
			"output-file":       outputFile,
			"emts-repo-version": repoVersion,
		}

		_, err := testSuite.generateStandaloneConfig(project, SArgs)

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
