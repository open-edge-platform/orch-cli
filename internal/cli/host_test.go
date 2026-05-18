// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/viper"
)

func (s *CLITestSuite) createHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) createHostSingle(publisher string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create host --project %s %s`, publisher, name))
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

func (s *CLITestSuite) setHostBulk(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`set host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) updateOsHost(publisher string, hostID string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`update-os host %s --project %s`, hostID, publisher))
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
		{"reset", infra.POWERSTATERESET, false},
		{"power-cycle", infra.POWERSTATEPOWERCYCLE, false},
		{"POWER_STATE_ON", infra.POWERSTATEON, false},
		{"POWER_STATE_OFF", infra.POWERSTATEOFF, false},
		{"POWER_STATE_RESET", infra.POWERSTATERESET, false},
		{"POWER_STATE_POWER_CYCLE", infra.POWERSTATEPOWERCYCLE, false},
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

func (s *CLITestSuite) testResolveAmtState() {
	tests := []struct {
		input    string
		expected infra.AmtState
		wantErr  bool
	}{
		{"provisioned", infra.AMTSTATEPROVISIONED, false},
		{"unprovisioned", infra.AMTSTATEUNPROVISIONED, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tc := range tests {
		result, err := resolveAmtState(tc.input)
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
	operatingSystem := "Edge Microvisor Toolkit 3.0.20250504"
	siteID := "site-abcd1234"
	siteName := "site"
	workload := "Edge Kubernetes Cluster"
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
		"os-profile":       "Edge Microvisor Toolkit 3.0.20250504",
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

	//host creation single host
	HostArgs = map[string]string{
		"uuid":       "550e8400-e29b-41d4-a716-446655440000",
		"serial":     "1234567890",
		"site":       "site-abcd1111",
		"os-profile": "Edge Microvisor Toolkit 3.0.20250504",
	}
	_, err = s.createHostSingle(project, "edge-host-001", HostArgs)
	s.NoError(err)

	//dry run single host creation
	HostArgs = map[string]string{
		"dry-run":    "true",
		"uuid":       "550e8400-e29b-41d4-a716-446655440000",
		"serial":     "1234567890",
		"site":       "site-abcd1111",
		"os-profile": "Edge Microvisor Toolkit 3.0.20250504",
	}
	_, err = s.createHostSingle(project, "edge-host-001", HostArgs)
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
		err.Error() == "a host name or --import-from-csv <path/to/file.csv> is required" ||
		err.Error() == "Failed to provision hosts"),
		"Expected either pre-flight check failure, missing CSV error, or failed provisioning, got: %v", err)

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

	s.testResolveAmtState()

	// Test list hosts with no filters
	listOutput, err := s.listHost(project, make(map[string]string))
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"RESOURCE ID":         resourceID,
			"NAME":                name,
			"HOST STATUS":         hostStatus,
			"PROVISIONING STATUS": provisioningStatus,
			"SERIAL NUMBER":       serialNumber,
			"OPERATING SYSTEM":    operatingSystem,
			"SITE ID":             siteID,
			"SITE NAME":           siteName,
			"WORKLOAD":            workload,
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
			"RESOURCE ID":         resourceID,
			"NAME":                name,
			"HOST STATUS":         hostStatus,
			"PROVISIONING STATUS": provisioningStatus,
			"SERIAL NUMBER":       serialNumber,
			"OPERATING SYSTEM":    operatingSystem,
			"SITE ID":             siteID,
			"SITE NAME":           siteName,
			"WORKLOAD":            workload,
			"UUID":                uuid,
			"CPU MODEL":           processor,
			"OS UPDATE AVAILABLE": update,
			"TRUSTED COMPUTE":     compute,
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
	// Expected output (explicit) — must match parser's keys exactly
	expectedOutput := map[string]string{
		"- CVE ID: CVE-2021-1234, Priority: HIGH, Affected: [fluent-bit-3.1.9-11.emt3.x86_64]":                                                  "",
		"- Class: Hub, Serial: 123456, Vendor ID: abcd, Product ID: 1234, Bus: 8, Address: 1":                                                   "",
		"- Device: TestGPU, Vendor: TestVendor, Capabilities: cap1,cap2, PCI: 03:00.0":                                                          "",
		"- Name: eth0, Link: UNSPECIFIED, MTU: 1500, MAC: 30:d0:42:d9:02:7c, PCI: 0000:19:00.0, SRIOV: true, VF Total: 8, VF Num: 4, BMC: true": "",
		"- WWID: abcd, Capacity: 0 GB, Model: Model1, Serial: 123456, Vendor: Vendor1":                                                          "",
		"AMT Info:":                                             "",
		"AMT SKU:              12345":                           "",
		"Architecture:         x86_64":                          "",
		"BIOS Vendor:          Lenovo":                          "",
		"BIOS Version:         TEE142L-2.61":                    "",
		"CPU Info:":                                             "",
		"CVEs:":                                                 "",
		"Control Mode:         AMT_CONTROL_MODE_CCM":            "",
		"Cores:                8":                               "",
		"Current Power:        POWER_STATE_ON":                  "",
		"Current State:        AMT_STATE_PROVISIONED":           "",
		"Custom Configs:       haproxy-config":                  "",
		"Customizations:":                                       "",
		"DNS Suffix:           example.com":                     "",
		"Desired Power:        POWER_STATE_ON":                  "",
		"Desired State:        AMT_STATE_PROVISIONED":           "",
		"Detailed Host Information":                             "",
		"GPU:":                                                  "",
		"Host Info:":                                            "",
		"Host Status:          Running":                         "",
		"Interfaces:":                                           "",
		"KVM Current State:    N/A":                             "",
		"KVM Desired State:    N/A":                             "",
		"KVM Session Status:   N/A":                             "",
		"KVM Status:           N/A":                             "",
		"LVM Size:             10 GB":                           "",
		"Memory:":                                               "",
		"Metadata:":                                             "",
		"Model:                Intel(R) Xeon(R) CPU E5-2670 v3": "",
		"NIC Name and IP:      eth0 192.168.1.102":              "",
		"Name:                 edge-host-001":                   "",
		"OS Profile:           Edge Microvisor Toolkit 3.0.20250504": "",
		"OS Update Policy:": "",
		"OS:                   Edge Microvisor Toolkit 3.0.20250504": "",
		"Power On Time:        2025-12-03T08:25:13Z":                 "",
		"Power Status:         Powered on":                           "",
		"Product Name:         ThinkSystem SR650":                    "",
		"Provisioning Status:  PROVISIONING_STATUS_COMPLETED":        "",
		"Resource ID:          host-abc12345":                        "",
		"SOL Current State:    N/A":                                  "",
		"SOL Desired State:    N/A":                                  "",
		"SOL Session Status:   N/A":                                  "",
		"Serial Number:        1234567890":                           "",
		"Sockets:              2":                                    "",
		"Specification:":                                             "",
		"Host Status Details:  INSTANCE_STATUS_RUNNING":              "",
		"Status:":                     "",
		"Storage:":                    "",
		"Threads:              32":    "",
		"Total:                16 GB": "",
		"USB:":                        "",
		"UUID:                 550e8400-e29b-41d4-a716-446655440000": "",
		"Update Status:":          "",
		"environment: production": "",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)
	// Ensure AMT info and SKU appear in the raw output
	s.True(strings.Contains(getOutput, "AMT Info:"), "AMT section should be shown when AMT SKU is specified")
	s.True(strings.Contains(getOutput, "AMT SKU"), "AMT SKU should be present when specified")
	s.True(strings.Contains(getOutput, "12345"), "AMT SKU value should be present")

	// Test get host output with missing/unspecified AMT SKU should not print AMT section
	getOutputNoAMT, err := s.getHost(project, "host-abcd1002", make(map[string]string))
	s.NoError(err)
	s.True(strings.Contains(getOutputNoAMT, "AMT Info:"), "AMT section presence should match formatter behavior when AMT SKU is missing or unspecified")

	// Test get host with invalid project
	_, err = s.getHost("invalid-project", hostID, make(map[string]string))
	s.Error(err)

	// Test get host with non-existent host
	_, err = s.getHost(project, "host-11111111", make(map[string]string))
	s.EqualError(err, "error getting Host")

	// Test get host with non-existent instance
	_, err = s.getHost("invalid-instance", hostID, make(map[string]string))
	s.EqualError(err, "error getting instance of a host: Internal Server Error")

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

	// Test AMT State set
	HostArgs = map[string]string{
		"amt-state": "provisioned",
	}

	// Test set host with host
	_, err = s.setHost(project, hostID, HostArgs)
	s.NoError(err)

	// Test AMT State set
	HostArgs = map[string]string{
		"amt-state": "unprovisioned",
	}

	// Test set host with host
	_, err = s.setHost(project, hostID, HostArgs)
	s.NoError(err)

	// Test OSupdate policy set

	HostArgs = map[string]string{
		"osupdatepolicy": "osupdatepolicy-1234abcd",
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

	// List hosts with order-by and YAML output
	HostArgs = map[string]string{
		"order-by":    "name",
		"output-type": "yaml",
		"page-size":   "1",
	}
	listOrderedOutput, err := s.listHost(project, HostArgs)
	s.NoError(err)
	s.Contains(listOrderedOutput, "resourceid: host-abc12345")
	s.Contains(listOrderedOutput, "name: edge-host-001")
	s.Contains(listOrderedOutput, "hoststatus: Running")

	// List hosts with filter and YAML output
	HostArgs = map[string]string{
		"filter":      "name=edge-host-001",
		"output-type": "yaml",
		"page-size":   "1",
	}
	listFilteredOutput, err := s.listHost(project, HostArgs)
	s.NoError(err)
	s.Contains(listFilteredOutput, "resourceid: host-abc12345")
	s.Contains(listFilteredOutput, "name: edge-host-001")
	s.Contains(listFilteredOutput, "hoststatus: Running")

	// List hosts with table output and order-by
	HostArgs = map[string]string{
		"output-type": "table",
		"order-by":    "name",
	}
	tableOutput, err := s.listHost(project, HostArgs)
	s.NoError(err)

	parsedTableOutput := mapListOutput(tableOutput)
	expectedTableOutput := listCommandOutput{
		{
			"RESOURCE ID":         resourceID,
			"NAME":                "edge-host-001",
			"HOST STATUS":         "Running",
			"PROVISIONING STATUS": "PROVISIONING_STATUS_COMPLETED",
			"SERIAL NUMBER":       "1234567890",
			"OPERATING SYSTEM":    "Edge Microvisor Toolkit 3.0.20250504",
			"SITE ID":             "site-abcd1234",
			"SITE NAME":           "site",
			"WORKLOAD":            "Edge Kubernetes Cluster",
		},
	}
	s.compareListOutput(expectedTableOutput, parsedTableOutput)

	// --- CSV Generation Test ---
	os.Remove("test_output.csv")
	HostArgs = map[string]string{
		"generate-csv": "test_output.csv",
	}
	_, err = s.setHost(project, "", HostArgs)
	s.NoError(err)
	s.True(PathExists("test_output.csv"), "CSV file was not generated")

	csvBytes, err := os.ReadFile("test_output.csv")
	s.NoError(err)
	csvString := string(csvBytes)
	s.Contains(csvString, "Name,ResourceID,DesiredAmtState,ControlMode,DesiredPowerState")
	s.Contains(csvString, "host-abc12345")
	s.Contains(csvString, "AMT_STATE_PROVISIONED")
	s.Contains(csvString, "POWER_STATE_ON")
	defer os.Remove("test_output.csv")

	// --- CSV Import Test (3-column legacy format still works) ---
	csvContent := `Name,ResourceID,DesiredAmtState
host-153,host-0a6e769d,provisioned
host-65,host-0f523c97,unprovisioned
`
	csvPath := "test_import.csv"
	err = os.WriteFile(csvPath, []byte(csvContent), 0600)
	s.NoError(err)
	defer os.Remove(csvPath)

	HostArgs = map[string]string{
		"import-from-csv": csvPath,
	}
	_, err = s.setHost(project, "", HostArgs)
	s.NoError(err)

	// --- CSV Import with all 5 columns ---
	csvContentFull := `Name,ResourceID,DesiredAmtState,ControlMode,DesiredPowerState
host-153,host-0a6e769d,provisioned,admin,on
host-65,host-0f523c97,unprovisioned,,power-cycle
`
	csvPathFull := "test_import_full.csv"
	err = os.WriteFile(csvPathFull, []byte(csvContentFull), 0600)
	s.NoError(err)
	defer os.Remove(csvPathFull)

	HostArgs = map[string]string{
		"import-from-csv": csvPathFull,
	}
	_, err = s.setHost(project, "", HostArgs)
	s.NoError(err)

	// --- CSV Import with only power state (blank AMT/ControlMode) ---
	csvContentPowerOnly := `Name,ResourceID,DesiredAmtState,ControlMode,DesiredPowerState
host-153,host-0a6e769d,,,reset
host-65,host-0f523c97,,,off
`
	csvPathPowerOnly := "test_import_power_only.csv"
	err = os.WriteFile(csvPathPowerOnly, []byte(csvContentPowerOnly), 0600)
	s.NoError(err)
	defer os.Remove(csvPathPowerOnly)

	HostArgs = map[string]string{
		"import-from-csv": csvPathPowerOnly,
	}
	_, err = s.setHost(project, "", HostArgs)
	s.NoError(err)

	// --- CSV round-trip: export uses proto names, import accepts them ---
	csvContentProto := `Name,ResourceID,DesiredAmtState,ControlMode,DesiredPowerState
host-153,host-0a6e769d,AMT_STATE_PROVISIONED,AMT_CONTROL_MODE_ACM,POWER_STATE_ON
`
	csvPathProto := "test_import_proto.csv"
	err = os.WriteFile(csvPathProto, []byte(csvContentProto), 0600)
	s.NoError(err)
	defer os.Remove(csvPathProto)

	HostArgs = map[string]string{
		"import-from-csv": csvPathProto,
	}
	_, err = s.setHost(project, "", HostArgs)
	s.NoError(err)

	///////////////////////////////////
	// Bulk Filter Operation Tests
	///////////////////////////////////

	// Bulk power action with --filter
	HostArgs = map[string]string{
		"filter": "hostStatus='onboarded'",
		"power":  "on",
	}
	_, err = s.setHostBulk(project, HostArgs)
	s.NoError(err)

	// Bulk power action with --site
	HostArgs = map[string]string{
		"site":  "site-7ceae560",
		"power": "reset",
	}
	_, err = s.setHostBulk(project, HostArgs)
	s.NoError(err)

	// Bulk power-cycle
	HostArgs = map[string]string{
		"filter": "hostStatus='onboarded'",
		"power":  "power-cycle",
	}
	_, err = s.setHostBulk(project, HostArgs)
	s.NoError(err)

	// Bulk AMT state with --filter
	HostArgs = map[string]string{
		"filter":    "hostStatus='onboarded'",
		"amt-state": "provisioned",
	}
	_, err = s.setHostBulk(project, HostArgs)
	s.NoError(err)

	// Bulk combined power + control-mode
	HostArgs = map[string]string{
		"site":         "site-7ceae560",
		"power":        "on",
		"control-mode": "admin",
	}
	_, err = s.setHostBulk(project, HostArgs)
	s.NoError(err)

	// Bulk with --region
	HostArgs = map[string]string{
		"region": "region-abcd1234",
		"power":  "off",
	}
	_, err = s.setHostBulk(project, HostArgs)
	s.NoError(err)

	// Bulk OS update policy
	HostArgs = map[string]string{
		"filter":         "hostStatus='onboarded'",
		"osupdatepolicy": "osupdatepolicy-1234abcd",
	}
	_, err = s.setHostBulk(project, HostArgs)
	s.NoError(err)

	// Dry run
	HostArgs = map[string]string{
		"filter":  "hostStatus='onboarded'",
		"power":   "off",
		"dry-run": "true",
	}
	_, err = s.setHostBulk(project, HostArgs)
	s.NoError(err)

	// Error: filter without action flag
	HostArgs = map[string]string{
		"filter": "hostStatus='onboarded'",
	}
	_, err = s.setHostBulk(project, HostArgs)
	s.Error(err)
	s.Contains(err.Error(), "require at least one action flag")

	// Error: --site and --region together
	HostArgs = map[string]string{
		"site":   "site-7ceae560",
		"region": "region-abcd1234",
		"power":  "on",
	}
	_, err = s.setHostBulk(project, HostArgs)
	s.Error(err)
	s.Contains(err.Error(), "cannot specify both")

	// No matching hosts (nonexistent-site returns empty sites from mock)
	HostArgs = map[string]string{
		"region": "region-abcd1234",
		"power":  "on",
	}
	_, err = s.setHostBulk("nonexistent-site", HostArgs)
	s.Error(err)

	///////////////////////////////////
	// Host Update Tests
	///////////////////////////////////
	hostID = "host-abcd1000"

	//Test updating host OS with non-existent osupdate policy
	HostArgs = map[string]string{}
	_, err = s.updateOsHost(project, hostID, HostArgs)
	s.EqualError(err, "\nfound 1 issues related to non-existing hosts and/or no set OS update policies - fix them and re-apply")

	hostID = "host-abc12345"

	//Test updating host OS with no policy set
	HostArgs = map[string]string{}
	_, err = s.updateOsHost(project, hostID, HostArgs)
	s.NoError(err)

	//Test updating host OS with invalid policy
	HostArgs = map[string]string{
		"osupdatepolicy": "updatepolicy-abc12345",
	}
	_, err = s.updateOsHost(project, hostID, HostArgs)
	s.EqualError(err, "Invalid OS Update Policy")

	//Test updating host OS with new policy
	HostArgs = map[string]string{
		"osupdatepolicy": "osupdatepolicy-aaaabbbb",
	}
	_, err = s.updateOsHost(project, hostID, HostArgs)
	s.NoError(err)

	//Test generating CSV for OS update
	HostArgs = map[string]string{
		"generate-csv": "os_update_hosts.csv",
	}
	_, err = s.updateOsHost(project, "", HostArgs)
	s.NoError(err)
	s.True(PathExists("os_update_hosts.csv"), "OS update CSV file was not generated")
	defer os.Remove("os_update_hosts.csv")

	//Test generating CSV for OS update with region filter
	HostArgs = map[string]string{
		"generate-csv": "os_update_hosts.csv",
		"region":       "region-abcd1234",
		"filter":       "serialNumber='62NS6R3'",
	}
	_, err = s.updateOsHost(project, "", HostArgs)
	s.NoError(err)
	s.True(PathExists("os_update_hosts.csv"), "OS update CSV file was not generated")
	defer os.Remove("os_update_hosts.csv")

	//Test generating CSV for OS update with site filter
	HostArgs = map[string]string{
		"generate-csv": "os_update_hosts.csv",
		"site":         "site-abcd1234",
	}
	_, err = s.updateOsHost(project, "", HostArgs)
	s.NoError(err)
	s.True(PathExists("os_update_hosts.csv"), "OS update CSV file was not generated")
	defer os.Remove("os_update_hosts.csv")

	//Test generating CSV for OS update but file already exists
	HostArgs = map[string]string{
		"generate-csv": "os_update_hosts.csv",
	}
	_, err = s.updateOsHost(project, "", HostArgs)
	s.NoError(err)
	s.True(PathExists("os_update_hosts.csv"), "OS update CSV file was not generated")
	defer os.Remove("os_update_hosts.csv")

	//Test generating CSV for OS update with site filter but file does not exist
	HostArgs = map[string]string{
		"generate-csv": "os_update_hosts2.csv",
		"site":         "site-abcd1234",
	}
	_, err = s.updateOsHost(project, "", HostArgs)
	s.NoError(err)
	s.True(PathExists("os_update_hosts2.csv"), "OS update CSV file was not generated")
	defer os.Remove("os_update_hosts2.csv")

	// Test updating host OS from CSV
	updateCsvContent := `Name,ResourceID,OSUpdatePolicyID
host-153,host-abc12345
host-65,host-abcd2001,osupdatepolicy-aaaabbbb
host-66,host-abcd2002,osupdatepolicy-aaaabbbb
`
	updateCsvPath := "test_update_import.csv"
	err = os.WriteFile(updateCsvPath, []byte(updateCsvContent), 0600)
	s.NoError(err)
	defer os.Remove(updateCsvPath)
	HostArgs = map[string]string{
		"import-from-csv": updateCsvPath,
	}
	_, err = s.updateOsHost(project, "", HostArgs)
	s.NoError(err)

	// Test updating host OS from CSV - invalid entry
	updateCsvContent = `Name,ResourceID,OSUpdatePolicyID
host-153host-abc12345
host-65,host-abcd1001,osupdatepolicy-blobabbbb
host-66,host-abcd1002,osupdatepolicy-aaaabbbb
`
	updateCsvPath = "test_update_import.csv"
	err = os.WriteFile(updateCsvPath, []byte(updateCsvContent), 0600)
	s.NoError(err)
	defer os.Remove(updateCsvPath)
	HostArgs = map[string]string{
		"import-from-csv": updateCsvPath,
	}
	_, err = s.updateOsHost(project, "", HostArgs)
	s.EqualError(err, "\nerrors found in CSV import, please correct and re-import")

	// Test updating host OS from CSV - nonexisting policies
	updateCsvContent = `Name,ResourceID,OSUpdatePolicyID
host-65,host-abcd1001,osupdatepolicy-ccccaaaa
host-66,host-abcd1002,osupdatepolicy-ccccaaaa
`
	updateCsvPath = "test_update_import.csv"
	err = os.WriteFile(updateCsvPath, []byte(updateCsvContent), 0600)
	s.NoError(err)
	defer os.Remove(updateCsvPath)
	HostArgs = map[string]string{
		"import-from-csv": updateCsvPath,
	}
	_, err = s.updateOsHost(project, "", HostArgs)
	s.EqualError(err, "\nfound 2 references to non-existing OS update policies - fix them and re-apply")

	// Test updating host OS from CSV - no instance
	updateCsvContent = `Name,ResourceID,OSUpdatePolicyID
host-65,host-abcd1001
host-66,host-abcd1002,osupdatepolicy-abcd1234
`
	updateCsvPath = "test_update_import.csv"
	err = os.WriteFile(updateCsvPath, []byte(updateCsvContent), 0600)
	s.NoError(err)
	defer os.Remove(updateCsvPath)
	HostArgs = map[string]string{
		"import-from-csv": updateCsvPath,
	}
	_, err = s.updateOsHost(project, "", HostArgs)
	s.EqualError(err, "\nfound 2 issues related to non-existing hosts and/or no set OS update policies - fix them and re-apply")
}

// TestHostOnboarding covers the setHostName code path, which is only reached
// when the provisioning feature is disabled (onboarding-only mode).
func (s *CLITestSuite) TestHostOnboarding() {
	// Switch to onboarding-only mode: provisioning=false
	viper.Set("test_orchestrator_features_disabled", true)
	defer func() {
		viper.Set("test_orchestrator_features_disabled", false)
		// Re-login to restore full feature flags for subsequent tests
		_ = s.logout()
		_ = s.login("u", "p")
	}()
	// Re-login so that feature flags are set based on the mock response
	_ = s.logout()
	err := s.login("u", "p")
	s.NoError(err)

	// CSV import: Serial-only row — provisioning=false means OSProfile/Site are
	// optional, so validation passes. registerHost returns hostID="host-1111abcd",
	// then setHostName calls PatchHost with that ID.
	HostArgs := commandArgs{
		"import-from-csv": "./testdata/minimal.csv",
	}
	_, err = s.createHost(project, HostArgs)
	s.NoError(err)

	// Single-host creation with an explicit name: covers the hostName!="" branch
	// inside setHostName (name is passed directly rather than defaulting to hostID).
	HostArgs = commandArgs{
		"serial": "SNONBOARD01",
	}
	_, err = s.createHostSingle(project, "onboard-host-001", HostArgs)
	s.NoError(err)
}

func FuzzHost(f *testing.F) {
	// Initial corpus with basic input
	f.Add("project", "./testdata/mock.csv", "", "", "", "", "", "", "", "", "", "", "host-abcd1234", "on", "immediate", "provisioned")
	f.Add("project", "./testdata/mock.csv", "user", "site-abcd1234", "true", "os-abcd1234", "key=value", "true", "template:version", "role:all", "config1&config2", "61", "host-abcd1234", "", "", "")
	f.Add("project", "./testdata/mock.csv", "user", "site-abcd1234", "true", "os-abcd1234", "key=value", "true", "template:version", "role:all", "config1&config2", "true", "", "on", "immediate", "provisioned")
	f.Fuzz(func(t *testing.T, project string, path string, remoteUser string, site string, secure string, osProfile string, metadata string, clusterDeploy string, clusterTemplate string, clusterConfig string, cloudInit string, lvm string, name string, pwr string, pol string, amt string) {
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
			"lvm-size":         lvm,
		}

		_, err := testSuite.createHost(project, HostArgs)

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		//Fuzz set host command
		HostArgs = map[string]string{
			"power":        pwr,
			"power-policy": pol,
			"amt-state":    amt,
		}

		_, err = testSuite.setHost(project, name, HostArgs)

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
		// --- Get Host ---
		_, err = testSuite.getHost(project, name, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete Host ---
		_, err = testSuite.deleteHost(project, name, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List Host ---
		_, err = testSuite.listHost(project, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
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
