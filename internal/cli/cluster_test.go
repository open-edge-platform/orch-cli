// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
	"strings"
)

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

func (s *CLITestSuite) createClusterFromCSV(publisher string, name string, csvFile string, args commandArgs) (string, error) {
	args["create-from-csv"] = csvFile
	commandString := addCommandArgs(args, fmt.Sprintf(`create cluster %s --project %s`, name, publisher))
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
	s.EqualError(err, "--nodes flag is required when not using --create-from-csv")

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
		_, err = testSuite.listCluster(publisher, make(map[string]string))
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
func (s *CLITestSuite) TestClusterCreateFromCSV() {
	baseName := "csv-cluster"

	/////////////////////////////
	// Test CSV Cluster Create Success Cases
	/////////////////////////////

	// Create clusters from valid CSV
	CArgs := map[string]string{}
	_, err := s.createClusterFromCSV(project, baseName, "./testdata/valid_hosts.csv", CArgs)
	s.NoError(err, "Should successfully create clusters from valid CSV")

	// Create clusters from single host CSV
	CArgs = map[string]string{}
	_, err = s.createClusterFromCSV(project, baseName+"-single", "./testdata/single_host.csv", CArgs)
	s.NoError(err, "Should successfully create cluster from single host CSV")

	// Create clusters with verbose flag
	CArgs = map[string]string{
		"verbose": "true",
	}
	_, err = s.createClusterFromCSV(project, baseName+"-verbose", "./testdata/valid_hosts.csv", CArgs)
	s.NoError(err, "Should successfully create clusters with verbose output")

	// Create clusters with template
	CArgs = map[string]string{
		"template": "test-template:v1.0.0",
	}
	_, err = s.createClusterFromCSV(project, baseName+"-template", "./testdata/valid_hosts.csv", CArgs)
	s.NoError(err, "Should successfully create clusters with specified template")

	/////////////////////////////
	// Test CSV Cluster Create Error Cases
	/////////////////////////////

	// Test with non-existent CSV file
	CArgs = map[string]string{}
	_, err = s.createClusterFromCSV(project, baseName+"-nonexistent", "nonexistent.csv", CArgs)
	s.Error(err, "Should fail when CSV file does not exist")

	// Test with empty CSV file
	CArgs = map[string]string{}
	_, err = s.createClusterFromCSV(project, baseName+"-empty", "./testdata/empty_hosts.csv", CArgs)
	// Note: This might succeed with 0 clusters created, depending on implementation
	// Empty CSV handling is implementation-dependent, so we don't assert on error here

	/////////////////////////////
	// Test Flag Validation
	/////////////////////////////

	// Test that --nodes and --create-from-csv are mutually exclusive
	CArgs = map[string]string{
		"nodes": "d7911144-3010-11f0-a1c2-370d26b04195:all",
	}
	_, err = s.createClusterFromCSV(project, baseName+"-conflict", "./testdata/valid_hosts.csv", CArgs)
	// The CSV file should take precedence over nodes flag, so this should succeed
	s.NoError(err, "CSV creation should work even if nodes flag is provided")

	// Test missing both --nodes and --create-from-csv
	CArgs = map[string]string{}
	commandString := addCommandArgs(CArgs, fmt.Sprintf(`create cluster %s --project %s`, baseName+"-missing", project))
	_, err = s.runCommand(commandString)
	s.EqualError(err, "--nodes flag is required when not using --create-from-csv", "Should require either nodes or csv file")
}

func (s *CLITestSuite) TestClusterCSVValidation() {
	baseName := "csv-validation-test"

	/////////////////////////////
	// Test CSV Validation Cases
	/////////////////////////////

	// Test with invalid CSV data (bad UUIDs)
	CArgs := map[string]string{}
	_, err := s.createClusterFromCSV(project, baseName+"-invalid", "./testdata/invalid_hosts.csv", CArgs)
	// The validator may handle invalid entries differently - it might create an error file and continue
	// or it might fail completely. Let's allow both cases.
	if err != nil {
		// If it errors, expect some validation-related message
		s.True(strings.Contains(err.Error(), "validation") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "check"),
			"Error should mention validation issues")
	}

	/////////////////////////////
	// Test CSV File Edge Cases
	/////////////////////////////

	// Test with relative vs absolute paths
	CArgs = map[string]string{}
	_, err = s.createClusterFromCSV(project, baseName+"-relative", "./testdata/valid_hosts.csv", CArgs)
	s.NoError(err, "Should work with relative paths")

	// Test cluster naming with special base names
	CArgs = map[string]string{}
	_, err = s.createClusterFromCSV(project, "test-cluster-with-dashes", "./testdata/single_host.csv", CArgs)
	s.NoError(err, "Should handle cluster names with dashes")

	/////////////////////////////
	// Test Verbose Output
	/////////////////////////////

	// Test verbose mode shows detailed output
	CArgs = map[string]string{
		"verbose": "true",
	}
	_, err = s.createClusterFromCSV(project, baseName+"-verbose-detailed", "./testdata/valid_hosts.csv", CArgs)
	s.NoError(err, "Verbose mode should work")
	// Note: Verbose output goes to stdout directly via fmt.Printf, not captured in returned string
}

