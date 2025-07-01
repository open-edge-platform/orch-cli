// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"os"
	"text/template"
)

var cloudInitTemplate = `
#cloud-config
users:
  - name: {{ .user_name }}
    primary_group: users
    groups: [sudo]
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: false
    passwd: "$6$wEdsdfg24$sDcHV9i9ImzWvUzpslfydFX9ROe3tAjP32W8BiubA1aAqMylYeH5DULATK49EVzKDr6gUvE9kM2FzKQaXUxzW0"
{{- if .ssh_key }}
	ssh_authorized_keys:
      - {{ .ssh_key }}
{{- end }}
runcmd:
  - |
    grep -qF "http_proxy" /etc/environment || echo http_proxy={{ .http_proxy }} >> /etc/environment
    grep -qF "https_proxy" /etc/environment || echo https_proxy={{ .https_proxy }} >> /etc/environment
    grep -qF "no_proxy" /etc/environment || echo no_proxy={{ .no_proxy }} >> /etc/environment
    grep -qF "HTTP_PROXY" /etc/environment || echo HTTP_PROXY={{ .HTTP_PROXY }} >> /etc/environment
    grep -qF "HTTPS_PROXY" /etc/environment || echo HTTPS_PROXY={{ .HTTPS_PROXY }} >> /etc/environment
    grep -qF "NO_PROXY" /etc/environment || echo NO_PROXY={{ .NO_PROXY }} >> /etc/environment
`

type StandaloneConfig struct {
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
	SSHKey     string
	UserName   string
	Password   string
}

func getStandaloneConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "standalone-config",
		Short:   "Generate custom config for standalone nodes",
		Args:    cobra.ExactArgs(0),
		RunE:    runGenerateStandaloneConfigCommand,
		Example: "orch-cli generate standalone-config",
	}
	cmd.Flags().StringP("config-file", "c", "", "config-file with user inputs")
	cmd.Flags().StringP("output-file", "o", "cloud-init.cfg", "Override output filename")
	return cmd
}

func getOutFile(cmd *cobra.Command) (string, error) {
	outPath, err := cmd.Flags().GetString("output-file")
	if err != nil {
		return "", err
	}
	return outPath, nil
}

func getConfigFileInput(cmd *cobra.Command) (string, error) {
	configFilePath, err := cmd.Flags().GetString("config-file")
	if err != nil {
		return "", err
	}
	if configFilePath == "" {
		return "", fmt.Errorf("required flag \"config-file\" not set")
	}
	return configFilePath, nil
}

func loadConfig(path string) (map[string]string, error) {
	// Load the file into environment variables
	config, err := godotenv.Read(path)
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %w", err)
	}

	fmt.Println(config)

	return config, nil
}

func generateCloudInit(config map[string]string) (string, error) {
	tmpl, err := template.New("cloudinit").Parse(cloudInitTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %v", err)
	}

	var rendered bytes.Buffer
	err = tmpl.Execute(&rendered, config)
	if err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}
	return rendered.String(), nil
}

func runGenerateStandaloneConfigCommand(cmd *cobra.Command, args []string) error {
	configFilePath, err := getConfigFileInput(cmd)
	if err != nil {
		return err
	}

	out, err := getOutFile(cmd)
	if err != nil {
		return err
	}

	config, err := loadConfig(configFilePath)
	if err != nil {
		return err
	}

	cloudInit, err := generateCloudInit(config)
	if err != nil {
		return err
	}

	err = os.WriteFile(out, []byte(cloudInit), 0644)
	if err != nil {
		return fmt.Errorf("failed to write cloud-init to path %q", out)
	}

	return nil
}
