// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"strings"
	"testing"
)

func (s *CLITestSuite) createAMT(publisher string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create amtprofile %s --project %s`, name, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listAMT(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list amtprofile --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getAMT(publisher string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get amtprofile %s --project %s`, name, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteAMT(publisher string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete amtprofile %s --project %s`, name, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestAMT() {
	name := "test-amtprofile-1"
	path := "./testdata/sample.pfx"

	/////////////////////////////
	// Test AMT Create
	/////////////////////////////

	//Create amtprofile
	CArgs := map[string]string{
		"cert":          path,
		"cert-pass":     "pass",
		"cert-format":   "string",
		"domain-suffix": "example.com",
	}
	_, err := s.createAMT(project, name, CArgs)
	s.NoError(err)

	//Create with missing cert
	CArgs = map[string]string{
		"cert-pass":     "pass",
		"cert-format":   "string",
		"domain-suffix": "example.com",
	}
	_, err = s.createAMT(project, name, CArgs)
	s.Error(err)

	//Create with missing pass
	CArgs = map[string]string{
		"cert":          path,
		"cert-format":   "string",
		"domain-suffix": "example.com",
	}
	_, err = s.createAMT(project, name, CArgs)
	s.EqualError(err, "certificate password must be provided with --cert-pass flag and cannot be empty")

	//Create with missing format
	CArgs = map[string]string{
		"cert":          path,
		"cert-pass":     "pass",
		"domain-suffix": "example.com",
	}
	_, err = s.createAMT(project, name, CArgs)
	s.EqualError(err, "certificate format must be provided with --cert-format flag with accepted arguments `string|raw` ")

	//Create with missing domain

	CArgs = map[string]string{
		"cert":        path,
		"cert-pass":   "pass",
		"cert-format": "string",
	}
	_, err = s.createAMT(project, name, CArgs)
	s.EqualError(err, "domain suffix format must be provided with --domain-suffix flag and cannot be empty")

	/////////////////////////////
	// Test AMT List
	/////////////////////////////

	//List amtprofile
	CArgs = map[string]string{}

	listOutput, err := s.listAMT(project, CArgs)
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"AMT Profile Name": "corporate-domain",
			"Domain Suffix":    "corp.example.com",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	//List Versbose

	CArgs = map[string]string{
		"verbose": "",
	}

	listOutput, err = s.listAMT(project, CArgs)
	s.NoError(err)

	parsedOutputList = mapListOutput(listOutput)

	expectedOutputList = listCommandOutput{
		{
			"AMT Profile Name": "corporate-domain",
			"Domain Suffix":    "corp.example.com",
			"Expiration date":  "2025-12-31 23:59:59 +0000 UTC",
			"Format":           "pfx",
			"Version":          "1.0.0",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	/////////////////////////////
	// Test AMT Get
	/////////////////////////////

	//Get amtprofile
	CArgs = map[string]string{}

	getOutput, err := s.getAMT(project, name, CArgs)
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Name:":            "corporate-domain",
		"Domain Suffix:":   "corp.example.com",
		"Cert Format:":     "pfx",
		"Tenant ID:":       "tenant-abc12345",
		"Version:":         "1.0.0",
		"Expiration Date:": "2025-12-31 23:59:59 +0000 UTC",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	/////////////////////////////
	// Test AMT Delete
	/////////////////////////////

	//Delete amtprofile
	CArgs = map[string]string{}

	_, err = s.deleteAMT(project, name, CArgs)
	s.NoError(err)

}

func FuzzAMTProfile(f *testing.F) {
	// Initial corpus with basic input
	f.Add("project", "host-abcd1234", "./testdata/sample.pfx", "pass", "string", "example.com")
	f.Add("project", "host-abcd1234", "", "pass", "string", "example.com")                  // missing cert
	f.Add("project", "host-abcd1234", "./testdata/sample.pfx", "", "string", "example.com") // missing pass
	f.Add("project", "host-abcd1234", "./testdata/sample.pfx", "pass", "", "example.com")   // missing format
	f.Add("project", "host-abcd1234", "./testdata/sample.pfx", "pass", "string", "")        // missing domain

	f.Fuzz(func(t *testing.T, project, name, cert, certPass, certFormat, domainSuffix string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{
			"cert":          cert,
			"cert-pass":     certPass,
			"cert-format":   certFormat,
			"domain-suffix": domainSuffix,
		}

		expErr1 := "certificate must be provided with --cert flag"
		expErr2 := "certificate passoword must be provided with --cert-pass flag"
		expErr3 := "certificate format must be provided with --cert-format flag with accepted arguments `string|raw`"
		expErr4 := "domain suffix format must be provided with --domain-suffix flag"
		expErr5 := "failed to read certificate file"
		_, err := testSuite.createAMT(project, name, args)

		switch {
		case cert == "" || strings.TrimSpace(cert) == "":
			if !testSuite.Error(err) {
				t.Errorf("Expected error for missing cert")
			}
		case certPass == "" || strings.TrimSpace(certPass) == "":
			if !testSuite.Error(err) {
				t.Errorf("Expected error for missing cert-pass")
			}
		case certFormat == "" || strings.TrimSpace(certFormat) == "" || (certFormat != "string" && certFormat != "raw"):
			if !testSuite.Error(err) {
				t.Errorf("Expected error for missing or invalid cert-format")
			}
		case domainSuffix == "" || strings.TrimSpace(domainSuffix) == "":
			if !testSuite.Error(err) {
				t.Errorf("Expected error for missing domain-suffix")
			}
		case err != nil && (strings.Contains(err.Error(), expErr1) ||
			strings.Contains(err.Error(), expErr2) ||
			strings.Contains(err.Error(), expErr3) ||
			strings.Contains(err.Error(), expErr4) ||
			strings.Contains(err.Error(), expErr5)):
			if !testSuite.Error(err) {
				t.Errorf("Unexpected error: %v", err)
			}
		default:
			if !testSuite.NoError(err) {
				t.Errorf("Unexpected result for AMT profile creation: %v", err)
			}
		}

		args = map[string]string{}

		// --- List ---
		_, err = testSuite.listAMT(project, args)
		if project == "nonexistent-project" {
			if err == nil || !strings.Contains(err.Error(), "error getting AMT Profiles") {
				t.Errorf("Expected error for nonexistent project in list, got: %v", err)
			}
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid AMT Profile list: %v", err)
		}

		// --- Get ---
		_, err = testSuite.getAMT(project, name, args)
		if name == "" || strings.TrimSpace(name) == "" {
			if err == nil || !strings.Contains(err.Error(), "no amt profile matches the given name") &&
				!strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
				t.Errorf("Expected error for missing profile name in get, got: %v", err)
			}
		} else if project == "nonexistent-project" {
			if err == nil || !strings.Contains(err.Error(), "error getting AMT Profile") {
				t.Errorf("Expected error for nonexistent project in get, got: %v", err)
			}
		} else if err != nil && (strings.Contains(err.Error(), "no amt profile matches the given name") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 2") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 3")) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid AMT Profile get: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.deleteAMT(project, name, args)
		if name == "" || strings.TrimSpace(name) == "" {
			if err == nil || !strings.Contains(err.Error(), "no amt profile matches the given name") &&
				!strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
				t.Errorf("Expected error for missing profile name in delete, got: %v", err)
			}
		} else if project == "invalid-project" {
			if err == nil || !strings.Contains(err.Error(), "error deleting AMT profile") {
				t.Errorf("Expected error for invalid project in delete, got: %v", err)
			}
		} else if project == "nonexistent-project" {
			if err == nil || !strings.Contains(err.Error(), "Error getting AMT profiles") {
				t.Errorf("Expected error for nonexistent project in delete, got: %v", err)
			}
		} else if err != nil && (strings.Contains(err.Error(), "no amt profile matches the given name") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 2") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 3")) {
			t.Log("Expected error:", err)
		} else if err != nil && strings.Contains(err.Error(), "already exists") {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid AMT Profile delete: %v", err)
		}
	})
}
