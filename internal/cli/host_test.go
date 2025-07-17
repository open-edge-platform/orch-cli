// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import "fmt"

func (s *CLITestSuite) createHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getHost(publisher string, hostID string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get host %s --project %s`, hostID, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deauthorizeHost(publisher string, hostID string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`deauthorize host %s --project %s`, hostID, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteHost(publisher string, hostID string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete host %s --project %s`, hostID, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestHost() {

	resourceID := "host-abc12345"
	name := "edge-host-001"
	hostStatus := "Not connected"
	provisioningStatus := "Not provisioned"
	serialNumber := "1234567890"
	operatingSystem := "Not provisioned"
	siteID := "Not provisioned"
	siteName := "Not provisioned"
	workload := "\"Edge Kubernetes Cluster\""
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	processor := "Intel(R) Xeon(R) CPU E5-2670 v3"
	update := "No update"
	compute := "Not compatible"

	//hostID := "host-abc12345"
	HostArgs := map[string]string{}

	//Generate CSV
	HostArgs["generate-csv"] = "test.csv"
	_, err := s.createHost(project, HostArgs)
	s.NoError(err)

	//Dry run host creation
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
		"dry-run":         "true",
	}
	_, err = s.createHost(project, HostArgs)
	s.NoError(err)

	//Dry run host creation wrong file
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.lol",
		"dry-run":         "true",
	}
	_, err = s.createHost(project, HostArgs)
	s.EqualError(err, "host import input file must be a CSV file")

	//Dry run host creation with overrides
	HostArgs = map[string]string{
		"import-from-csv":  "./testdata/mock.csv",
		"dry-run":          "true",
		"site":             "site-abcd1111",
		"secure":           "true",
		"remote-user":      "user",
		"os-profile":       "microvisor-nonrt",
		"metadata":         "key1=value1",
		"cloud-init":       "custom",
		"cluster-deploy":   "true",
		"cluster-config":   "role:all;name:mycluster;labels:sample-label=samplevalue&sample-label2=samplevalue",
		"cluster-template": "baseline:v2.0.2",
	}
	_, err = s.createHost(project, HostArgs)
	s.NoError(err)

	//host creation
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
	}
	_, err = s.createHost(project, HostArgs)
	s.NoError(err)

	// Host creation with invalid project
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
	}
	_, err = s.createHost("invalid-project", HostArgs)
	s.Error(err)

	// Host creation with minimal CSV (no overrides)
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/minimal.csv",
	}
	_, err = s.createHost(project, HostArgs)
	s.NoError(err)
	fmt.Println("Host creation tests completed successfully.")

	// Host creation with duplicate host scenario
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
	}
	_, err = s.createHost("duplicate-host-project", HostArgs)
	s.Error(err)
	fmt.Println("Host creation with duplicates completed successfully.")

	// Host creation with no site
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
		"site":            "",
	}
	_, err = s.createHost(project, HostArgs)
	s.EqualError(err, "Pre-flight check failed")

	// Host creation with invaid security setting
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
		"secure":          "true",
		"os-profile":      "microvisor-rt",
	}
	_, err = s.createHost(project, HostArgs)
	s.EqualError(err, "Failed to provision hosts")

	//Dry run host creation with wrong template
	HostArgs = map[string]string{
		"import-from-csv":  "./testdata/mock.csv",
		"cluster-deploy":   "true",
		"cluster-config":   "role:all;name:mycluster;labels:sample-label=samplevalue&sample-label2=samplevalue",
		"cluster-template": "nonexistent-template:v2.0.2",
	}
	_, err = s.createHost(project, HostArgs)
	s.EqualError(err, "Failed to provision hosts")

	// Test list hosts functionality
	listOutput, err := s.listHost(project, make(map[string]string))
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Resource ID":         resourceID,
			"Name":                name,
			"Host Status":         hostStatus,
			"Provisioning Status": provisioningStatus,
			"Serial Number":       serialNumber,
			"Operating System":    operatingSystem,
			"Site ID":             siteID,
			"Site Name":           siteName,
			"Workload":            workload,
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	// Test list hosts  verbose functionality
	HostArgs = map[string]string{
		"verbose": "true",
	}
	listOutput, err = s.listHost(project, HostArgs)
	s.NoError(err)

	parsedOutputList = mapListOutput(listOutput)

	expectedOutputList = listCommandOutput{
		{
			"Resource ID":         resourceID,
			"Name":                name,
			"Host Status":         hostStatus,
			"Provisioning Status": provisioningStatus,
			"Serial Number":       serialNumber,
			"Operating System":    operatingSystem,
			"Site ID":             siteID,
			"Site Name":           siteName,
			"Workload":            workload,
			"Host ID":             name,
			"UUID":                uuid,
			"Processor":           processor,
			"Available Update":    update,
			"Trusted Compute":     compute,
		},
		{
			"Resource ID": "Total Hosts: 1",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	// Test list hosts with invalid project
	_, err = s.listHost("nonexistent-project", make(map[string]string))
	s.Error(err)

	// Test list hosts functionality with site filter
	HostArgs = map[string]string{
		"site":     "site-7ceae560",
		"region":   "region-abcd1234",
		"filter":   "filter=0",
		"workload": "workload-abcd1234",
	}
	_, err = s.listHost(project, HostArgs)
	s.NoError(err)

	// Test list hosts functionality with region filters - non existent site
	HostArgs = map[string]string{
		"region":   "region-abcd1234",
		"workload": "NotAssigned",
	}
	_, err = s.listHost("nonexistent-site", HostArgs)
	s.EqualError(err, "no site was found in provided region")

	// Test list hosts functionality with region filters -existent site
	HostArgs = map[string]string{
		"region":   "region-abcd1234",
		"workload": "NotAssigned",
		"filter":   "filter=0",
	}
	_, err = s.listHost(project, HostArgs)
	s.NoError(err)

	// Test get specific host
	hostID := resourceID
	getOutput, err := s.getHost(project, hostID, make(map[string]string))
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Host Info:":            "",
		"-   Host Resurce ID:":  "host-abc12345",
		"-   Name:":             "edge-host-001",
		"-   OS Profile:":       "",
		"Status details:":       "",
		"-   Host Status:":      "Running",
		"-   Update Status:":    "",
		"Specification:":        "",
		"-   Serial Number:":    "1234567890",
		"-   UUID:":             "550e8400-e29b-41d4-a716-446655440000",
		"-   OS:":               "",
		"-   BIOS Vendor:":      "Lenovo",
		"-   Product Name:":     "ThinkSystem SR650",
		"Customizations:":       "",
		"-   Custom configs:":   "",
		"CPU Info:":             "",
		"-   CPU Model:":        "Intel(R) Xeon(R) CPU E5-2670 v3",
		"-   CPU Cores:":        "8",
		"-   CPU Architecture:": "x86_64",
		"-   CPU Threads:":      "32",
		"-   CPU Sockets:":      "2",
	}

	// // DEBUG: Print parsed output
	// fmt.Printf("=== DEBUG: Parsed output ===\n")
	// if len(parsedOutput) == 0 {
	// 	fmt.Printf("  (empty parsed output)\n")
	// } else {
	// 	for key, value := range parsedOutput {
	// 		fmt.Printf("  '%s': '%s'\n", key, value)
	// 	}
	// }
	// fmt.Printf("=== END DEBUG ===\n")
	// // DEBUG: Print expected output
	// fmt.Printf("=== DEBUG: Expected output ===\n")
	// for key, value := range expectedOutput {
	// 	fmt.Printf("  '%s': '%s'\n", key, value)
	// }
	// fmt.Printf("=== END DEBUG ===\n")

	s.compareGetOutput(expectedOutput, parsedOutput)

	// Test get host with invalid project
	_, err = s.getHost("invalid-project", hostID, make(map[string]string))
	s.Error(err)

	// Test get host with non-existent host
	_, err = s.getHost(project, "host-11111111", make(map[string]string))
	s.EqualError(err, "error getting Host")

	// Test get host with non-existent instance
	_, err = s.getHost("invalid-instance", hostID, make(map[string]string))
	s.EqualError(err, "error getting instance of a host:[Internal Server Error]")

	// Test deauthorize host
	_, err = s.deauthorizeHost(project, hostID, make(map[string]string))
	s.NoError(err)

	// Test deauthorize host with invalid project
	_, err = s.deauthorizeHost("invalid-project", hostID, make(map[string]string))
	s.Error(err)

	// Test deauthorize host with non-existent host
	_, err = s.deauthorizeHost(project, "host-11111111", make(map[string]string))
	s.Error(err)

	// Test delete host
	_, err = s.deleteHost(project, hostID, make(map[string]string))
	s.NoError(err)

	// Test delete host with invalid project
	_, err = s.deleteHost("invalid-project", hostID, make(map[string]string))
	s.Error(err)

	// Test delete host with non-existent host
	_, err = s.deleteHost(project, "host-11111111", make(map[string]string))
	s.Error(err)
}
