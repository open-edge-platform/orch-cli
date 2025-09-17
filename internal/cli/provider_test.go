// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"testing"
)

func (s *CLITestSuite) createProvider(project string, name string, kind string, api string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create provider %s %s %s --project %s`, name, kind, api, project))
	fmt.Println("Create command:", commandString)
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listProvider(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list provider --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getProvider(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get provider "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteProvider(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete provider "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestProvider() {

	name := "provider"
	kind := "PROVIDER_KIND_BAREMETAL"
	vendor := "PROVIDER_VENDOR_UNSPECIFIED"
	api := "hello.com"
	resourceID := "provider-7ceae560"

	/////////////////////////////
	// Test Provider Creation
	/////////////////////////////

	//create provider
	SArgs := map[string]string{}
	_, err := s.createProvider(project, name, kind, api, SArgs)
	s.NoError(err)

	SArgs = map[string]string{
		"config": "\"\"defaultOs\":\"os-0921fdc0\",\"autoProvision\":false,\"defaultLocalAccount\":\"\",\"osSecurityFeatureEnable\":false\"",
		"vendor": vendor,
	}
	//create provider with flags
	_, err = s.createProvider(project, name, kind, api, SArgs)
	s.NoError(err)

	//create with invalid vendor
	SArgs = map[string]string{
		"vendor": "invalid",
	}
	_, err = s.createProvider(project, name, kind, api, SArgs)
	s.EqualError(err, "invalid vendor. Accepted values: \"PROVIDER_VENDOR_UNSPECIFIED\", \"PROVIDER_VENDOR_LENOVO_LXCA\", \"PROVIDER_VENDOR_LENOVO_LOCA\"")

	//create with apircreds - improve in future to read from terminal
	SArgs = map[string]string{
		"apicredentials": "",
	}
	_, err = s.createProvider(project, name, kind, api, SArgs)
	s.EqualError(err, "inappropriate ioctl for device")

	/////////////////////////////
	// Test Provider Listing
	/////////////////////////////

	//List provider

	SArgs = map[string]string{}
	listOutput, err := s.listProvider(project, SArgs)
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Name":        name,
			"Resource ID": resourceID,
			"Kind":        kind,
			"Vendor":      vendor,
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	//List site --verbose
	SArgs = map[string]string{
		"verbose": "true",
	}
	listOutput, err = s.listProvider(project, SArgs)
	s.NoError(err)

	parsedOutputList = mapListOutput(listOutput)

	expectedOutputList = listCommandOutput{
		{
			"Name":         name,
			"Resource ID":  resourceID,
			"Kind":         kind,
			"Vendor":       vendor,
			"API Endpoint": api,
			"Created At":   "2025-01-15 10:30:00 +0000 UTC",
			"Updated At":   "2025-01-15 10:30:00 +0000 UTC",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	/////////////////////////////
	// Test Provider Get
	/////////////////////////////

	getOutput, err := s.getProvider(project, resourceID, make(map[string]string))
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Name:":         name,
		"Resource ID:":  resourceID,
		"Kind:":         kind,
		"Vendor:":       vendor,
		"API Endpoint:": api,
		"Config:":       "{\"defaultOs\": \"\", \"autoProvision\": false, \"defaultLocalAccount\": \"\", \"osSecurityFeatureEnable\": false}",
		"Created At:":   "2025-01-15 10:30:00 +0000 UTC",
		"Updated At:":   "2025-01-15 10:30:00 +0000 UTC",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	/////////////////////////////
	// Test Provider Delete
	/////////////////////////////

	// Mock user input for confirmation
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	r, w, errPipe := os.Pipe()
	if errPipe != nil {
		s.T().Fatalf("failed to create pipe: %v", errPipe)
	}
	_, _ = w.Write([]byte("y"))
	w.Close()

	//delete custom config
	os.Stdin = r
	_, err = s.deleteProvider(project, resourceID, make(map[string]string))
	s.NoError(err)

	r, w, errPipe = os.Pipe()
	if errPipe != nil {
		s.T().Fatalf("failed to create pipe: %v", errPipe)
	}
	_, _ = w.Write([]byte("y"))
	w.Close()

	//delete invalid custom config
	os.Stdin = r
	_, err = s.deleteProvider(project, "nonexistent-provider", make(map[string]string))
	s.EqualError(err, "error while deleting provider: Not Found")

	//delete invalid custom config with N as response
	_, err = s.deleteProvider(project, "nonexistent-provider", make(map[string]string))
	s.EqualError(err, "operation cancelled by user")

}

func FuzzProvider(f *testing.F) {
	// Initial corpus with valid and invalid input
	f.Add("project", "provider", "PROVIDER_KIND_BAREMETAL", "hello.com", "{\"defaultOs\":\"\",\"autoProvision\":false,\"defaultLocalAccount\":\"\",\"osSecurityFeatureEnable\":false}", "PROVIDER_VENDOR_UNSPECIFIED", "provider-7ceae560")
	f.Add("project", "provider", "bloblb", "hello.com", "blobl", "bloblb", "provider-7ceae560")

	f.Fuzz(func(t *testing.T, project, name, kind, api, config, vendor, providerID string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{
			"config": "\"\"defaultOs\":\"\",\"autoProvision\":false,\"defaultLocalAccount\":\"\",\"osSecurityFeatureEnable\":false\"",
			"vendor": vendor,
		}

		// Call your provider creation logic (replace with your actual function if needed)
		_, err := testSuite.createProvider(project, name, kind, api, args)

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List ---
		_, err = testSuite.listSite(project, make(map[string]string))

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.deleteSite(project, providerID, make(map[string]string))

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
