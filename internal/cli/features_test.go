// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

func (s *CLITestSuite) listFeatures(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list features  --project %s`, project))
	return s.runCommand(commandString)
}
func (s *CLITestSuite) TestFeatures() {
	getOutput, err := s.listFeatures(project, make(map[string]string))
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)

	expectedOutput := map[string]string{

		"Edge Orchestrator Features:":              "",
		"application-orchestration":                "enabled",
		"cluster-orchestration":                    "enabled",
		"edge-infrastructure-manager":              "enabled",
		"edge-infrastructure-manager.day2":         "enabled",
		"edge-infrastructure-manager.onboarding":   "enabled",
		"edge-infrastructure-manager.oob":          "enabled",
		"edge-infrastructure-manager.provisioning": "enabled",
		"edge-infrastructure-manager.oxm-profile":  "enabled",
		"multitenancy":                             "enabled",
		"orchestrator-observability":               "enabled",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	// Print config file contents
	homeDir, err := os.UserHomeDir()
	s.NoError(err)
	configPath := filepath.Join(homeDir, ".orch-cli", "orch-cli.yaml")
	configContent, err := os.ReadFile(configPath)
	s.NoError(err)
	fmt.Printf("\n=== Config file contents (%s) ===\n%s\n=== End of config file ===\n\n", configPath, string(configContent))

	//Disable a feature using config command
	_, err = s.runCommand(fmt.Sprintf(`config set orchestrator.features.edge-infrastructure-manager.onboarding.installed "false" --project %s`, project))
	s.NoError(err)

	// Print config file contents after disabling feature
	configContent, err = os.ReadFile(configPath)
	s.NoError(err)
	fmt.Printf("\n=== Config file after disabling onboarding ===\n%s\n=== End of config file ===\n\n", string(configContent))

	getOutput, err = s.listFeatures(project, make(map[string]string))
	s.NoError(err)

	parsedOutput = mapGetOutput(getOutput)

	expectedOutput = map[string]string{

		"Edge Orchestrator Features:":              "",
		"application-orchestration":                "enabled",
		"cluster-orchestration":                    "enabled",
		"edge-infrastructure-manager":              "enabled",
		"edge-infrastructure-manager.day2":         "enabled",
		"edge-infrastructure-manager.onboarding":   "disabled",
		"edge-infrastructure-manager.oob":          "enabled",
		"edge-infrastructure-manager.provisioning": "enabled",
		"edge-infrastructure-manager.oxm-profile":  "enabled",
		"multitenancy":                             "enabled",
		"orchestrator-observability":               "enabled",
	}
	s.compareGetOutput(expectedOutput, parsedOutput)

	_, err = s.runCommand(fmt.Sprintf(`config set features.edge-infrastructure-manager.onboarding.installed true --project %s`, project))
	s.NoError(err)
}
