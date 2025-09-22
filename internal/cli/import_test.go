// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) importHelmChart(project string, path string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`import helm-chart %s --project %s`, path, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestImport() {
	IArgs := map[string]string{
		"values-file": "./testdata/values.yaml",
	}
	//TODO import needs refactoring to be more testable with mock
	_, err := s.importHelmChart(project, "oci://url", IArgs)
	s.Error(err)

}

func FuzzImportHelmChart(f *testing.F) {
	// Seed with valid and invalid input combinations
	f.Add("project", "oci://url", "./testdata/values.yaml")     // valid
	f.Add("", "oci://url", "./testdata/values.yaml")            // missing project
	f.Add("project", "", "./testdata/values.yaml")              // missing path
	f.Add("project", "oci://url", "")                           // missing values file
	f.Add("project", "oci://invalid", "./testdata/values.yaml") // invalid path
	f.Add("project", "oci://url", "./testdata/invalid.yaml")    // invalid values file

	f.Fuzz(func(t *testing.T, project, path, valuesFile string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{
			"values-file": valuesFile,
		}

		_, err := testSuite.importHelmChart(project, path, args)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
