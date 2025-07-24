// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import "fmt"

func (s *CLITestSuite) createCluster(publisher string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create cluster %s --project %s`, name, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listCluster(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list cluster --project %s`, publisher))
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

	/////////////////////////////
	// Test Cluster List
	/////////////////////////////

	//List cluster
	CArgs = map[string]string{}

	_, err = s.listCluster(project, CArgs)
	s.NoError(err)

	/////////////////////////////
	// Test Cluster List
	/////////////////////////////

	//List cluster
	CArgs = map[string]string{}

	_, err = s.listCluster(project, CArgs)
	s.NoError(err)

	//List cluster verbose
	CArgs = map[string]string{
		"verbose": "true",
	}

	_, err = s.listCluster(project, CArgs)
	s.NoError(err)

	//List cluster by not ready
	CArgs = map[string]string{
		"not-ready": "",
	}

	_, err = s.listCluster(project, CArgs)
	s.NoError(err)

	/////////////////////////////
	// Test Cluster Get
	/////////////////////////////

	//Get cluster
	CArgs = map[string]string{}

	_, err = s.getCluster(project, name, CArgs)
	s.NoError(err)

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
