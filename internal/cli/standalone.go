// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bufio"
	"bytes"
	"fmt"
	sprig "github.com/go-task/slim-sprig"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
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

merge_how:
  - name: list
    settings: [ append ]
  - name: dict
    settings: [ no_replace, recurse_list ]

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
  {{- range .CloudInitWriteFiles }}
  - path: {{ .Path }}
    permissions: "{{ .Permissions }}"
    content: |
      {{- .Content | nindent 6 }}
  {{- end }}

runcmd:
{{- if gt (len .UserApps) 0 }}
  - |
    mkdir -p /opt/user-apps
{{- $nginx_fqdn := .NginxFQDN }}
{{- range .UserApps }}
    curl --noproxy '*' -k {{ $nginx_fqdn }}/tink-stack/user-apps/{{ . }} -o /opt/user-apps/{{ . }}
{{- end }}
{{- end }}
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
{{- if .huge_page_config }}
    echo .huge_page_config | tee /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages
{{- end }}
    chmod +x /etc/cloud/k3s-configure.sh
    bash /etc/cloud/k3s-configure.sh
{{- end }}
{{- range .CloudInitServicesEnable }}
  - systemctl enable {{ . }}
{{- end }}
{{- range .CloudInitServicesDisable }}
  - systemctl disable {{ . }}
{{- end }}
{{- range .CloudInitRuncmd }}
  - |
    {{- . | nindent 4 }}
{{- end }}
`

type CloudInitSection struct {
	Services struct {
		Enable  []string `yaml:"enable"`
		Disable []string `yaml:"disable"`
	} `yaml:"services"`
	WriteFiles []WriteFile `yaml:"write_files"`
	RunCmd     []string    `yaml:"runcmd"`
}

type WriteFile struct {
	Path        string `yaml:"path"`
	Permissions string `yaml:"permissions"`
	Content     string `yaml:"content"`
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
	cmd.Flags().StringP("user-apps", "u", "", "Directory with user apps to pre-load")
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

func getUserAppsFlag(cmd *cobra.Command) (string, error) {
	userAppsDirectory, err := cmd.Flags().GetString("user-apps")
	if err != nil {
		return "", err
	}
	return userAppsDirectory, nil
}

func getNginxFQDNFromAPIEndpoint(cmd *cobra.Command) (string, error) {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return "", err
	}

	nginxFQDN := strings.Replace(serverAddress, "api.", "tinkerbell-nginx.", 1)
	fmt.Println("getting nginx", nginxFQDN)
	return nginxFQDN, nil
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

func extractYamlBlock(path string) (CloudInitSection, error) {
	file, err := os.Open(path)
	if err != nil {
		return CloudInitSection{}, err
	}
	defer file.Close()

	var (
		yamlLines   []string
		startedYaml bool
		scanner     = bufio.NewScanner(file)
	)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "services:") {
			startedYaml = true
		}
		if startedYaml {
			yamlLines = append(yamlLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return CloudInitSection{}, err
	}

	yamlContent := strings.Join(yamlLines, "\n")
	var parsed CloudInitSection
	err = yaml.Unmarshal([]byte(yamlContent), &parsed)
	return parsed, err
}

func getPasswordFromUserInput(username string) (string, error) {
	fmt.Printf("Please Set the Password for %q\n", username)
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Println()

	return string(bytePassword), nil
}

func loadConfig(path, userAppsDir, nginxFQDN string) (map[string]interface{}, error) {
	config := make(map[string]interface{})
	config["NginxFQDN"] = nginxFQDN

	cloudInit, err := extractYamlBlock(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML block: %s", err)
	}

	tmpFile, err := os.CreateTemp("", "tmp_*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	inputFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}
	defer inputFile.Close()

	// remove YAML blocks from env file so it's parseable by godotenv
	writer := bufio.NewWriter(tmpFile)

	scanner := bufio.NewScanner(inputFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Custom cloud-init Config file") {
			break // Stop copying when we reach the YAML marker
		}
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return nil, fmt.Errorf("failed writing to temp file: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}
	writer.Flush()
	tmpFile.Close()

	// Load the file into environment variables
	envVars, err := godotenv.Read(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %w", err)
	}
	for k, v := range envVars {
		config[k] = v
	}

	config["CloudInitWriteFiles"] = cloudInit.WriteFiles
	config["CloudInitServicesEnable"] = cloudInit.Services.Enable
	config["CloudInitServicesDisable"] = cloudInit.Services.Disable
	config["CloudInitRuncmd"] = cloudInit.RunCmd

	if config["passwd"], err = getPasswordFromUserInput(config["user_name"].(string)); err != nil {
		return nil, err
	}

	hashed, err := hashPassword(config["passwd"].(string))
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

	userApps := make([]string, 0)
	if userAppsDir != "" {
		entries, err := os.ReadDir(userAppsDir)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				userApps = append(userApps, entry.Name())
			}
		}

		config["UserApps"] = userApps
	}

	return config, nil
}

func generateCloudInit(config map[string]interface{}) (string, error) {

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

	userApps, err := getUserAppsFlag(cmd)
	if err != nil {
		return err
	}

	nginxFQDN := ""
	if userApps != "" {
		nginxFQDN, err = getNginxFQDNFromAPIEndpoint(cmd)
		if err != nil {
			return err
		}
		if nginxFQDN == "" {
			return fmt.Errorf("setting API endpoint is mandatory when uploading user apps")
		}
	}

	config, err := loadConfig(configFilePath, userApps, nginxFQDN)
	if err != nil {
		return err
	}

	fmt.Println(config)

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
