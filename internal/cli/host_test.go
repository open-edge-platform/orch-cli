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

	s.testResolveAmtState()

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

	// Add this debug print to see raw output
	fmt.Printf("=== RAW OUTPUT ===\n%s\n=== END RAW OUTPUT ===\n", getOutput)

	parsedOutput := mapGetOutput(getOutput)

	// Add this to see parsed output
	fmt.Printf("=== PARSED OUTPUT ===\n")
	for k, v := range parsedOutput {
		fmt.Printf("Key: %q -> Value: %q\n", k, v)
	}
	fmt.Printf("=== END PARSED OUTPUT ===\n")
	expectedOutput := map[string]string{
		"Detailed Host Information":       "",
		"Host Info:":                      "",
		"-   Host Resurce ID:":            "host-abc12345",
		"-   Name:":                       "edge-host-001",
		"-   OS Profile:":                 "Edge Microvisor Toolkit 3.0.20250504",
		"-   Host Status Details:":        "INSTANCE_STATUS_RUNNING",
		"-   Provisioning Status:":        "PROVISIONING_STATUS_COMPLETED",
		"-   OS Update Policy:":           "",
		"Status details:":                 "",
		"-   Host Status:":                "Running",
		"-   Update Status:":              "",
		"-   NIC Name and IP Address:":    "eth0 192.168.1.102;",
		"-   LVM Size:":                   "10 GB",
		"Specification:":                  "",
		"-   Serial Number:":              "1234567890",
		"-   UUID:":                       "550e8400-e29b-41d4-a716-446655440000",
		"-   OS:":                         "Edge Microvisor Toolkit 3.0.20250504",
		"-   BIOS Vendor:":                "Lenovo",
		"-   Product Name:":               "ThinkSystem SR650",
		"Customizations:":                 "",
		"-   Custom configs:":             "nginx-config",
		"CPU Info:":                       "",
		"Model":                           "Cores   |Architecture   |Threads   |Sockets",
		"Intel(R) Xeon(R) CPU E5-2670 v3": "8       |x86_64         |32        |2",
		"Memory Info:":                    "",
		"Total (GB)":                      "",
		"16":                              "",
		"Storage Info:":                   "",
		"WWID":                            "Capacity   |Model    |Serial   |Vendor",
		"abcd":                            "0 GB       |Model1   |123456   |Vendor1",
		"GPU Info:":                       "",
		"Device":                          "Vendor       |Capabilities   |PCI Address",
		"TestGPU":                         "TestVendor   |cap1,cap2      |03:00.0",
		"USB Info:":                       "",
		"Class":                           "Serial   |Vendor ID   |Product ID   |Bus   |Address",
		"Hub":                             "123456   |abcd        |1234         |8     |1",
		"Interfaces Info:":                "",
		"Name":                            "Links State   |MTU      |MAC Address         |PCI Identifier   |SRIOV   |SRIOV VF Total   |SRIOV VF Number   |BMC Interface",
		"eth0":                            "UNSPECIFIED   |1500     |30:d0:42:d9:02:7c   |0000:19:00.0     |true    |8                |4                 |true",
		"AMT Info:":                       "",
		"-   AMT Status:":                 "AMT_STATE_PROVISIONED",
		"-   Current Power Status:":       "POWER_STATE_ON",
		"-   Desired Power Status:":       "POWER_STATE_ON",
		"-   Power Command Policy :":      "POWER_COMMAND_POLICY_ALWAYS_ON",
		"-   PowerOn Time :":              "1",
		"-   Desired AMT State :":         "AMT_STATE_PROVISIONED",
		"CVE Info (existing CVEs):":       "",
		"-   CVE ID:":                     "CVE-2021-1234",
		"-   Affected Packages:":          "[fluent-bit-3.1.9-11.emt3.x86_64]",
		"-   Priority:":                   "HIGH",
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

	// --- CSV Generation Test ---
	os.Remove("test_output.csv")
	HostArgs = map[string]string{
		"generate-csv": "test_output.csv",
	}
	_, err = s.setHost(project, "", HostArgs)
	files, _ := os.ReadDir(".")
	for _, f := range files {
		fmt.Println("File:", f.Name())
	}
	s.NoError(err)
	s.True(PathExists("test_output.csv"), "CSV file was not generated")
	defer os.Remove("test_output.csv")

	// --- CSV Import Test ---
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
