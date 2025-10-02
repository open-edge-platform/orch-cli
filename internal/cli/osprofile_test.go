// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) createOSProfile(project string, path string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create osprofile %s --project %s`, path, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listOSProfile(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list osprofile --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getOSProfile(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get osprofile "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteOSProfile(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete osprofile "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestOSProfile() {
	name := "Edge Microvisor Toolkit 3.0.20250504"
	expectedArchitecture := "x86_64"
	expectedSecurityFeature := "SECURITY_FEATURE_NONE"
	expectedProfileName := "microvisor-nonrt"
	expectedRepoURL := "files-edge-orch/repository/microvisor/non_rt/"
	//expectedOsResourceID := "os-1234abcd"
	//expectedImageID := "3.0.20250504"
	//expectedImageURL := "files-edge-orch/repository/microvisor/non_rt/artifact.raw.gz"
	//expectedOsType := "OPERATING_SYSTEM_TYPE_IMMUTABLE"
	//expectedOsProvider := "OPERATING_SYSTEM_PROVIDER_INFRA"
	//expectedPlatformBundle := ""
	expectedSHA := "abc123def456"
	//expectedProfileVersion := "3.0.20250504"
	expectedKernelCommand := "console=ttyS0, root=/dev/sda1"
	//expectedUpdateSources := "&[https://updates.example.com]"
	//expectedInstalledPackages := "wget\\ncurl\\nvim"
	//SexpectedTimestamp := "2025-01-15 10:30:00 +0000 UTC"
	path := "./testdata/osprofile.yaml"
	OSPArgs := map[string]string{}

	//Test OSProfile Creation
	_, err := s.createOSProfile(project, path, OSPArgs)
	s.NoError(err)

	//Invalid profile path
	path = "./testdata/sadasd.yaml"
	_, err = s.createOSProfile(project, path, OSPArgs)
	s.EqualError(err, "open ./testdata/sadasd.yaml: no such file or directory")

	//Invalid profile format
	path = "./testdata/osprofile.blob"
	_, err = s.createOSProfile(project, path, OSPArgs)
	s.EqualError(err, "os Profile input must be a yaml file")

	//Invalid endpoint (fail at list)
	path = "./testdata/osprofile.yaml"
	_, err = s.createOSProfile("nonexistent-project", path, OSPArgs)
	s.EqualError(err, "Error getting OS profiles: Internal Server Error")

	//Invalid endpoint (fail at get)
	path = "./testdata/osprofile.yaml"
	_, err = s.createOSProfile("invalid-project", path, OSPArgs)
	s.EqualError(err, "error while creating OS Profile from ./testdata/osprofile.yaml: Internal Server Error")

	//Duplicate name
	path = "./testdata/osprofilenameduplicate.yaml"
	_, err = s.createOSProfile(project, path, OSPArgs)
	s.EqualError(err, "OS Profile Edge Microvisor Toolkit 3.0.20250504 already exists")

	// Test Listing OSProfiles
	OSPArgs["filter"] = "osType=OS_TYPE_IMMUTABLE"
	listOutput, err := s.listOSProfile(project, OSPArgs)
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)
	expectedOutputList := listCommandOutput{
		{
			"Name":             name,
			"Architecture":     expectedArchitecture,
			"Security Feature": expectedSecurityFeature,
		},
	}
	s.compareListOutput(expectedOutputList, parsedOutputList)

	//Test Listing OSProfiles with verbose
	OSPArgs["verbose"] = "true"
	listOutput, err = s.listOSProfile(project, OSPArgs)
	s.NoError(err)

	parsedOutput := mapGetOutput(listOutput)
	expectedOutput := map[string]string{
		"Name:":             name,
		"Profile Name:":     expectedProfileName,
		"Security Feature:": expectedSecurityFeature,
		"Architecture:":     expectedArchitecture,
		"Repository URL:":   expectedRepoURL,
		"sha256:":           expectedSHA,
		"Kernel Command:":   expectedKernelCommand,
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

	_, err = s.listOSProfile("nonexistent-project", OSPArgs)
	s.EqualError(err, "error getting OS Profiles:[Internal Server Error]")

	// Test Getting OSProfile

	OSPArgs = map[string]string{}

	//Get os profile
	linesOutput, err := s.getOSProfile(project, name, OSPArgs)
	s.NoError(err)

	parsedLinesOutput := mapLinesOutput(linesOutput)

	expectedLinesOutput := []string{
		"",
		"OS Profile Field       |Value",
		"Name:                  |Edge Microvisor Toolkit 3.0.20250504",
		"Profile Name:          |microvisor-nonrt",
		"OS Resource ID:        |os-1234abcd",
		"version:               |\"3.0.20250504\"",
		"sha256:                |abc123def456",
		"Image ID:              |3.0.20250504",
		"Image URL:             |files-edge-orch/repository/microvisor/non_rt/artifact.raw.gz",
		"Repository URL:        |files-edge-orch/repository/microvisor/non_rt/",
		"Security Feature:      |\"SECURITY_FEATURE_NONE\"",
		"Architecture:          |x86_64",
		"OS type:               |OPERATING_SYSTEM_TYPE_IMMUTABLE",
		"OS provider:           |OPERATING_SYSTEM_PROVIDER_INFRA",
		"Platform Bundle:       |",
		"Update Sources:        |&[https://updates.example.com]",
		"Installed Packages:    |\"wget\\ncurl\\nvim\"",
		"Created:               |2025-01-15 10:30:00 +0000 UTC",
		"Updated:               |2025-01-15 10:30:00 +0000 UTC",
		"",
		"CVE Info:",
		"   | Existing CVEs: ",
		"",
		"-   |   |CVE ID:              | CVE-2021-1234",
		"-   |   |Priority:            | HIGH",
		"-   |   |Affected Packages:   | [fluent-bit-3.1.9-11.emt3.x86_64]",
		"",
		"   | Fixed CVEs: ",
		"",
		"-   |   |CVE ID:              | CVE-2021-5678",
		"-   |   |Priority:            | MEDIUM",
		"-   |   |Affected Packages:   | [curl-7.68.0-1ubuntu2.24]",
		"",
	}

	s.compareLinesOutput(expectedLinesOutput, parsedLinesOutput)

	//Get invalid os profile
	_, err = s.getOSProfile(project, "random", OSPArgs)
	s.EqualError(err, "no os profile matches the given name")

	//Server error sim
	_, err = s.getOSProfile("nonexistent-project", name, OSPArgs)
	s.EqualError(err, "error getting OS Profile:[Internal Server Error]")

	//Test deleting OSProfile

	//Delete profile
	_, err = s.deleteOSProfile(project, name, OSPArgs)
	s.NoError(err)

	//Non existing profile deletion
	_, err = s.deleteOSProfile(project, "random", OSPArgs)
	s.EqualError(err, "no os profile matches the given name")

	//Server error sim
	_, err = s.deleteOSProfile("invalid-project", name, OSPArgs)
	s.EqualError(err, "error deleting OS profile Edge Microvisor Toolkit 3.0.20250504: Internal Server Error")

	//Server error sim list
	_, err = s.deleteOSProfile("nonexistent-project", name, OSPArgs)
	s.EqualError(err, "Error getting OS profiles: Internal Server Error")
}

