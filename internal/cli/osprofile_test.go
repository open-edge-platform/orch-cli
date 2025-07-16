// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
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
	expectedOsResourceID := "test-os-resource-id"
	expectedImageID := "3.0.20250504"
	expectedImageURL := "files-edge-orch/repository/microvisor/non_rt/artifact.raw.gz"
	expectedOsType := "OPERATING_SYSTEM_TYPE_IMMUTABLE"
	expectedOsProvider := "OPERATING_SYSTEM_PROVIDER_INFRA"
	expectedPlatformBundle := ""
	expectedSHA := "abc123def456"
	expectedProfileVersion := "3.0.20250504"
	expectedKernelCommand := "console=ttyS0, root=/dev/sda1"
	expectedUpdateSources := "&[https://updates.example.com]"
	expectedInstalledPackages := "wget\\ncurl\\nvim"
	expectedTimestamp := "2025-01-15 10:30:00 +0000 UTC"
	path := "./testdata/osprofile.yaml"
	OSPArgs := map[string]string{}

	//Test OSProfile Creation
	_, err := s.createOSProfile(project, path, OSPArgs)
	s.NoError(err)

	//Invalid profile path
	path = "./testdata/sadasd.yaml"
	_, err = s.createOSProfile(project, path, OSPArgs)
	s.EqualError(err, "file does not exist: ./testdata/sadasd.yaml")

	//Invalid profile format
	path = "./testdata/osprofile.blob"
	_, err = s.createOSProfile(project, path, OSPArgs)
	s.EqualError(err, "os Profile input must be a yaml file")

	//Invalid endpoint
	path = "./testdata/osprofile.yaml"
	_, err = s.createOSProfile("nonexistent-project", path, OSPArgs)
	s.EqualError(err, "os Profile input must be a yaml file")

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

	OSPArgs["verbose"] = "true"
	listOutput, err = s.listOSProfile(project, OSPArgs)
	s.NoError(err)

	parsedOutput := mapGetOutput(listOutput)
	expectedOutput := map[string]string{
		"Name":             name,
		"Profile Name":     expectedProfileName,
		"Security Feature": expectedSecurityFeature,
		"Architecture":     expectedArchitecture,
		"Repository URL":   expectedRepoURL,
		"sha256":           expectedSHA,
		"Kernel Command":   expectedKernelCommand,
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

	// Test Getting OSProfile

	OSPArgs = map[string]string{}
	getOutput, err := s.getOSProfile(project, name, OSPArgs)
	s.NoError(err)

	parsedOutput = mapGetOutput(getOutput)
	expectedOutput = map[string]string{
		"OS Profile Field":   "Value",
		"Name":               name,
		"Profile Name":       expectedProfileName,
		"OS Resource ID":     expectedOsResourceID,
		"version":            expectedProfileVersion,
		"sha256":             expectedSHA,
		"Image ID":           expectedImageID,
		"Image URL":          expectedImageURL,
		"Repository URL":     expectedRepoURL,
		"Security Feature":   expectedSecurityFeature,
		"Architecture":       expectedArchitecture,
		"OS type":            expectedOsType,
		"OS provider":        expectedOsProvider,
		"Platform Bundle":    expectedPlatformBundle,
		"Update Sources":     expectedUpdateSources,
		"Installed Packages": expectedInstalledPackages,
		"Created":            expectedTimestamp,
		"Updated":            expectedTimestamp,
	}

	_, err = s.getOSProfile(project, "random", OSPArgs)
	s.EqualError(err, "no os profile matches the given name")

	//Test deleting OSProfile

	s.compareGetOutput(expectedOutput, parsedOutput)
	_, err = s.deleteOSProfile(project, name, OSPArgs)
	s.NoError(err)

	_, err = s.deleteOSProfile(project, "random", OSPArgs)
	s.EqualError(err, "no os profile matches the given name")
}
