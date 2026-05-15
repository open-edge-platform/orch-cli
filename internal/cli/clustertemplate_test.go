// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) listClusterTemplates(publisher string, verbose bool, orderBy string, filter string, outputType string, pageSize string, args commandArgs) (string, error) {
	commandString := fmt.Sprintf(`list clustertemplates --project %s`, publisher)
	if verbose {
		commandString += " --verbose"
	}
	if orderBy != "" {
		commandString += " --order-by=" + orderBy
	}
	if filter != "" {
		commandString += " --filter=" + filter
	}
	if outputType != "" {
		commandString += " --output-type=" + outputType
	}
	if pageSize != "" {
		commandString += " --page-size=" + pageSize
	}
	commandString = addCommandArgs(args, commandString)
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestClusterTemplate() {

	/////////////////////////////
	// Test Cluster List
	/////////////////////////////

	//List cluster
	_, err := s.listClusterTemplates(project, false, "", "", "", "", commandArgs{})
	s.NoError(err)

	//List cluster with non existent project
	_, err = s.listClusterTemplates("nonexistent-project", false, "", "", "", "", commandArgs{})
	s.Error(err)

	//List cluster --verbose
	_, err = s.listClusterTemplates(project, true, "", "", "", "", commandArgs{})
	s.NoError(err)

	expectedOrderedOutput := linesCommandOutput{
		"- clusterlabels:",
		"    created-by: test",
		"  clusternetwork:",
		"    pods:",
		"      cidrblocks:",
		"      - 10.244.0.0/16",
		"    services:",
		"      cidrblocks:",
		"      - 10.96.0.0/12",
		"  clusterconfiguration:",
		"    apiServer:",
		"      port: 6443",
		"  controlplaneprovidertype: kubernetes",
		"  description: Default Kubernetes cluster template",
		"  infraprovidertype: type",
		"  kubernetesversion: v1.28.0",
		"  name: default-template",
		"  version: v1.0.0",
		"- clusterlabels:",
		"    created-by: test",
		"  clusternetwork:",
		"    pods:",
		"      cidrblocks:",
		"      - 10.244.0.0/16",
		"    services:",
		"      cidrblocks:",
		"      - 10.96.0.0/12",
		"  clusterconfiguration:",
		"    apiServer:",
		"      port: 6443",
		"  controlplaneprovidertype: kubernetes",
		"  description: High availability cluster template",
		"  infraprovidertype: type",
		"  kubernetesversion: v1.28.0",
		"  name: ha-template",
		"  version: v1.1.0",
	}

	// List cluster templates with order-by and YAML output
	listOrderedOutput, err := s.listClusterTemplates(project, false, "name", "", "yaml", "1", commandArgs{})
	s.NoError(err)
	s.compareLinesOutput(expectedOrderedOutput, mapLinesOutput(listOrderedOutput))

	// List cluster templates with filter and YAML output
	listFilteredOutput, err := s.listClusterTemplates(project, false, "", "name=default-template", "yaml", "1", commandArgs{})
	s.NoError(err)
	s.compareLinesOutput(expectedOrderedOutput, mapLinesOutput(listFilteredOutput))
}

func FuzzClusterTemplate(f *testing.F) {
	// Seed with valid and invalid input combinations
	f.Add("true", project)
	f.Add("false", project)
	f.Add("", project)
	f.Add("", "")

	f.Fuzz(func(t *testing.T, flag, publisher string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		verbose := flag == "true"
		// --- List Cluster ---
		_, err := testSuite.listClusterTemplates(publisher, verbose, "", "", "", "", commandArgs{})
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
