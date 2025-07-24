// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import "fmt"

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
	s.EqualError(err, "certificate passoword must be provided with --cert-pass flag ")

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
	s.EqualError(err, "domain suffix format must be provided with --domain-suffix flag ")

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
