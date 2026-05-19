// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) createCluster(publisher string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create cluster %s --project %s`, name, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listCluster(publisher string, verbose bool, orderBy string, filter string, outputType string, pageSize string, args commandArgs) (string, error) {
	commandString := fmt.Sprintf(`list cluster --project %s`, publisher)
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

func (s *CLITestSuite) getCluster(publisher string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get cluster %s --project %s`, name, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteCluster(publisher string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete cluster %s --project %s`, name, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestCluster() {
	name := "test-cluster-1"

	/////////////////////////////
	// Test Cluster Create
	/////////////////////////////

	//Create cluster
	CArgs := map[string]string{
		"nodes": "d7911144-3010-11f0-a1c2-370d26b04195:all",
	}
	_, err := s.createCluster(project, name, CArgs)
	s.NoError(err)

	//Create without nodes
	CArgs = map[string]string{}
	_, err = s.createCluster(project, name, CArgs)
	s.EqualError(err, "required flag(s) \"nodes\" not set")

	//Create cluster
	CArgs = map[string]string{
		"nodes":   "d7911144-3010-11f0-a1c2-370d26b04195:all",
		"verbose": "",
	}
	_, err = s.createCluster(project, name, CArgs)
	s.NoError(err)

	//Create cluster - hostby name
	CArgs = map[string]string{
		"nodes":   "edge-host-001:all",
		"verbose": "",
	}
	_, err = s.createCluster(project, name, CArgs)
	s.NoError(err)

	//Create cluster
	CArgs = map[string]string{
		"nodes":   "duplicate:all",
		"verbose": "",
	}
	_, err = s.createCluster("duplicate-host", name, CArgs)
	s.EqualError(err, "multiple hosts found with name \"duplicate\"; use a resource ID instead:\n  name: duplicate  resource-id: host-abc12345\n  name: duplicate  resource-id: host-abc12345")

	/////////////////////////////
	// Test Cluster List
	/////////////////////////////

	//List cluster
	_, err = s.listCluster(project, false, "", "", "", "", commandArgs{})
	s.NoError(err)

	/////////////////////////////
	// Test Cluster List
	/////////////////////////////

	//List cluster
	_, err = s.listCluster(project, false, "", "", "", "", commandArgs{})
	s.NoError(err)

	//List cluster verbose
	verboseOut, err := s.listCluster(project, true, "", "", "", "", commandArgs{})
	s.NoError(err)

	expectedVerboseOutput := linesCommandOutput{
		"Name: test-cluster-1",
		"Kubernetes Version: v1.28.0",
		"Node Count: 2",
		"Status:",
		"  Lifecycle Phase: Provisioned",
		"  Provider Status: Ready",
		"  Control Plane Ready: Ready",
		"  Infrastructure Ready: Ready",
		"  Node Health: Healthy",
		"Labels: <none>",
		"",
	}
	s.compareLinesOutput(expectedVerboseOutput, mapLinesOutput(verboseOut))

	//List cluster by not ready
	_, err = s.listCluster(project, false, "", "", "", "", commandArgs{"not-ready": ""})
	s.NoError(err)

	expectedYAMLOutput := linesCommandOutput{
		"- controlplaneready:",
		"    indicator: STATUS_INDICATION_IDLE",
		"    message: Ready",
		"    timestamp: null",
		"  infrastructureready:",
		"    indicator: STATUS_INDICATION_IDLE",
		"    message: Ready",
		"    timestamp: null",
		"  kubernetesversion: v1.28.0",
		"  labels: null",
		"  lifecyclephase:",
		"    indicator: STATUS_INDICATION_IDLE",
		"    message: Provisioned",
		"    timestamp: null",
		"  name: test-cluster-1",
		"  nodehealth:",
		"    indicator: STATUS_INDICATION_IDLE",
		"    message: Healthy",
		"    timestamp: null",
		"  nodequantity: 2",
		"  providerstatus:",
		"    indicator: STATUS_INDICATION_IDLE",
		"    message: Ready",
		"    timestamp: null",
	}

	// List clusters with order-by and YAML output
	listOrderedOutput, err := s.listCluster(project, false, "name", "", "yaml", "1", commandArgs{})
	s.NoError(err)
	s.compareLinesOutput(expectedYAMLOutput, mapLinesOutput(listOrderedOutput))

	// List clusters with filter and YAML output
	listFilteredOutput, err := s.listCluster(project, false, "", "name=test-cluster-1", "yaml", "1", commandArgs{})
	s.NoError(err)
	s.compareLinesOutput(expectedYAMLOutput, mapLinesOutput(listFilteredOutput))

	/////////////////////////////
	// Test Cluster Get
	/////////////////////////////

	//Get cluster
	CArgs = map[string]string{}

	_, err = s.getCluster(project, name, CArgs)
	s.NoError(err)

	//Get cluster verbose
	verboseGetOut, err := s.getCluster(project, name, map[string]string{"verbose": ""})
	s.NoError(err)

	expectedVerboseGetOutput := linesCommandOutput{
		"Project: project",
		"Name: test-cluster-1",
		"Kubernetes Version: v1.28.0",
		"Template: default-template-v1.0.0",
		"Nodes:",
		"  - ID: default-node-id, Role: control-plane",
		"Status:",
		"  Lifecycle Phase: Provisioned",
		"  Provider Status: Ready",
		"  Control Plane Ready: Ready",
		"  Infrastructure Ready: Ready",
		"  Node Health: Healthy",
		"Labels:",
		"  created-by: test",
		"",
	}
	s.compareLinesOutput(expectedVerboseGetOutput, mapLinesOutput(verboseGetOut))

	//Get non existing cluster
	_, err = s.getCluster("nonexistent-cluster", "nonexistent-cluster", CArgs)
	s.EqualError(err, "failed to get cluster details: cluster nonexistent-cluster not found in project nonexistent-cluster")

	/////////////////////////////
	// Test Cluster Delete
	/////////////////////////////

	//List cluster
	CArgs = map[string]string{}

	_, err = s.deleteCluster(project, name, CArgs)
	s.NoError(err)

	//Delete cluster force
	CArgs = map[string]string{
		"force": "",
	}

	_, err = s.deleteCluster(project, name, CArgs)
	s.NoError(err)

	//Delete nonexisting cluster
	CArgs = map[string]string{}

	_, err = s.deleteCluster("nonexistent-project", name, CArgs)
	s.EqualError(err, "failed to soft delete cluster 'test-cluster-1': failed to delete cluster test-cluster-1: Not Found")

}

func FuzzCluster(f *testing.F) {
	// Seed with valid and invalid input combinations
	f.Add("test-cluster", "d7911144-3010-11f0-a1c2-370d26b04195:all", project)
	f.Add("", "d7911144-3010-11f0-a1c2-370d26b04195:all", project)
	f.Add("test-cluster", "", project)
	f.Add("test-cluster", "d7911144-3010-11f0-a1c2-370d26b04195:all", "")

	f.Fuzz(func(t *testing.T, clusterName, nodeName, publisher string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{
			"nodes": nodeName,
		}

		// --- Create Cluster ---
		_, err := testSuite.createCluster(clusterName, nodeName, args)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List Cluster ---
		_, err = testSuite.listCluster(publisher, false, "", "", "", "", commandArgs{})
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Get Cluster ---
		_, err = testSuite.getCluster(publisher, clusterName, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete Cluster ---
		_, err = testSuite.deleteCluster(publisher, clusterName, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
