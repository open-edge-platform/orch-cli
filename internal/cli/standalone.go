// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"fmt"
	sprig "github.com/go-task/slim-sprig"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

const (
	CollectLogsScriptSource  = "https://raw.githubusercontent.com/open-edge-platform/edge-microvisor-toolkit-standalone-node/refs/tags/standalone-node/3.0.11/standalone-node/provisioning_scripts/collect-logs.sh"
	K3sConfigureScriptSource = "https://raw.githubusercontent.com/open-edge-platform/edge-microvisor-toolkit-standalone-node/refs/tags/standalone-node/3.0.11/standalone-node/provisioning_scripts/k3s-configure.sh"
	K3sInstallerScriptSource = "https://raw.githubusercontent.com/open-edge-platform/edge-microvisor-toolkit-standalone-node/refs/tags/standalone-node/3.0.11/standalone-node/cluster_installers/sen-k3s-installer.sh"
)

var cloudInitTemplate = `
#cloud-config

# NTP Time Sync Configuration
ntp:
  enabled: true
  ntp_client: systemd-timesyncd
  servers:
    - time.google.com

users:
  - name: {{ .user_name }}
    primary_group: users
    groups: [sudo]
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: false
    passwd: "{{ .passwd }}"
{{- if .ssh_key }}
	ssh_authorized_keys:
      - {{ .ssh_key }}
{{- end }}

write_files:
  - path: /etc/cloud/collect-logs.sh
    content: |
      {{- .CollectLogsScript | indent 6 }}
{{- if eq .host_type "kubernetes" }}
  - path: /etc/cloud/k3s-configure.sh
    content: |
      {{- .K3sConfigureScript | indent 6 }}
  - path: /tmp/k3s-artifacts/sen-k3s-installer.sh
    content: |
      {{- .K3sInstallerScript | indent 6 }}
{{- end }}

runcmd:
  - |
    grep -qF "http_proxy" /etc/environment || echo http_proxy={{ .http_proxy }} >> /etc/environment
    grep -qF "https_proxy" /etc/environment || echo https_proxy={{ .https_proxy }} >> /etc/environment
    grep -qF "no_proxy" /etc/environment || echo no_proxy={{ .no_proxy }} >> /etc/environment
    grep -qF "HTTP_PROXY" /etc/environment || echo HTTP_PROXY={{ .HTTP_PROXY }} >> /etc/environment
    grep -qF "HTTPS_PROXY" /etc/environment || echo HTTPS_PROXY={{ .HTTPS_PROXY }} >> /etc/environment
    grep -qF "NO_PROXY" /etc/environment || echo NO_PROXY={{ .NO_PROXY }} >> /etc/environment
	
    sed -i 's|^PATH="\(.*\)"$|PATH="\1:/var/lib/rancher/k3s/bin"|' /etc/environment
	
    source /etc/environment

    echo "source /etc/environment" >> /home/{{ .user_name }}/.bashrc
    echo "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml" >> /home/{{ .user_name }}/.bashrc
    echo "alias k='KUBECONFIG=/etc/rancher/k3s/k3s.yaml /usr/bin/k3s kubectl'" >> /home/{{ .user_name }}/.bashrc
{{- if eq .host_type "kubernetes" }}
    chmod +x /etc/cloud/k3s-configure.sh
    bash /etc/cloud/k3s-configure.sh
{{- end }}
`

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

func hashPassword(password string) (string, error) {
	cmd := exec.Command("openssl", "passwd", "-6", password)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func downloadFileFromURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body: %s", err)
	}
	return string(data), nil
}

func loadConfig(path string) (map[string]string, error) {
	// Load the file into environment variables
	config, err := godotenv.Read(path)
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %w", err)
	}

	hashed, err := hashPassword(config["passwd"])
	if err != nil {
		return nil, err
	}

	config["passwd"] = hashed

	collectLogsScript, err := downloadFileFromURL(CollectLogsScriptSource)
	if err != nil {
		return nil, err
	}

	k3sConfigureScript, err := downloadFileFromURL(K3sConfigureScriptSource)
	if err != nil {
		return nil, err
	}

	k3sInstallerScript, err := downloadFileFromURL(K3sInstallerScriptSource)
	if err != nil {
		return nil, err
	}

	config["CollectLogsScript"] = collectLogsScript
	config["K3sConfigureScript"] = k3sConfigureScript
	config["K3sInstallerScript"] = k3sInstallerScript

	return config, nil
}

func generateCloudInit(config map[string]string) (string, error) {

	tmpl, err := template.New("cloud-init").Option("missingkey=error").Funcs(sprig.TxtFuncMap()).Parse(cloudInitTemplate)
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

func runGenerateStandaloneConfigCommand(cmd *cobra.Command, _ []string) error {
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

	err = os.WriteFile(out, []byte(cloudInit), 0600)
	if err != nil {
		return fmt.Errorf("failed to write cloud-init to path %q", out)
	}

	return nil
}