func FuzzOSProfile(f *testing.F) {
	// Initial corpus with valid and invalid input
	f.Add("project", "./testdata/osprofile.yaml", "Edge Microvisor Toolkit 3.0.20250504")
	f.Add("project", "./testdata/osprofilenameduplicate.yaml", "Edge Microvisor Toolkit 3.0.20250504")
	f.Add("project", "", "Edge Microvisor Toolkit 3.0.20250504")                                       // missing file
	f.Add("project", "./testdata/osprofile.blob", "Edge Microvisor Toolkit 3.0.20250504")              // invalid format
	f.Add("nonexistent-project", "./testdata/osprofile.yaml", "Edge Microvisor Toolkit 3.0.20250504")  // invalid project (list)
	f.Add("invalid-project", "./testdata/osprofile.yaml", "Edge Microvisor Toolkit 3.0.20250504")      // invalid project (create)
	f.Add("project", "./testdata/osprofilenameduplicate.yaml", "Edge Microvisor Toolkit 3.0.20250504") // duplicate name
	f.Add("project", "./testdata/osprofile.yaml", "")                                                  // missing profile name for get/delete

	f.Fuzz(func(t *testing.T, project, path, name string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{}

		// --- Create ---
		_, err := testSuite.createOSProfile(project, path, args)

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List ---
		_, err = testSuite.listOSProfile(project, args)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Get ---
		_, err = testSuite.getOSProfile(project, name, args)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.deleteOSProfile(project, name, args)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
