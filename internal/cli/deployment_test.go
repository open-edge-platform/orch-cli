// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
)

func (s *CLITestSuite) TestDeploymentGetOverrideValues() {
	ov, err := getOverrideValuesRaw(
		map[string]string{
			"foo": "ns1",
		},
		map[string]string{
			"foo.property.string":     "string1",
			"foo.property.int":        "123",
			"foo.property.float":      "123.321",
			"foo.property.bool":       "true",
			"foo.another.string":      "string2",
			"foo.property.nested.int": "420",
			"bar.property.bool":       "false",
			"bar.another.int":         "42",
		})
	s.NoError(err)

	for _, v := range ov {
		if v.TargetNamespace == nil {
			fmt.Printf("%s(NONE)=", v.AppName)
		} else {
			fmt.Printf("%s(%s)=\n", v.AppName, *v.TargetNamespace)
		}
		b, err := json.Marshal(v.Values)
		s.NoError(err)
		fmt.Println(string(b))
	}
}

func (s *CLITestSuite) createDeployment(appName string, version string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`create deployment %s %s`,
		appName, version))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) listDeployment(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list deployments --project %s`,
		publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getDeployment(deployment string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get deployment %s`, deployment))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) setDeployment(deployment string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`set deployment %s`, deployment))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteDeployment(deployment string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete deployment %s`,
		deployment))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestDeployment() {
	s.T().Skip("Skip until fixed")
	err := s.createDeployment("test-app", "v1.0.0", map[string]string{
		"project":           project,
		"display-name":      "Test",
		"profile":           "test-profile",
		"application-label": "test-app.l1=l1value,test-app.l2=l2value",
	})
	s.NoError(err)

	_, err = s.listDeployment(project, make(map[string]string))
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	_, err = s.getDeployment("test-deployment", make(map[string]string))
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	_, err = s.setDeployment("test-deployment", make(map[string]string))
	s.NoError(err)

	_, err = s.deleteDeployment("test-deployment", make(map[string]string))
	s.NoError(err)
}