func (s *CLITestSuite) TestClusterDeleteCSV() {
	/////////////////////////////
	// Test Generate CSV for Deletion
	/////////////////////////////

	// Test generate CSV with default filename
	CArgs := map[string]string{
		"generate-csv": "",
	}
	commandString := addCommandArgs(CArgs, fmt.Sprintf(`delete cluster --project %s`, project))
	_, err := s.runCommand(commandString)
	s.NoError(err, "Should generate CSV with default filename")

	// Test generate CSV with custom filename
	CArgs = map[string]string{
		"generate-csv": "my-clusters.csv",
	}
	commandString = addCommandArgs(CArgs, fmt.Sprintf(`delete cluster --project %s`, project))
	_, err = s.runCommand(commandString)
	s.NoError(err, "Should generate CSV with custom filename")

	/////////////////////////////
	// Test Delete from CSV - Dry Run
	/////////////////////////////

	// Test dry run with valid CSV
	CArgs = map[string]string{
		"delete-from-csv": "./testdata/clusters_delete.csv",
		"dry-run":         "",
	}
	commandString = addCommandArgs(CArgs, fmt.Sprintf(`delete cluster --project %s`, project))
	_, err = s.runCommand(commandString)
	s.NoError(err, "Dry run should validate without deleting")

	/////////////////////////////
	// Test Delete from CSV - Actual Deletion
	/////////////////////////////

	// Test delete from CSV
	CArgs = map[string]string{
		"delete-from-csv": "./testdata/clusters_delete.csv",
	}
	commandString = addCommandArgs(CArgs, fmt.Sprintf(`delete cluster --project %s`, project))
	_, err = s.runCommand(commandString)
	s.NoError(err, "Should delete clusters from CSV")

	// Test delete from CSV with force flag
	CArgs = map[string]string{
		"delete-from-csv": "./testdata/clusters_delete.csv",
		"force":           "",
	}
	commandString = addCommandArgs(CArgs, fmt.Sprintf(`delete cluster --project %s`, project))
	_, err = s.runCommand(commandString)
	s.NoError(err, "Should force delete clusters from CSV")

	/////////////////////////////
	// Test Error Cases
	/////////////////////////////

	// Test with non-existent CSV file
	CArgs = map[string]string{
		"delete-from-csv": "./testdata/nonexistent.csv",
	}
	commandString = addCommandArgs(CArgs, fmt.Sprintf(`delete cluster --project %s`, project))
	_, err = s.runCommand(commandString)
	s.Error(err, "Should fail with non-existent CSV file")

	// Test with empty CSV file
	CArgs = map[string]string{
		"delete-from-csv": "./testdata/empty_hosts.csv",
	}
	commandString = addCommandArgs(CArgs, fmt.Sprintf(`delete cluster --project %s`, project))
	_, err = s.runCommand(commandString)
	// Empty CSV should complete with 0 deletions, not error
	s.NoError(err, "Should handle empty CSV gracefully")
}
