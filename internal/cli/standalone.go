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

const defaultEmtsRepoCommitID = "a28db5e6d2d9fb6ec5368246c13bfff7fc1a1ae2"

var (
	CollectLogsScriptSource   = "https://raw.githubusercontent.com/open-edge-platform/edge-microvisor-toolkit-standalone-node/%s/standalone-node/provisioning_scripts/collect-logs.sh"
	K3sConfigureScriptSource  = "https://raw.githubusercontent.com/open-edge-platform/edge-microvisor-toolkit-standalone-node/%s/standalone-node/provisioning_scripts/k3s-configure.sh"
	K3sInstallerScriptSource  = "https://raw.githubusercontent.com/open-edge-platform/edge-microvisor-toolkit-standalone-node/%s/standalone-node/cluster_installers/sen-k3s-installer.sh"
	K3sPostRebootScriptSource = "https://raw.githubusercontent.com/open-edge-platform/edge-microvisor-toolkit-standalone-node/%s/standalone-node/provisioning_scripts/k3s-setup-post-reboot.sh"
)

var cloudInitTemplate = `
#cloud-config

merge_how: 'dict(recurse_array,no_replace)+list(append)'

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
  - path: /etc/cloud/k3s-setup-post-reboot.sh
    content: |
      {{- .K3sPostRebootScript | indent 6 }}
{{- end }}
  {{- range .CloudInitWriteFiles }}
  - path: {{ .Path }}
    permissions: "{{ .Permissions }}"
    content: |
      {{- .Content | nindent 6 }}
  {{- end }}

runcmd:
{{- if .WithUserApps }}
  - |
    mkdir -p /opt/user-apps
    curl --noproxy '*' -k {{ .NginxFQDN }}/tink-stack/user-apps.tar.gz -o /tmp/user-apps.tar.gz
    tar -xzvf /tmp/user-apps.tar.gz -C /opt/user-apps
{{- end }}
  - |
    grep -qF "http_proxy" /etc/environment || echo http_proxy={{ .http_proxy }} >> /etc/environment
    grep -qF "https_proxy" /etc/environment || echo https_proxy={{ .https_proxy }} >> /etc/environment
    grep -qF "no_proxy" /etc/environment || echo no_proxy="{{ .no_proxy }}" >> /etc/environment
    grep -qF "HTTP_PROXY" /etc/environment || echo HTTP_PROXY={{ .HTTP_PROXY }} >> /etc/environment
    grep -qF "HTTPS_PROXY" /etc/environment || echo HTTPS_PROXY={{ .HTTPS_PROXY }} >> /etc/environment
    grep -qF "NO_PROXY" /etc/environment || echo NO_PROXY={{ .NO_PROXY }} >> /etc/environment
    sed -i 's|^PATH="\(.*\)"$|PATH="\1:/var/lib/rancher/k3s/bin"|' /etc/environment
    source /etc/environment
    echo "source /etc/environment" >> /home/{{ .user_name }}/.bashrc
    echo "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml" >> /home/{{ .user_name }}/.bashrc
    echo "alias k='KUBECONFIG=/etc/rancher/k3s/k3s.yaml /usr/local/bin/k3s kubectl'" >> /home/{{ .user_name }}/.bashrc
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

const commandHelp = `# Generate cloud-init config for EMT-Standalone nodes (config-file must be populated before)
orch-cli generate standalone-config -c config-file 

# Generate cloud-init config for EMT-Standalone nodes, specify output file
orch-cli generate standalone-config -c config-file -o /tmp/cloud-init.cfg

# Generate cloud-init config for EMT-Standalone nodes with user apps (--api-endpoint is mandatory if user apps are enabled)
orch-cli generate standalone-config -c config-file --user-apps=true --api-endpoint https://api.cluster.onprem

# Generate cloud-init config for EMT-Standalone nodes in sync with a specific EMT-S repository commit ID
orch-cli generate standalone-config -c config-file --emts-repo-version <commit-ID>
`

func getStandaloneConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "standalone-config",
		Short:   "Generate custom config for standalone nodes",
		Args:    cobra.ExactArgs(0),
		RunE:    runGenerateStandaloneConfigCommand,
		Example: commandHelp,
	}
	cmd.Flags().StringP("config-file", "c", "", "config-file with user inputs")
	cmd.Flags().StringP("output-file", "o", "cloud-init.cfg", "Override output filename")
	cmd.Flags().BoolP("user-apps", "u", false, "Pre-load user apps")
	cmd.Flags().StringP("emts-repo-version", "", defaultEmtsRepoCommitID, "Commit ID of EMT-S repository to sync with")
	return cmd
}

func getOutFile(cmd *cobra.Command) (string, error) {
	outPath, err := cmd.Flags().GetString("output-file")
	if err != nil {
		return "", err
	}
	return outPath, nil
}

func getEMTSRepoID(cmd *cobra.Command) (string, error) {
	repoID, err := cmd.Flags().GetString("emts-repo-version")
	if err != nil {
		return "", err
	}
	return repoID, nil
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

func getUserAppsFlag(cmd *cobra.Command) (bool, error) {
	withUserApps, err := cmd.Flags().GetBool("user-apps")
	if err != nil {
		return false, err
	}
	return withUserApps, nil
}

func getNginxFQDNFromAPIEndpoint(cmd *cobra.Command) (string, error) {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return "", err
	}

	nginxFQDN := strings.Replace(serverAddress, "api.", "tinkerbell-nginx.", 1)

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
	//nolint:gosec //
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
		if strings.Contains(line, "Custom cloud-init Config file") {
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

func loadConfig(path string, withUserApps bool, nginxFQDN, emtsRepoID string) (map[string]interface{}, error) {
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

	CollectLogsScriptSourceURL := fmt.Sprintf(CollectLogsScriptSource, emtsRepoID)
	K3sConfigureScriptSourceURL := fmt.Sprintf(K3sConfigureScriptSource, emtsRepoID)
	K3sInstallerScriptSourceURL := fmt.Sprintf(K3sInstallerScriptSource, emtsRepoID)
	K3sPostRebootScriptSourceURL := fmt.Sprintf(K3sPostRebootScriptSource, emtsRepoID)

	collectLogsScript, err := downloadFileFromURL(CollectLogsScriptSourceURL)
	if err != nil {
		return nil, err
	}

	k3sConfigureScript, err := downloadFileFromURL(K3sConfigureScriptSourceURL)
	if err != nil {
		return nil, err
	}

	k3sInstallerScript, err := downloadFileFromURL(K3sInstallerScriptSourceURL)
	if err != nil {
		return nil, err
	}

	k3sPostRebootScript, err := downloadFileFromURL(K3sPostRebootScriptSourceURL)
	if err != nil {
		return nil, err
	}

	config["CollectLogsScript"] = collectLogsScript
	config["K3sConfigureScript"] = k3sConfigureScript
	config["K3sInstallerScript"] = k3sInstallerScript
	config["K3sPostRebootScript"] = k3sPostRebootScript
	config["WithUserApps"] = withUserApps

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

	emtsRepoID, err := getEMTSRepoID(cmd)
	if err != nil {
		return err
	}

	withUserApps, err := getUserAppsFlag(cmd)
	if err != nil {
		return err
	}

	nginxFQDN := ""
	if withUserApps {
		nginxFQDN, err = getNginxFQDNFromAPIEndpoint(cmd)
		if err != nil {
			return err
		}
		if nginxFQDN == "" {
			return fmt.Errorf("setting API endpoint is mandatory when uploading user apps")
		}
	}

	config, err := loadConfig(configFilePath, withUserApps, nginxFQDN, emtsRepoID)
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
