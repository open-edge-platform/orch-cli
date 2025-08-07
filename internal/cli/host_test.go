// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/open-edge-platform/cli/pkg/rest/infra"
)

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

func (s *CLITestSuite) setHost(publisher string, hostID string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`set host %s --project %s`, hostID, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) filterTest() {
	testCases := []struct {
		input    string
		expected *string
	}{
		{"", nil},
		{"onboarded", stringPtr("hostStatus='onboarded'")},
		{"registered", stringPtr("hostStatus='registered'")},
		{"provisioned", stringPtr("hostStatus='provisioned'")},
		{"deauthorized", stringPtr("hostStatus='invalidated'")},
		{"not connected", stringPtr("hostStatus=''")},
		{"error", stringPtr("hostStatus='error'")},
		{"unknown", stringPtr("unknown")},
	}

	for _, tc := range testCases {
		result := filterHelper(tc.input)
		if tc.expected == nil {
			s.Nil(result)
		} else {
			s.Equal(*tc.expected, *result)
		}
	}
}
func (s *CLITestSuite) testResolvePower() {
	tests := []struct {
		input    string
		expected infra.PowerState
		wantErr  bool
	}{
		{"on", infra.POWERSTATEON, false},
		{"off", infra.POWERSTATEOFF, false},
		{"cycle", infra.POWERSTATEPOWERCYCLE, false},
		{"hibernate", infra.POWERSTATEHIBERNATE, false},
		{"reset", infra.POWERSTATERESET, false},
		{"sleep", infra.POWERSTATESLEEP, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tc := range tests {
		result, err := resolvePower(tc.input)
		if tc.wantErr {
			s.Error(err, "expected error for input %q", tc.input)
		} else {
			s.NoError(err, "unexpected error for input %q", tc.input)
			s.Equal(tc.expected, result, "unexpected result for input %q", tc.input)
		}
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

func (s *CLITestSuite) TestHost() {

	resourceID := "host-abc12345"
	name := "edge-host-001"
	hostStatus := "Running"
	provisioningStatus := "PROVISIONING_STATUS_COMPLETED"
	serialNumber := "1234567890"
	operatingSystem := "\"Edge Microvisor Toolkit 3.0.20250504\""
	siteID := "\"site-abcd1234\""
	siteName := "\"site\""
	workload := "\"Edge Kubernetes Cluster\""
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	processor := "Intel(R) Xeon(R) CPU E5-2670 v3"
	update := "No update"
	compute := "Not compatible"

	//hostID := "host-abc12345"
	HostArgs := map[string]string{}

	///////////////////////////////////
	// Host Creation Tests
	///////////////////////////////////

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

	// Host creation with duplicate cluster scenario
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
	}
	_, err = s.createHost("duplicate-cluster-project", HostArgs)
	s.Error(err)

	// Host creation with duplicate host scenario
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
	}
	_, err = s.createHost("duplicate-host-project", HostArgs)
	s.Error(err)

	// Host creation with no site
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
		"site":            "",
	}
	_, err = s.createHost(project, HostArgs)
	// Accept either error message as valid
	s.True(err != nil && (err.Error() == "Pre-flight check failed" ||
		err.Error() == "--import-from-csv <path/to/file.csv> is required, cannot be empty"),
		"Expected either pre-flight check failure or missing CSV error, got: %v", err)

	// Host creation with wrong cloud init
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
		"cloud-init":      "init",
	}
	_, err = s.createHost("nonexistent-init", HostArgs)
	s.EqualError(err, "Failed to provision hosts")

	// Host creation with wrong user
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
		"remote-user":     "init",
	}
	_, err = s.createHost("nonexistent-user", HostArgs)
	s.EqualError(err, "Failed to provision hosts")

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

	////////////////////////////////
	// Test list hosts functionality
	////////////////////////////////

	//Test filter
	s.filterTest()

	s.testResolvePower()

	// Test list hosts with no filters
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
		"Host Info:":                   "",
		"-   Host Resurce ID:":         "host-abc12345",
		"-   Name:":                    "edge-host-001",
		"-   OS Profile:":              "Edge Microvisor Toolkit 3.0.20250504",
		"-   Host Status Details:":     "INSTANCE_STATUS_RUNNING",
		"-   Provisioning Status:":     "PROVISIONING_STATUS_COMPLETED",
		"Status details:":              "",
		"-   Host Status:":             "Running",
		"-   Update Status:":           "",
		"-   NIC Name and IP Address:": "eth0 192.168.1.102;",
		"Specification:":               "",
		"-   Serial Number:":           "1234567890",
		"-   UUID:":                    "550e8400-e29b-41d4-a716-446655440000",
		"-   OS:":                      "Edge Microvisor Toolkit 3.0.20250504",
		"-   BIOS Vendor:":             "Lenovo",
		"-   Product Name:":            "ThinkSystem SR650",
		"Customizations:":              "",
		"-   Custom configs:":          "nginx-config",
		"CPU Info:":                    "",
		"-   CPU Model:":               "Intel(R) Xeon(R) CPU E5-2670 v3",
		"-   CPU Cores:":               "8",
		"-   CPU Architecture:":        "x86_64",
		"-   CPU Threads:":             "32",
		"-   CPU Sockets:":             "2",
		"AMT Info:":                    "",
		"-   AMT Status:":              "AMT_STATE_PROVISIONED",
		"-   Current Power Status:":    "POWER_STATE_ON",
		"-   Desired Power Status:":    "POWER_STATE_ON",
		"-   Power Command Policy :":   "POWER_COMMAND_POLICY_ALWAYS_ON",
		"-   PowerOn Time :":           "1",
		"-   Affected Packages:":       "[fluent-bit-3.1.9-11.emt3.x86_64]",
		"-   CVE ID:":                  "CVE-2021-1234",
		"-   Priority:":                "HIGH",
		"CVE Info (existing CVEs):":    "",
	}

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

	HostArgs = map[string]string{
		"power-policy": "ordered",
		"power":        "off",
	}

	// Test set host with non-existent host
	_, err = s.setHost(project, "host-11111111", HostArgs)
	s.Error(err)

	// Test set host with host
	_, err = s.setHost(project, hostID, HostArgs)
	s.NoError(err)

	HostArgs = map[string]string{
		"power-policy": "immediate",
		"power":        "on",
	}

	// Test set host with host
	_, err = s.setHost(project, hostID, HostArgs)
	s.NoError(err)

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

func FuzzHost(f *testing.F) {
	// Initial corpus with basic input
	f.Add("project", "./testdata/mock.csv", "", "", "", "", "", "", "", "", "", "", "host-abcd1234", "on", "immediate")
	f.Add("project", "./testdata/mock.csv", "user", "site-abcd1234", "true", "os-abcd1234", "key=value", "true", "template:version", "role:all", "config1&config2", "true", "host-abcd1234", "", "")
	f.Add("project", "./testdata/mock.csv", "user", "site-abcd1234", "true", "os-abcd1234", "key=value", "true", "template:version", "role:all", "config1&config2", "true", "", "on", "immediate")
	f.Fuzz(func(t *testing.T, project string, path string, remoteUser string, site string, secure string, osProfile string, metadata string, clusterDeploy string, clusterTemplate string, clusterConfig string, cloudInit string, amt string, name string, pwr string, pol string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t) // Set the testing.T instance
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()

		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		//Fuzz create host command
		// Host arguments with override flags
		HostArgs := map[string]string{
			"import-from-csv":  path,
			"remote-user":      remoteUser,
			"site":             site,
			"secure":           secure,
			"os-profile":       osProfile,
			"metadata":         metadata,
			"cluster-deploy":   clusterDeploy,
			"cluster-template": clusterTemplate,
			"cluster-config":   clusterConfig,
			"cloud-init":       cloudInit,
			"amt":              amt,
		}

		expErr1 := "--import-from-csv <path/to/file.csv> is required, cannot be empty"
		expErr2 := "host import input file must be a CSV file"
		expErr3 := "Failed to provision hosts"
		expErr4 := "Pre-flight check failed"

		_, err := testSuite.createHost(project, HostArgs)

		if path == "" || strings.TrimSpace(path) == "" {
			if !testSuite.Error(err) {
				t.Errorf("Unexpected result for path %s", path)
			}
		} else if !PathExists(path) || !HasCSVExtension(path) {
			if !testSuite.Error(err) {
				t.Errorf("Unexpected result for path %s", path)
			}
		} else if err != nil && (strings.Contains(err.Error(), expErr1) || strings.Contains(err.Error(), expErr2) || strings.Contains(err.Error(), expErr3) || strings.Contains(err.Error(), expErr4)) {
			if !testSuite.Error(err) {
				t.Errorf("Unexpected result for path %s", path)
			}
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected result for path %s", path)
		}

		//Fuzz set host command
		HostArgs = map[string]string{
			"power":        pwr,
			"power-policy": pol,
		}

		expErr1 = "incorrect power policy provided with --power-policy flag use one of immediate|ordered"
		expErr2 = "accepts 1 arg(s), received 2"
		expErr3 = "incorrect power action provided with --power flag use one of on|off|cycle|hibernate|reset|sleep"

		_, err = testSuite.setHost(project, name, HostArgs)

		if (pwr == "" || strings.TrimSpace(pwr) == "") || (pol == "" || strings.TrimSpace(pol) == "") || (name == "" || strings.TrimSpace(name) == "") {
			if !testSuite.Error(err) {
				t.Errorf("Unexpected result for %s for power %s or policy %s", name, pwr, pol)
			}
		} else if err != nil && (strings.Contains(err.Error(), expErr1) || strings.Contains(err.Error(), expErr2) || strings.Contains(err.Error(), expErr3)) {

		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected result for %s power %s or policy %s", name, pwr, pol)
		}

		// --- Get Host ---
		_, err = testSuite.getHost(project, name, make(map[string]string))
		if name == "" || strings.TrimSpace(name) == "" {
			if !testSuite.Error(err) {
				t.Errorf("Expected error for missing host name in getHost, got: %v", err)
			}
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for getHost with name %s: %v", name, err)
		}

		// --- Delete Host ---
		_, err = testSuite.deleteHost(project, name, make(map[string]string))
		if name == "" || strings.TrimSpace(name) == "" {
			if !testSuite.Error(err) {
				t.Errorf("Expected error for missing host name in deleteHost, got: %v", err)
			}
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for deleteHost with name %s: %v", name, err)
		}

		// --- List Host ---
		_, err = testSuite.listHost(project, make(map[string]string))
		if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for listHost with project %s: %v", project, err)
		}

	})
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func HasCSVExtension(path string) bool {
	return strings.HasSuffix(path, ".csv")
}
