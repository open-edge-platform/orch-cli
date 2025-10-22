// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	e "github.com/open-edge-platform/cli/internal/errors"
	"github.com/open-edge-platform/cli/internal/files"
	"github.com/open-edge-platform/cli/internal/types"
	"github.com/open-edge-platform/cli/internal/validator"
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/cluster"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const listHostExamples = `# List all hosts
orch-cli list host --project some-project

# List hosts using a predefined filter (options: provisioned, onboarded, registered, "not connected", deauthorized) 
orch-cli list host --project some-project --filter provisioned

# List hosts using a custom filter (see: https://google.aip.dev/160 and API spec @ https://github.com/open-edge-platform/orch-utils/blob/main/tenancy-api-mapping/openapispecs/generated/amc-infra-core-edge-infrastructure-manager-openapi-all.yaml )
orch-cli list host --project some-project --filter "serialNumber='123456789'"

# List hosts in a specific site using site ID (--site flag will take precedence over --region flag)
orch-cli list host --project some-project --site site-c69a3c81

# List hosts in a specific region using region ID (--site flag will take precedence over --region flag)
orch-cli list host --project some-project --region region-1234abcd

# List hosts with a specific workload using workload name
orch-cli list host --project some-project --workload cluster-sn000320

# List hosts without a workload using NotAssigned argument
orch-cli list host --project some-project --workload NotAssigned
`

const getHostExamples = `# Get detailed information about specific host using the host Resource ID
orch-cli get host host-1234abcd --project some-project`

const createHostExamples = `# Provision a host or a number of hosts from a CSV file

# Generate CSV input file using the --generate-csv flag - the default output will be a base test.csv file.
orch-cli create host --project some-project --generate-csv

# Generate CSV input file using the --generate-csv flag - the defined output will be a base myhosts.csv file.
orch-cli create host --project some-project --generate-csv=myhosts.csv

# Sample input csv file hosts.csv

Serial - Serial Number of the machine - mandatory field (both or one of Serial or UUID must be provided)
UUID - UUID of the machine - mandatory field (both or one of Serial or UUID must be provided), UUID must be provided if K8s cluster is going to be auto provisioned
OSProfile - OS Profile to be used for provisioning of the host - name of the profile or it's resource ID - mandatory field
Site - The resource ID of the site to which the host will be provisioned - mandatory field
Secure - Optional security feature to configure for the host - must be supported by OS Profile if enabled
Remote User - Optional remote user name or resource ID to configure for the host
Metadata - Optional metadata to configure for the host
LVMSize - Optional LVM size to be configured for the host
CloudInitMeta - Optional Cloud Init Metadata to be configured for the host
K8sEnable - Optional command to enable cluster deployment
K8sClusterTemplate - Optional Cluster template to be used for K8s deployment on the host, must be provided if K8sEnable is true
K8sClusterConfig - Optional Cluster config to be used to specify role and cluster name and/or cluster labels

Serial,UUID,OSProfile,Site,Secure,RemoteUser,Metadata,LVMSize,CloudInitMeta,K8sEnable,K8sClusterTemplate,K8sConfig,Error - do not fill
2500JF3,4c4c4544-2046-5310-8052-cac04f515233,"Edge Microvisor Toolkit 3.0.20250617",site-c69a3c81,,localaccount-4c2c5f5a
1500JF3,1c4c4544-2046-5310-8052-cac04f515233,"Edge Microvisor Toolkit 3.0.20250617",site-c69a3c81,false,,key1=value1&key2=value2
15002F3,114c4544-2046-5310-8052-cac04f512233,"Edge Microvisor Toolkit 3.0.20250617",site-c69a3c81,false,,key1=value2&key3=value4
11002F3,2c4c4544-2046-5310-8052-cac04f512233,"Edge Microvisor Toolkit 3.0.20250617",site-c69a3c81,false,,key1=value2&key3=value4,,cloudinitname&customconfig-1234abcd
25002F3,214c4544-2046-5310-8052-cac04f512233,"Edge Microvisor Toolkit 3.0.20250617",site-c69a3c81,false,user,key1=value2&key3=value4,60,,true,baseline:v2.0.2,,role:all;name:mycluster;labels:key1=val1&key2=val2

# --dry-run allows for verification of the validity of the input csv file without creating hosts
orch-cli create host --project some-project --import-from-csv test.csv --dry-run

# Create hosts - --import-from-csv is a mandatory flag pointing to the input file. Successfully provisioned host indicated by output - errors provided in output file
orch-cli create host --project some-project --import-from-csv test.csv

# Optional flag ovverides - the flag will override all instances of an attribute inside the CSV file

--remote-user - name or id of a SSH user
--site - site ID
--secure - true or false - security feature configuration
--os-profile - name or ID of the OS profile
--metadata - key value paired metatada separated by &, must be put in quotes.
--cluster-deploy - true or false - cluster deployment configuration
--cluster-template - name and version of the cluster template to be used for cluster cration (separated by :)
--cluster-config - extra configuration for cluster creation empty defaults to "role:all", if not empty role must be defined, name and labels are optional (labels separated by &)
--cloud-init - name or resource ID of custom config - multiple configs must be separated by &
--lvm-size - size of the LVM to be configured for the host

# Create hosts from CSV and override provided values
/orch-cli create host --project some-project --import-from-csv test.csv --os-profile ubuntu-22.04-lts-generic-ext --secure false --site site-7ca0a77c --remote-user user --metadata "key7=val7key3=val3"
`

const deleteHostExamples = `#Delete a host using it's host Resource ID
orch-cli delete host host-1234abcd  --project itep`

const deauthorizeHostExamples = `#Deauthorize the host and it's access to Edge Orchestrator using the host Resource ID
orch-cli deauthorize host host-1234abcd  --project itep`

const setHostExamples = `#Set an attribute of a host or execute an action - at least one flag must be specified

#Set host power state to on
orch-cli set host host-1234abcd  --project itep --power on

#Set host power command policy
orch-cli set host host-1234abcd  --project itep --power-policy ordered

--power - Set desired power state of host to on|off|cycle|hibernate|reset|sleep
--power-policy - Set the desired power command policy to ordered|immediate

#Set host AMT state to provisioned
orch-cli set host host-1234abcd --project some-project --amt-state provisioned

--amt-state - Set desired AMT state of host to provisioned|unprovisioned

# Generate CSV input file using the --generate-csv flag - the default output will be a base test.csv file.
orch-cli set host --project some-project --generate-csv

# Generate CSV input file using the --generate-csv flag - the defined output will be a base myhosts.csv file.
orch-cli set host --project some-project --generate-csv=myhosts.csv

# Sample input csv file hosts.csv

Name - Name of the machine - mandatory field
ResourceID - Unique Identifier of host - mandatory field
DesiredAmtState - Desired AMT state of host - provisioned|unprovisioned - mandatory field

Name,ResourceID,DesiredAmtState
host-1,host-1234abcd,provisioned

# --dry-run allows for verification of the validity of the input csv file without updating hosts
orch-cli set host --project some-project --import-from-csv test.csv --dry-run

# Set hosts - --import-from-csv is a mandatory flag pointing to the input file. Successfully provisioned host indicated by output - errors provided in output file
orch-cli set host --project some-project --import-from-csv test.csv

#Set host OS Update policy
orch-cli set host host-1234abcd  --project itep --osupdatepolicy <resourceID>

--osupdatepolicy - Set the OS Update policy for the host, must be a valid resource ID of an OS Update policy
`

var hostHeaderGet = "\nDetailed Host Information\n"
var filename = "test.csv"

const kVSize = 2

type ResponseCache struct {
	OSProfileCache          map[string]infra.OperatingSystemResource
	SiteCache               map[string]infra.SiteResource
	LACache                 map[string]infra.LocalAccountResource
	HostCache               map[string]infra.HostResource
	K8sClusterTemplateCache map[string]cluster.TemplateInfo
	K8sClusterNodesCache    map[string][]cluster.NodeSpec
	CICache                 map[string]infra.CustomConfigResource
}

type CVEEntry struct {
	CVEID            string   `json:"cve_id"`
	Priority         string   `json:"priority"`
	AffectedPackages []string `json:"affected_packages"`
}

func filterHelper(f string) *string {
	if f != "" {
		switch f {
		case "onboarded":
			f = "hostStatus='onboarded'"
		case "registered":
			f = "hostStatus='registered'"
		case "provisioned":
			f = "hostStatus='provisioned'"
		case "deauthorized":
			f = "hostStatus='invalidated'"
		case "not connected":
			f = "hostStatus=''"
		case "error":
			f = "hostStatus='error'"
		default:
		}
		return &f
	}
	return nil

}

func filterSitesHelper(s string) (*string, error) {
	if s != "" {
		re := regexp.MustCompile(`^site-[a-zA-Z0-9]{8}$`)
		if !re.MatchString(s) {
			return nil, fmt.Errorf("invalid site id %s --site expects site-abcd1234 format", s)
		}
		return &s, nil
	}
	return nil, nil
}

func filterRegionsHelper(r string) (*string, error) {
	if r != "" {
		re := regexp.MustCompile(`^region-[a-zA-Z0-9]{8}$`)
		if !re.MatchString(r) {
			return nil, fmt.Errorf("invalid region id %s --region expects region-abcd1234 format", r)
		}
		return &r, nil
	}
	return nil, nil
}

// Prints Host list in tabular format
func printHosts(writer io.Writer, hosts *[]infra.HostResource, verbose bool) {
	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "Resource ID", "Name", "Host Status", "Provisioning Status",
			"Serial Number", "Operating System", "Site ID", "Site Name", "Workload", "Host ID", "UUID", "Processor", "Available Update", "Trusted Compute")
	} else {
		var shortHeader = fmt.Sprintf("\n%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s", "Resource ID", "Name", "Host Status", "Provisioning Status", "Serial Number", "Operating System", "Site ID", "Site Name", "Workload")
		fmt.Fprintf(writer, "%s\n", shortHeader)
	}
	for _, h := range *hosts {
		//TODO clean this up
		os, workload, site, siteName, provStat := "Not provisioned", "Not assigned", "Not provisioned", "Not provisioned", "Not provisioned"
		host := "Not connected"

		if h.Instance != nil {
			if h.Instance.CurrentOs != nil && h.Instance.CurrentOs.Name != nil {
				os = toJSON(h.Instance.CurrentOs.Name)
			}
			if h.Instance.WorkloadMembers != nil && len(*h.Instance.WorkloadMembers) > 0 {
				workload = toJSON((*h.Instance.WorkloadMembers)[0].Workload.Name)
			}
		}
		if h.SiteId != nil {
			site = toJSON(h.SiteId)
		}

		if h.Site != nil && h.Site.Name != nil {
			siteName = toJSON(h.Site.Name)
		}

		if h.HostStatus != nil && *h.HostStatus != "" {
			// Only display 'Waiting on node agents' when HostStatus is 'error' (case-insensitive), Instance is not nil, and InstanceStatusDetail contains 'of 10 components running'
			if strings.EqualFold(*h.HostStatus, "error") && h.Instance != nil && h.Instance.InstanceStatusDetail != nil && strings.Contains(*h.Instance.InstanceStatusDetail, "of 10 components running") {
				host = "Waiting on node agents"
			} else {
				host = *h.HostStatus
			}
		}

		if h.Instance != nil && h.Instance.ProvisioningStatus != nil {
			provStat = *h.Instance.ProvisioningStatus
		}

		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%v\t%v\t%v\t%v\t%v\n", *h.ResourceId, h.Name, host, provStat, *h.SerialNumber, os, site, siteName, workload)
		} else {
			avupdt := "No update"
			tcomp := "Not compatible"

			//TODO
			//if h.CurrentOs != h.desiredOS avupdt is available
			//if tcomp is set then reflect

			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", *h.ResourceId, h.Name, host, provStat, *h.SerialNumber,
				os, site, siteName, workload, h.Name, *h.Uuid, *h.CpuModel, avupdt, tcomp)
		}
	}
}

func printHost(writer io.Writer, host *infra.HostResource) {

	updatestatus := ""
	hoststatus := "Not connected"
	currentOS := ""
	osprofile := ""
	customcfg := ""
	ip := ""
	var cveEntries []CVEEntry
	provstatus := "Not Provisioned"
	hostdetails := ""
	lvmsize := ""

	//TODO Build out the host information
	if host != nil && host.Instance != nil && host.Instance.UpdateStatus != nil {
		updatestatus = toJSON(host.Instance.UpdateStatus)
	}

	if host != nil && host.Instance != nil && host.Instance.CurrentOs != nil && host.Instance.CurrentOs.Name != nil {
		currentOS = toJSON(host.Instance.CurrentOs.Name)
	}

	if host != nil && host.Instance != nil && host.Instance.Os != nil && host.Instance.Os.Name != nil {
		osprofile = toJSON(host.Instance.Os.Name)
	}

	if *host.HostStatus != "" {
		// Only display 'Waiting on node agents' when HostStatus is 'error' (case-insensitive), Instance is not nil, and InstanceStatusDetail contains 'of 10 components running'
		if strings.EqualFold(*host.HostStatus, "error") && host.Instance != nil && host.Instance.InstanceStatusDetail != nil && strings.Contains(*host.Instance.InstanceStatusDetail, "of 10 components running") {
			hoststatus = "Waiting on node agents"
		} else {
			hoststatus = *host.HostStatus
		}
	}

	if host.Instance != nil && host.Instance.ProvisioningStatus != nil && *host.Instance.ProvisioningStatus != "" {
		provstatus = *host.Instance.ProvisioningStatus
	}

	if host.Instance != nil && host.Instance.InstanceStatusDetail != nil && *host.Instance.InstanceStatusDetail != "" {
		hostdetails = *host.Instance.InstanceStatusDetail
	}

	if host.Instance != nil && host.Instance.CustomConfig != nil {
		if len(*host.Instance.CustomConfig) > 0 {
			configs := ""
			for _, ccfg := range *host.Instance.CustomConfig {
				configs = configs + ccfg.Name + " "
			}
			customcfg = configs
		}
	}

	if host.HostNics != nil && len(*host.HostNics) > 0 {
		for _, nic := range *host.HostNics {
			if nic.Ipaddresses != nil && len(*nic.Ipaddresses) > 0 && nic.DeviceName != nil && (*nic.Ipaddresses)[0].Address != nil {
				deviceName := *nic.DeviceName
				address := *(*nic.Ipaddresses)[0].Address
				ip = ip + deviceName + " " + address + "; "
			}
		}
	}

	if host.UserLvmSize != nil {
		lvmsize = strconv.FormatInt(int64(*host.UserLvmSize), 10) + " GB"
	}

	_, _ = fmt.Fprintf(writer, "Host Info: \n\n")
	_, _ = fmt.Fprintf(writer, "-\tHost Resurce ID:\t %s\n", *host.ResourceId)
	_, _ = fmt.Fprintf(writer, "-\tName:\t %s\n", host.Name)
	_, _ = fmt.Fprintf(writer, "-\tOS Profile:\t %v\n", osprofile)
	_, _ = fmt.Fprintf(writer, "-\tNIC Name and IP Address:\t %v\n", ip)
	_, _ = fmt.Fprintf(writer, "-\tLVM Size:\t %v\n\n", lvmsize)

	_, _ = fmt.Fprintf(writer, "Status details: \n\n")
	_, _ = fmt.Fprintf(writer, "-\tHost Status:\t %s\n", hoststatus)
	_, _ = fmt.Fprintf(writer, "-\tHost Status Details:\t %s\n", hostdetails)
	_, _ = fmt.Fprintf(writer, "-\tProvisioning Status:\t %s\n", provstatus)
	_, _ = fmt.Fprintf(writer, "-\tUpdate Status:\t %s\n\n", updatestatus)

	_, _ = fmt.Fprintf(writer, "Specification: \n\n")
	_, _ = fmt.Fprintf(writer, "-\tSerial Number:\t %s\n", *host.SerialNumber)
	_, _ = fmt.Fprintf(writer, "-\tUUID:\t %s\n", *host.Uuid)
	_, _ = fmt.Fprintf(writer, "-\tOS:\t %v\n", currentOS)
	_, _ = fmt.Fprintf(writer, "-\tBIOS Vendor:\t %v\n", *host.BiosVendor)
	_, _ = fmt.Fprintf(writer, "-\tProduct Name:\t %v\n\n", *host.ProductName)

	_, _ = fmt.Fprintf(writer, "Customizations: \n\n")
	_, _ = fmt.Fprintf(writer, "-\tCustom configs:\t %s\n\n", customcfg)

	_, _ = fmt.Fprintf(writer, "CPU Info: \n\n")
	_, _ = fmt.Fprintf(writer, "-\tCPU Model:\t %v\n", *host.CpuModel)
	_, _ = fmt.Fprintf(writer, "-\tCPU Cores:\t %v\n", *host.CpuCores)
	_, _ = fmt.Fprintf(writer, "-\tCPU Architecture:\t %v\n", *host.CpuArchitecture)
	_, _ = fmt.Fprintf(writer, "-\tCPU Threads:\t %v\n", *host.CpuThreads)
	_, _ = fmt.Fprintf(writer, "-\tCPU Sockets:\t %v\n\n", *host.CpuSockets)

	if host.Instance != nil && host.Instance.ExistingCves != nil && host.Instance.CurrentOs != nil && host.Instance.CurrentOs.FixedCves != nil {

		if *host.Instance.ExistingCves != "" {
			err := json.Unmarshal([]byte(*host.Instance.ExistingCves), &cveEntries)
			if err != nil {
				fmt.Println("Error unmarshaling JSON: existing CVE entries:", err)
				return
			}
		}

		_, _ = fmt.Fprintf(writer, "CVE Info (existing CVEs): \n\n")
		for _, cve := range cveEntries {
			_, _ = fmt.Fprintf(writer, "-\tCVE ID:\t %v\n", cve.CVEID)
			_, _ = fmt.Fprintf(writer, "-\tPriority:\t %v\n", cve.Priority)
			_, _ = fmt.Fprintf(writer, "-\tAffected Packages:\t %v\n\n", cve.AffectedPackages)
		}
	}
	if host.CurrentAmtState != nil && *host.CurrentAmtState == infra.AMTSTATEPROVISIONED {
		_, _ = fmt.Fprintf(writer, "AMT Info: \n\n")
		_, _ = fmt.Fprintf(writer, "-\tAMT Status:\t %v\n", *host.CurrentAmtState)
		_, _ = fmt.Fprintf(writer, "-\tCurrent Power Status:\t %v\n", *host.CurrentPowerState)
		_, _ = fmt.Fprintf(writer, "-\tDesired Power Status:\t %v\n", *host.DesiredPowerState)
		_, _ = fmt.Fprintf(writer, "-\tPower Command Policy :\t %v\n", *host.PowerCommandPolicy)
		_, _ = fmt.Fprintf(writer, "-\tPowerOn Time :\t %v\n", *host.PowerOnTime)
		_, _ = fmt.Fprintf(writer, "-\tDesired AMT State :\t %v\n", *host.DesiredAmtState)
	}

	if host.CurrentAmtState != nil && *host.CurrentAmtState != infra.AMTSTATEPROVISIONED {
		_, _ = fmt.Fprintf(writer, "AMT not active and/or not supported: No info available \n\n")
	}

}

// Helper function to verify that the input file exists and is of right format
func verifyCSVInput(path string) error {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".csv" {
		return errors.New("host import input file must be a CSV file")
	}

	return nil
}

func generateCSV(filename string) error {
	// The CSV generation logic
	fmt.Printf("Generating empty CSV template file: %s\n", filename)
	return files.CreateFile(filename)
}

// Runs the registration workflow
func doRegister(ctx context.Context, ctx2 context.Context, hClient infra.ClientWithResponsesInterface, projectName string, rIn types.HostRecord, respCache ResponseCache, globalAttr *types.HostRecord, erringRecords *[]types.HostRecord, cClient cluster.ClientWithResponsesInterface) {

	// get the required fields from the record
	sNo := rIn.Serial
	uuid := rIn.UUID
	var lvmSize *int
	// predefine other fields
	hostName := ""
	hostID := ""
	autonboard := true

	rOut, err := sanitizeProvisioningFields(ctx, ctx2, hClient, projectName, rIn, respCache, globalAttr, erringRecords, cClient)
	if err != nil {
		return
	}

	if rOut.LVMSize != "" {
		if lvmInt, err := strconv.Atoi(rOut.LVMSize); err == nil {
			lvmSize = &lvmInt
		}
	}

	hostID, err = registerHost(ctx, hClient, respCache, projectName, hostName, sNo, uuid, autonboard, lvmSize)
	if err != nil {
		rIn.Error = err.Error()
		*erringRecords = append(*erringRecords, rIn)
		return
	}

	err = createInstance(ctx, hClient, respCache, projectName, hostID, rOut, rIn, globalAttr)
	if err != nil {
		rIn.Error = err.Error()
		*erringRecords = append(*erringRecords, rIn)
		return
	}

	err = allocateHostToSiteAndAddMetadata(ctx, hClient, projectName, hostID, rOut)
	if err != nil {
		rIn.Error = err.Error()
		*erringRecords = append(*erringRecords, rIn)
		return
	}

	if rOut.K8sEnable == "true" {
		err = createCluster(ctx2, cClient, respCache, projectName, hostID, rOut)
		if err != nil {
			rIn.Error = err.Error()
			*erringRecords = append(*erringRecords, rIn)
			return
		}
	}

	// Print host_id from response if successful
	fmt.Printf("✔ Host Serial number : %s  UUID : %s registered. Name : %s\n", sNo, uuid, hostID)
}

// Decodes the provided metadata from input string
func decodeMetadata(metadata string) (*[]infra.MetadataItem, error) {
	metadataList := make([]infra.MetadataItem, 0)
	if metadata == "" {
		return &metadataList, nil
	}
	metadataPairs := strings.Split(metadata, "&")
	for _, pair := range metadataPairs {
		kv := strings.Split(pair, "=")
		if len(kv) != kVSize {
			return &metadataList, e.NewCustomError(e.ErrInvalidMetadata)
		}
		mItem := infra.MetadataItem{
			Key:   kv[0],
			Value: kv[1],
		}
		metadataList = append(metadataList, mItem)
	}
	return &metadataList, nil
}

// Breaks up the provided cloud init metadata from input string
func breakupCloudInitMetadata(CImetadata string) *[]string {
	var CImetaList []string
	if CImetadata == "" {
		return &CImetaList
	}
	CImetaList = strings.Split(CImetadata, "&")

	return &CImetaList
}

func decodeK8sTemplate(ctemplate string) (string, string, error) {
	template := strings.Split(ctemplate, ":")

	if len(template) != 2 {
		return "", "", errors.New("invalid cluster template configuration")
	}
	tname := strings.TrimSpace(template[0])
	tver := strings.TrimSpace(template[1])

	return tname, tver, nil
}

func decodeK8sConfig(config string) (string, string, map[string]string, error) {

	if config == "" {
		config = "role:all"
	}

	configSplit := strings.Split(config, ";")
	roleSplit := strings.Split(strings.TrimSpace(configSplit[0]), ":")

	if strings.TrimSpace(roleSplit[0]) != "role" || len(roleSplit) < 2 {
		return "", "", nil, errors.New("invalid Cluster role configuration")
	}

	crole := roleSplit[1]
	if crole != "all" && crole != "worker" && crole != "controlplane" {
		return "", "", nil, errors.New("invalid Cluster role set")
	}

	cname := ""
	clabellist := ""
	clabels := make(map[string]string)
	//check if name was provided correctly
	if len(configSplit) > 1 {
		argSplit := strings.Split(strings.TrimSpace(configSplit[1]), ":")
		if strings.TrimSpace(argSplit[0]) == "name" && len(argSplit) < 2 {
			return "", "", nil, errors.New("invalid Cluster name configuration")
		} else if strings.TrimSpace(argSplit[0]) == "name" {
			cname = argSplit[1]
		}
		if strings.TrimSpace(argSplit[0]) == "labels" && len(argSplit) < 2 {
			return "", "", nil, errors.New("invalid label configuration")
		} else if strings.TrimSpace(argSplit[0]) == "labels" {
			clabellist = argSplit[1]
		}
		if strings.TrimSpace(argSplit[0]) != "labels" && strings.TrimSpace(argSplit[0]) != "name" {
			return "", "", nil, errors.New("invalid Cluster configuration")
		}
		if len(configSplit) > 2 {
			argSplit := strings.Split(strings.TrimSpace(configSplit[2]), ":")
			if strings.TrimSpace(argSplit[0]) == "labels" && len(argSplit) < 2 {
				return "", "", nil, errors.New("invalid label configuration")
			} else if strings.TrimSpace(argSplit[0]) == "labels" {
				clabellist = argSplit[1]
			} else {
				return "", "", nil, errors.New("invalid label configuration")
			}
		}

		labelPairs := strings.Split(clabellist, "&")
		for _, pair := range labelPairs {
			kv := strings.Split(pair, "=")
			if len(kv) == 2 {
				key := kv[0]
				value := kv[1]
				// Populate the map with the key-value pair
				clabels[key] = value
			}
		}
	}

	return cname, crole, clabels, nil
}

func resolveSecure(recordSecure, globalSecure types.RecordSecure) types.RecordSecure {
	if globalSecure != recordSecure && globalSecure != types.SecureUnspecified {
		return globalSecure
	}
	return recordSecure
}

// Sanitize fields, convert named resources to resource IDs
func sanitizeProvisioningFields(ctx context.Context, ctx2 context.Context, hClient infra.ClientWithResponsesInterface, projectName string, record types.HostRecord, respCache ResponseCache, globalAttr *types.HostRecord, erringRecords *[]types.HostRecord, cClient cluster.ClientWithResponsesInterface) (*types.HostRecord, error) {

	isSecure := resolveSecure(record.Secure, globalAttr.Secure)

	osProfileID, err := resolveOSProfile(ctx, hClient, projectName, record.OSProfile, globalAttr.OSProfile, record, respCache, erringRecords)
	if err != nil {
		return nil, err
	}

	if valErr := validateSecurityFeature(record.OSProfile, globalAttr.OSProfile, isSecure, record, respCache, erringRecords); valErr != nil {
		return nil, valErr
	}

	siteID, err := resolveSite(ctx, hClient, projectName, record.Site, globalAttr.Site, record, respCache, erringRecords)
	if err != nil {
		return nil, err
	}

	laID, err := resolveRemoteUser(ctx, hClient, projectName, record.RemoteUser, globalAttr.RemoteUser, record, respCache, erringRecords)
	if err != nil {
		return nil, err
	}

	metadataToUse := resolveMetadata(record.Metadata, globalAttr.Metadata)

	cloudInitIDs, err := resolveCloudInit(ctx, hClient, projectName, record.CloudInitMeta, globalAttr.CloudInitMeta, record, respCache, erringRecords)
	if err != nil {
		return nil, err
	}

	lvmSize := resolveLVMSize(record.LVMSize, globalAttr.LVMSize)

	isK8s := resolveCluster(record.K8sEnable, globalAttr.K8sEnable)
	k8sConfig := record.K8sConfig
	k8sTmplID := record.K8sClusterTemplate
	if isK8s == "true" {
		if record.K8sConfig != "" || globalAttr.K8sConfig != "" {
			k8sConfig, err = resolveClusterConfig(record.K8sConfig, globalAttr.K8sConfig)
			if err != nil {
				return nil, err
			}
		}

		if record.K8sClusterTemplate != "" || globalAttr.K8sClusterTemplate != "" {
			k8sTmplID, err = resolveClusterTemplate(ctx2, cClient, projectName, record.K8sClusterTemplate, globalAttr.K8sClusterTemplate, record, respCache, erringRecords)
			if err != nil {
				return nil, err
			}
		}
	}

	return &types.HostRecord{
		OSProfile:          osProfileID,
		RemoteUser:         laID,
		Site:               siteID,
		Secure:             isSecure,
		UUID:               record.UUID,
		Serial:             record.Serial,
		Metadata:           metadataToUse,
		LVMSize:            lvmSize,
		CloudInitMeta:      cloudInitIDs,
		K8sEnable:          isK8s,
		K8sClusterTemplate: k8sTmplID,
		K8sConfig:          k8sConfig,
	}, nil
}

// Ensures that OS profile exists
func resolveOSProfile(ctx context.Context, hClient infra.ClientWithResponsesInterface, projectName string, recordOSProfile string,
	globalOSProfile string, record types.HostRecord, respCache ResponseCache, erringRecords *[]types.HostRecord,
) (string, error) {

	osProfile := recordOSProfile

	if globalOSProfile != "" {
		osProfile = globalOSProfile
	}

	if osProfile == "" {
		record.Error = e.NewCustomError(e.ErrInvalidOSProfile).Error()
		*erringRecords = append(*erringRecords, record)
		return "", e.NewCustomError(e.ErrInvalidOSProfile)
	}

	// Check cache first
	if osResource, ok := respCache.OSProfileCache[osProfile]; ok {
		return *osResource.ResourceId, nil
	}

	ospfilter := fmt.Sprintf("name='%s' OR resourceId='%s'", osProfile, osProfile)
	resp, err := hClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
		&infra.OperatingSystemServiceListOperatingSystemsParams{
			Filter: &ospfilter,
		}, auth.AddAuthHeader)

	if err != nil {
		record.Error = err.Error()
		*erringRecords = append(*erringRecords, record)
		return "", err
	}

	if resp.JSON200 == nil || len(resp.JSON200.OperatingSystemResources) == 0 {
		record.Error = "OS Profile not found"
		*erringRecords = append(*erringRecords, record)
		return "", errors.New(record.Error)

	}

	// The API may return multiple OS profiles matching the filter
	// Filter results for exact matches
	var exactMatch *infra.OperatingSystemResource
	for _, ospResource := range resp.JSON200.OperatingSystemResources {
		// Check for exact name match or resource ID match
		if (ospResource.Name != nil && *ospResource.Name == osProfile) ||
			(ospResource.ResourceId != nil && *ospResource.ResourceId == osProfile) {
			exactMatch = &ospResource
			break // Take the first exact match, not the last
		}
	}

	if exactMatch == nil {
		record.Error = "OS Profile not found"
		*erringRecords = append(*erringRecords, record)
		return "", errors.New(record.Error)

	}

	// Cache the exact match
	respCache.OSProfileCache[osProfile] = *exactMatch
	return *exactMatch.ResourceId, nil
}

// Checks input security feature vs what is capable by host
func validateSecurityFeature(osProfileID string, globalOSProfile string, isSecure types.RecordSecure,
	record types.HostRecord, respCache ResponseCache, erringRecords *[]types.HostRecord,
) error {
	if globalOSProfile != "" {
		osProfileID = globalOSProfile
	}

	osProfile, ok := respCache.OSProfileCache[osProfileID]
	if !ok || (*osProfile.SecurityFeature != infra.SECURITYFEATURESECUREBOOTANDFULLDISKENCRYPTION && isSecure == types.SecureTrue) {
		record.Error = e.NewCustomError(e.ErrOSSecurityMismatch).Error()
		*erringRecords = append(*erringRecords, record)
		return e.NewCustomError(e.ErrOSSecurityMismatch)
	}
	return nil
}

// Validates the format of OS Profile ID
func validateOSProfile(osProfileID string) error {
	osRe := regexp.MustCompile(validator.OSPIDPATTERN)
	if !osRe.MatchString(osProfileID) {
		return e.NewCustomError(e.ErrInvalidOSProfile)
	}
	return nil
}

// Checks if site is valid and exists
func resolveSite(ctx context.Context, hClient infra.ClientWithResponsesInterface, projectName string, recordSite string,
	globalSite string, record types.HostRecord, respCache ResponseCache, erringRecords *[]types.HostRecord,
) (string, error) {

	siteToQuery := recordSite

	if globalSite != "" {
		siteToQuery = globalSite
	}

	if siteToQuery == "" {
		record.Error = e.NewCustomError(e.ErrInvalidSite).Error()
		*erringRecords = append(*erringRecords, record)
		return "", e.NewCustomError(e.ErrInvalidSite)
	}

	if siteResource, ok := respCache.SiteCache[siteToQuery]; ok {
		return *siteResource.ResourceId, nil
	}

	resp, err := hClient.SiteServiceGetSiteWithResponse(ctx, projectName, "regionID", siteToQuery, auth.AddAuthHeader)
	if err != nil {
		record.Error = err.Error()
		*erringRecords = append(*erringRecords, record)
		return "", err
	}

	err = checkResponse(resp.HTTPResponse, resp.Body, "error Site not found")
	if err != nil {
		record.Error = err.Error()
		*erringRecords = append(*erringRecords, record)
		return "", err
	}

	respCache.SiteCache[siteToQuery] = *resp.JSON200
	return *resp.JSON200.ResourceId, nil
}

// Checks if LVM size is valid
func resolveLVMSize(recordLVMSize string, globalLVMSize string) string {
	lvmSize := ""

	if recordLVMSize != "" {
		lvmSize = recordLVMSize
	}

	if globalLVMSize != "" {
		lvmSize = globalLVMSize
	}

	return lvmSize
}

// Checks if cluster deployment is enabled
func resolveCluster(recordClusterEnable string,
	globalClusterEnable string) string {

	isEnabled := recordClusterEnable

	if globalClusterEnable != "" {
		isEnabled = globalClusterEnable
	}

	if isEnabled != "true" {
		return ""
	}

	return isEnabled
}

// Checks if cluster template is valid and existss
func resolveClusterTemplate(ctx context.Context, cClient cluster.ClientWithResponsesInterface, projectName string, recordClusterTemplate string,
	globalClusterTemplate string, record types.HostRecord, respCache ResponseCache, erringRecords *[]types.HostRecord,
) (string, error) {

	remoteCTempToQuery := recordClusterTemplate

	if globalClusterTemplate != "" {
		remoteCTempToQuery = globalClusterTemplate
	}

	if remoteCTempToQuery == "" {
		return "", nil
	}

	if cTempResource, ok := respCache.K8sClusterTemplateCache[remoteCTempToQuery]; ok {
		return cTempResource.Name + ":" + cTempResource.Version, nil
	}

	template := strings.Split(remoteCTempToQuery, ":")

	resp, err := cClient.GetV2ProjectsProjectNameTemplatesNameVersionsVersionWithResponse(ctx, projectName,
		strings.TrimSpace(template[0]), strings.TrimSpace(template[1]), auth.AddAuthHeader)
	if err != nil {
		record.Error = err.Error()
		*erringRecords = append(*erringRecords, record)
		return "", err
	}
	if resp.JSON200 != nil {
		respCache.K8sClusterTemplateCache[remoteCTempToQuery] = *resp.JSON200
		return resp.JSON200.Name + ":" + resp.JSON200.Version, nil
	}
	record.Error = "Cluster Template not found"
	*erringRecords = append(*erringRecords, record)
	return "", errors.New(record.Error)
}

// Checks if cluster config is valid
func resolveClusterConfig(recordClusterConfig string, globalClusterConfig string) (string, error) {

	configToValidate := recordClusterConfig

	if globalClusterConfig != "" {
		configToValidate = globalClusterConfig
	}

	return configToValidate, nil
}

// Checks if remote user is valid and exists
func resolveRemoteUser(ctx context.Context, hClient infra.ClientWithResponsesInterface, projectName string, recordRemoteUser string,
	globalRemoteUser string, record types.HostRecord, respCache ResponseCache, erringRecords *[]types.HostRecord,
) (string, error) {

	remoteUserToQuery := recordRemoteUser

	if globalRemoteUser != "" {
		remoteUserToQuery = globalRemoteUser
	}

	if remoteUserToQuery == "" {
		return "", nil
	}

	if lAResource, ok := respCache.LACache[remoteUserToQuery]; ok {
		return *lAResource.ResourceId, nil
	}

	lafilter := fmt.Sprintf("username='%s' OR resourceId='%s'", remoteUserToQuery, remoteUserToQuery)
	resp, err := hClient.LocalAccountServiceListLocalAccountsWithResponse(ctx, projectName,
		&infra.LocalAccountServiceListLocalAccountsParams{
			Filter: &lafilter,
		}, auth.AddAuthHeader)
	if err != nil {
		record.Error = err.Error()
		*erringRecords = append(*erringRecords, record)
		return "", err
	}
	if resp.JSON200 != nil && resp.JSON200.LocalAccounts != nil {
		localAccounts := resp.JSON200.LocalAccounts
		if len(localAccounts) > 0 {
			respCache.LACache[remoteUserToQuery] = localAccounts[len(localAccounts)-1]
			return *localAccounts[len(localAccounts)-1].ResourceId, nil
		}
	}
	record.Error = "Remote User not found"
	*erringRecords = append(*erringRecords, record)
	return "", errors.New(record.Error)
}

// Cecks if remote user is valid and exists
func resolveCloudInit(ctx context.Context, hClient infra.ClientWithResponsesInterface, projectName string, recordCloudInitMeta string,
	globalCloudInitMeta string, record types.HostRecord, respCache ResponseCache, erringRecords *[]types.HostRecord,
) (string, error) {

	cloudInitMetaToQuery := recordCloudInitMeta

	if globalCloudInitMeta != "" {
		cloudInitMetaToQuery = globalCloudInitMeta
	}

	if cloudInitMetaToQuery == "" {
		return "", nil
	}

	rawCloudInitEntries := breakupCloudInitMetadata(cloudInitMetaToQuery)
	var sanCloudInitEntries []string
	wrongCloudInits := ""

	for _, cloudInit := range *rawCloudInitEntries {

		if cImetaResource, ok := respCache.CICache[cloudInit]; ok {
			sanCloudInitEntries = append(sanCloudInitEntries, *cImetaResource.ResourceId)
			continue
		}

		cImfilter := fmt.Sprintf("name='%s' OR resourceId='%s'", cloudInit, cloudInit)
		resp, err := hClient.CustomConfigServiceListCustomConfigsWithResponse(ctx, projectName,
			&infra.CustomConfigServiceListCustomConfigsParams{
				Filter: &cImfilter,
			}, auth.AddAuthHeader)
		if err != nil {
			record.Error = err.Error()
			*erringRecords = append(*erringRecords, record)
			continue
			//return "", err
		}
		if resp.JSON200 != nil && resp.JSON200.CustomConfigs != nil {
			cloudInits := resp.JSON200.CustomConfigs
			if len(cloudInits) > 0 {
				respCache.CICache[cloudInit] = cloudInits[len(cloudInits)-1]
				sanCloudInitEntries = append(sanCloudInitEntries, *cloudInits[len(cloudInits)-1].ResourceId)
				continue
			}
		}
		wrongCloudInits = wrongCloudInits + " " + cloudInit
	}

	if wrongCloudInits != "" {
		erroMsg := fmt.Sprintf("Remote Cloud Init custom config %s not found", wrongCloudInits)
		record.Error = erroMsg
		*erringRecords = append(*erringRecords, record)
		return "", errors.New(record.Error)
	}
	return strings.Join(sanCloudInitEntries, "&"), nil
}

func resolveMetadata(recordMetadata, globalMetadata string) string {
	if globalMetadata != "" {
		return globalMetadata
	}
	return recordMetadata
}

func getDeauthorizeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "deauthorize",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Deauthorize host",
		PersistentPreRunE: auth.CheckAuth,
	}

	cmd.AddCommand(
		getDeauthorizeHostCommand(),
	)
	return cmd
}

func getListHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host [flags]",
		Short:   "Lists all hosts",
		Example: listHostExamples,
		Aliases: hostAliases,
		RunE:    runListHostCommand,
	}

	// Local persistent flags
	cmd.PersistentFlags().StringP("filter", "f", viper.GetString("filter"), "Optional filter provided as part of host list command\nUsage:\n\tCustom filter: --filter \"<custom filter>\" ie. --filter \"osType=OS_TYPE_IMMUTABLE\" see https://google.aip.dev/160 and API spec. \n\tPredefined filters: --filter provisioned/onboarded/registered/nor connected/deauthorized")
	cmd.PersistentFlags().StringP("site", "s", viper.GetString("site"), "Optional filter provided as part of host list to filter hosts by site")
	cmd.PersistentFlags().StringP("region", "r", viper.GetString("region"), "Optional filter provided as part of host list to filter hosts by region")
	cmd.PersistentFlags().StringP("workload", "w", viper.GetString("workload"), "Optional filter provided as part of host list to filter hosts by workload")
	return cmd
}

func getGetHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host <resourceID> [flags]",
		Short:   "Gets a host",
		Example: getHostExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: hostAliases,
		RunE:    runGetHostCommand,
	}
	return cmd
}

func getCreateHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host --import-from-csv]",
		Short:   "Provisions a host or hosts",
		Example: createHostExamples,
		Aliases: hostAliases,
		RunE:    runCreateHostCommand,
	}

	// Local persistent flags
	cmd.PersistentFlags().StringP("import-from-csv", "i", viper.GetString("import-from-csv"), "CSV file containing information about to be provisioned hosts")
	cmd.PersistentFlags().BoolP("dry-run", "d", viper.GetBool("dry-run"), "Verify the validity of input CSV file")
	cmd.PersistentFlags().StringP("generate-csv", "g", viper.GetString("generate-csv"), "Generates a template CSV file for host import")
	cmd.PersistentFlags().Lookup("generate-csv").NoOptDefVal = filename
	// Overrides
	cmd.PersistentFlags().StringP("os-profile", "o", viper.GetString("os-profile"), "Override the OSProfile provided in CSV file for all hosts")
	cmd.PersistentFlags().StringP("site", "s", viper.GetString("site"), "Override the site provided in CSV file for all hosts")
	cmd.PersistentFlags().StringP("metadata", "m", viper.GetString("metadata"), "Override the metadata provided in CSV file for all hosts")
	cmd.PersistentFlags().StringP("remote-user", "r", viper.GetString("remote-user"), "Override the metadata provided in CSV file for all hosts")
	cmd.PersistentFlags().StringP("cluster-deploy", "c", viper.GetString("cluster-deploy"), "Override the cluster deployment flag provided in CSV file for all hosts")
	cmd.PersistentFlags().StringP("cluster-template", "t", viper.GetString("cluster-template"), "Override the cluster template provided in CSV file for all hosts")
	cmd.PersistentFlags().StringP("cluster-config", "f", viper.GetString("cluster-config"), "Override the cluster configuration provided in CSV file for all hosts")
	cmd.PersistentFlags().StringP("cloud-init", "j", viper.GetString("cloud-init"), "Override the cloud init metadata provided in CSV file for all hosts")
	cmd.PersistentFlags().StringP("secure", "x", viper.GetString("secure"), "Override the security feature configuration provided in CSV file for all hosts")
	cmd.PersistentFlags().StringP("lvm-size", "l", viper.GetString("lvm-size"), "Override the LVM size configuration provided in CSV file for all hosts")

	return cmd
}

func getDeleteHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host <resourceID> [flags]",
		Short:   "Deletes a host and associated instance",
		Example: deleteHostExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: hostAliases,
		RunE:    runDeleteHostCommand,
	}
	return cmd
}

func getSetHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host <resourceID> [flags]",
		Short:   "Sets a host attribute or action",
		Example: setHostExamples,
		Args: func(cmd *cobra.Command, args []string) error {
			generateCSV, _ := cmd.Flags().GetString("generate-csv")
			if generateCSV == "" {
				generateCSV = "test.csv"
			}
			importCSV, _ := cmd.Flags().GetString("import-from-csv")
			if generateCSV != "" || importCSV != "" {
				// No positional arg required for bulk operations
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
			}
			return nil
		},
		Aliases: hostAliases,
		RunE:    runSetHostCommand,
	}
	cmd.PersistentFlags().StringP("import-from-csv", "i", viper.GetString("import-from-csv"), "CSV file containing information about provisioned hosts")
	cmd.PersistentFlags().BoolP("dry-run", "d", viper.GetBool("dry-run"), "Verify the validity of input CSV file")
	cmd.PersistentFlags().StringP("generate-csv", "g", viper.GetString("generate-csv"), "Generates a template CSV file for host import")
	cmd.PersistentFlags().Lookup("generate-csv").NoOptDefVal = filename
	cmd.PersistentFlags().StringP("power", "r", viper.GetString("power"), "Power on|off|cycle|hibernate|reset|sleep")
	cmd.PersistentFlags().StringP("power-policy", "c", viper.GetString("power-policy"), "Set power policy immediate|ordered")
	cmd.PersistentFlags().StringP("amt-state", "a", viper.GetString("amt-state"), "Set AMT state <provisioned|unprovisioned>")
	cmd.PersistentFlags().StringP("osupdatepolicy", "u", viper.GetString("osupdatepolicy"), "Set OS update policy <resourceID>")

	return cmd
}

func getDeauthorizeHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host <resourceID> [flags]",
		Short:   "Deauthorizes a host",
		Example: deauthorizeHostExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: hostAliases,
		RunE:    runDeauthorizeHostCommand,
	}
	return cmd
}

// Lists all Hosts - retrieves all hosts and displays selected information in tabular format
func runListHostCommand(cmd *cobra.Command, _ []string) error {

	workload, _ := cmd.Flags().GetString("workload")
	filtflag, _ := cmd.Flags().GetString("filter")
	filter := filterHelper(filtflag)

	siteFlag, _ := cmd.Flags().GetString("site")
	site, err := filterSitesHelper(siteFlag)
	if err != nil {
		return err
	}

	regFlag, _ := cmd.Flags().GetString("region")
	region, err := filterRegionsHelper(regFlag)
	if err != nil {
		return err
	}

	if siteFlag != "" && regFlag != "" {
		fmt.Printf("--region flag ignored, using --site as it is more precise")
	}

	writer, verbose := getOutputContext(cmd)

	ctx, hostClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	//If all host for a given region are queried, sites need to be found first
	if siteFlag == "" && regFlag != "" {

		regFilter := fmt.Sprintf("region.resource_id='%s' OR region.parent_region.resource_id='%s' OR region.parent_region.parent_region.resource_id='%s' OR region.parent_region.parent_region.parent_region.resource_id='%s'", regFlag, regFlag, regFlag, regFlag)

		cresp, err := hostClient.SiteServiceListSitesWithResponse(ctx, projectName, *region,
			&infra.SiteServiceListSitesParams{
				Filter: &regFilter,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}

		//create site filter
		siteFilter := ""
		if cresp.JSON200.TotalElements != 0 {
			for i, s := range cresp.JSON200.Sites {
				if i == 0 {
					siteFilter = fmt.Sprintf("site.resourceId='%s'", *s.ResourceId)
				} else {
					siteFilter = fmt.Sprintf("%s OR site.resourceId='%s'", siteFilter, *s.ResourceId)
				}
			}
		} else {
			return errors.New("no site was found in provided region")
		}

		//if additional filter exists add sites to that filter if not replace empty filter with sites
		if filtflag != "" {
			*filter = fmt.Sprintf("%s AND (%s)", *filter, siteFilter)
		} else {
			filter = &siteFilter
		}
	}

	if siteFlag != "" {
		siteFilter := fmt.Sprintf("site.resourceId='%s'", *site)
		if filtflag != "" {
			*filter = fmt.Sprintf("%s AND (%s)", *filter, siteFilter)
		} else {
			filter = &siteFilter
		}
	}

	pageSize := 20
	hosts := make([]infra.HostResource, 0)
	for offset := 0; ; offset += pageSize {
		resp, err := hostClient.HostServiceListHostsWithResponse(ctx, projectName,
			&infra.HostServiceListHostsParams{
				Filter:   filter,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}

		if err := checkResponse(resp.HTTPResponse, resp.Body, "error while retrieving hosts"); err != nil {
			return err
		}
		hosts = append(hosts, resp.JSON200.Hosts...)
		if !resp.JSON200.HasNext {
			break // No more hosts to process
		}
	}

	// Get instances in order to map additional host details
	instances := make([]infra.InstanceResource, 0)
	for offset := 0; ; offset += pageSize {
		iresp, err := hostClient.InstanceServiceListInstancesWithResponse(ctx, projectName,
			&infra.InstanceServiceListInstancesParams{
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(iresp.HTTPResponse, iresp.Body, "error while retrieving instance"); err != nil {
			return err
		}
		instances = append(instances, iresp.JSON200.Instances...)
		if !iresp.JSON200.HasNext {
			break // No more instances to process
		}
	}
	matchedHosts := make([]infra.HostResource, 0)
	notMatchedHosts := make([]infra.HostResource, 0)

	//Map workloads to hosts
	for _, host := range hosts {
		for _, instance := range instances {
			if instance.WorkloadMembers != nil && instance.InstanceID != nil && host.Instance != nil && host.Instance.InstanceID != nil && *instance.InstanceID == *host.Instance.InstanceID {
				host.Instance.WorkloadMembers = instance.WorkloadMembers
				if workload != "" && len(*host.Instance.WorkloadMembers) > 0 {
					if *(*host.Instance.WorkloadMembers)[0].Workload.Name == workload {
						matchedHosts = append(matchedHosts, host)
					}
				}
				break
			}
		}
		if workload == "NotAssigned" {
			if (host.Instance != nil && len(*host.Instance.WorkloadMembers) == 0) || host.Instance == nil {
				notMatchedHosts = append(notMatchedHosts, host)
			}
		}
	}

	if workload != "" {
		hosts = matchedHosts
	}

	if workload == "NotAssigned" {
		hosts = notMatchedHosts
	}

	printHosts(writer, &hosts, verbose)
	if verbose {
		if filter != nil {
			fmt.Fprintf(writer, "\nTotal Hosts (filter: %v): %d\n", *filter, len(hosts))
		} else {
			fmt.Fprintf(writer, "\nTotal Hosts: %d\n", len(hosts))
		}
	}
	return writer.Flush()
}

// Gets specific Host - retrieves a host using resource ID and displays detailed information
func runGetHostCommand(cmd *cobra.Command, args []string) error {

	hostID := args[0]
	writer, verbose := getOutputContext(cmd)
	ctx, hostClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := hostClient.HostServiceGetHostWithResponse(ctx, projectName,
		hostID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		hostHeaderGet, "error getting Host"); !proceed {
		return err
	}

	var instanceID *string
	if resp.JSON200.Instance != nil && resp.JSON200.Instance.InstanceID != nil {
		instanceID = resp.JSON200.Instance.InstanceID

		iresp, err := hostClient.InstanceServiceGetInstanceWithResponse(ctx, projectName,
			*instanceID, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}

		if proceed, err := processResponse(iresp.HTTPResponse, resp.Body, writer, verbose,
			"", "error getting instance of a host"); !proceed {
			return err
		}

		resp.JSON200.Instance = iresp.JSON200
	}

	printHost(writer, resp.JSON200)
	return writer.Flush()
}

// Lists all Hosts - retrieves all hosts and displays selected information in tabular format
func runCreateHostCommand(cmd *cobra.Command, _ []string) error {

	currentPath, err := os.Getwd()
	if err != nil {
		fmt.Println("Error finding current path for template generation:", err)
		return err
	}

	generate, _ := cmd.Flags().GetString("generate-csv")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	csvFilePath, _ := cmd.Flags().GetString("import-from-csv")
	osProfileIn, _ := cmd.Flags().GetString("os-profile")
	siteIn, _ := cmd.Flags().GetString("site")
	metadataIn, _ := cmd.Flags().GetString("metadata")
	cloudInitIn, _ := cmd.Flags().GetString("cloud-init")
	remoteUserIn, _ := cmd.Flags().GetString("remote-user")
	secureIn, _ := cmd.Flags().GetString("secure")
	k8sIn, _ := cmd.Flags().GetString("cluster-deploy")
	k8sTmplIn, _ := cmd.Flags().GetString("cluster-template")
	k8sConfigIn, _ := cmd.Flags().GetString("cluster-config")
	lvmIn, _ := cmd.Flags().GetString("lvm-size")

	globalAttr := &types.HostRecord{
		OSProfile:          osProfileIn,
		Site:               siteIn,
		Secure:             types.StringToRecordSecure(secureIn),
		RemoteUser:         remoteUserIn,
		Metadata:           metadataIn,
		LVMSize:            lvmIn,
		K8sEnable:          k8sIn,
		K8sClusterTemplate: k8sTmplIn,
		K8sConfig:          k8sConfigIn,
		CloudInitMeta:      cloudInitIn,
	}

	if cmd.Flags().Changed("generate-csv") && (dryRun || csvFilePath != "") {
		return fmt.Errorf("cannot use --generate-csv flag with --dry-run and/or --import-from-csv")
	}

	if cmd.Flags().Changed("generate-csv") {
		if generate != filename {
			filename = generate
		}
		if strings.HasSuffix(filename, ".csv") {
			err = generateCSV(fmt.Sprintf("%s/%s", currentPath, filename))
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("--generate-csv requires that file name ends with .csv")
		}
		return nil
	}

	if csvFilePath == "" || strings.HasPrefix(csvFilePath, "--") {
		return fmt.Errorf("--import-from-csv <path/to/file.csv> is required, cannot be empty")
	}

	err = verifyCSVInput(csvFilePath)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Println("--dry-run flag provided, validating input, hosts will not be imported")
		_, err := validator.CheckCSV(csvFilePath, *globalAttr)
		if err != nil {
			return err
		}
		fmt.Println("CSV validation successful")
		return nil
	}

	validated, err := validator.CheckCSV(csvFilePath, *globalAttr)
	if err != nil {
		return err
	}

	respCache := ResponseCache{
		OSProfileCache:          make(map[string]infra.OperatingSystemResource),
		SiteCache:               make(map[string]infra.SiteResource),
		LACache:                 make(map[string]infra.LocalAccountResource),
		HostCache:               make(map[string]infra.HostResource),
		K8sClusterTemplateCache: make(map[string]cluster.TemplateInfo),
		K8sClusterNodesCache:    make(map[string][]cluster.NodeSpec),
		CICache:                 make(map[string]infra.CustomConfigResource),
	}

	ctx, hostClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	ctx2, clusterClient, _, err := ClusterFactory(cmd)
	if err != nil {
		return err
	}

	erringRecords := []types.HostRecord{}

	for _, record := range validated {
		doRegister(ctx, ctx2, hostClient, projectName, record, respCache, globalAttr, &erringRecords, clusterClient)
	}

	if len(erringRecords) > 0 {
		newFilename := fmt.Sprintf("%s_%s_%s", "import_error",
			time.Now().Format(time.RFC3339), filepath.Base(currentPath))
		fmt.Printf("Generating error file: %s\n", newFilename)
		if err := files.WriteHostRecords(newFilename, erringRecords); err != nil {
			return e.NewCustomError(e.ErrFileRW)
		}
		return e.NewCustomError(e.ErrImportFailed)
	}

	return nil

}

// Deletes specific Host - finds a host using resource ID and deletes it
func runDeleteHostCommand(cmd *cobra.Command, args []string) error {
	hostID := args[0]
	ctx, hostClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	// retrieve the host (to check if it has an instance associated with it)
	resp1, err := hostClient.HostServiceGetHostWithResponse(ctx, projectName, hostID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp1.HTTPResponse, resp1.Body, "error while retrieving host"); err != nil {
		return err
	}
	host := *resp1.JSON200

	// delete the instance if it exists
	if host.Instance != nil {
		instanceID := host.Instance.InstanceID

		if instanceID != nil && *instanceID != "" {
			resp2, err := hostClient.InstanceServiceDeleteInstanceWithResponse(ctx, projectName, *instanceID, auth.AddAuthHeader)
			if err != nil {
				return processError(err)
			}
			if err := checkResponse(resp2.HTTPResponse, resp2.Body, "error while deleting instance"); err != nil {
				return err
			}
		}
	}

	// delete the host
	resp3, err := hostClient.HostServiceDeleteHostWithResponse(ctx, projectName,
		hostID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp3.HTTPResponse, resp3.Body, "error while deleting host"); err != nil {
		return err
	}
	fmt.Printf("Host %s deleted successfully\n", hostID)
	return nil
}

// Set attributes for specific Host - finds a host using resource ID
func runSetHostCommand(cmd *cobra.Command, args []string) error {

	generateCSV, _ := cmd.Flags().GetString("generate-csv")
	importCSV, _ := cmd.Flags().GetString("import-from-csv")
	policyFlag, _ := cmd.Flags().GetString("power-policy")
	powerFlag, _ := cmd.Flags().GetString("power")
	updFlag, _ := cmd.Flags().GetString("osupdatepolicy")
	amtFlag, _ := cmd.Flags().GetString("amt-state")

	// Bulk CSV generation
	if generateCSV != "" {
		// Fetch all hosts (reuse your list logic)
		ctx, hostClient, projectName, err := InfraFactory(cmd)
		if err != nil {
			return err
		}
		pageSize := 100
		hosts := make([]infra.HostResource, 0)
		for offset := 0; ; offset += pageSize {
			resp, err := hostClient.HostServiceListHostsWithResponse(ctx, projectName,
				&infra.HostServiceListHostsParams{
					PageSize: &pageSize,
					Offset:   &offset,
				}, auth.AddAuthHeader)
			if err != nil {
				return processError(err)
			}
			hosts = append(hosts, resp.JSON200.Hosts...)
			if !resp.JSON200.HasNext {
				break
			}
		}
		// Write CSV
		// Use absolute path for CSV file if not already absolute
		csvPath := generateCSV
		if !filepath.IsAbs(csvPath) {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			csvPath = filepath.Join(wd, csvPath)
		}
		// Check if file already exists
		if _, err := os.Stat(csvPath); err == nil {
			fmt.Printf("File %s already exists not generating\n", csvPath)
			return nil
		}
		f, err := os.Create(csvPath)
		if err != nil {
			return err
		}
		defer f.Close()
		fmt.Fprintln(f, "Name,ResourceID,DesiredAmtState")
		for _, h := range hosts {
			name := h.Name
			resourceID := ""
			desiredAmtState := ""
			if h.ResourceId != nil {
				resourceID = *h.ResourceId
			}
			if h.DesiredAmtState != nil {
				desiredAmtState = string(*h.DesiredAmtState)
			}
			fmt.Fprintf(f, "%s,%s,%s\n", name, resourceID, desiredAmtState)
		}
		fmt.Printf("CSV template generated: %s\n", generateCSV)
		return nil
	}

	// Bulk CSV import
	if importCSV != "" {
		file, err := os.Open(importCSV)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			line := scanner.Text()
			lineNum++
			if lineNum == 1 {
				continue // skip header
			}
			fields := strings.Split(line, ",")
			if len(fields) < 3 {
				fmt.Printf("Skipping invalid line %d: %s\n", lineNum, line)
				continue
			}
			name := strings.TrimSpace(fields[0])
			resourceID := strings.TrimSpace(fields[1])
			desiredAmtState := strings.TrimSpace(fields[2])
			// Validate desiredAmtState
			amtState, err := resolveAmtState(desiredAmtState)
			if err != nil {
				fmt.Printf("Invalid AMT state for host %s: %s\n", name, desiredAmtState)
				continue
			}
			// Patch host
			ctx, hostClient, projectName, err := InfraFactory(cmd)
			if err != nil {
				fmt.Printf("InfraFactory error for host %s: %v\n", name, err)
				continue
			}
			resp, err := hostClient.HostServicePatchHostWithResponse(ctx, projectName, resourceID, infra.HostServicePatchHostJSONRequestBody{
				DesiredAmtState: &amtState,
			}, auth.AddAuthHeader)
			if err != nil {
				fmt.Printf("Failed to patch host %s: %v\n", name, err)
				continue
			}
			if err := checkResponse(resp.HTTPResponse, resp.Body, "error while executing host set for AMT"); err != nil {
				fmt.Printf("Failed to patch host %s: %v\n", name, err)
				continue
			}
			fmt.Printf("Host %s (%s) AMT state updated to %s\n", name, resourceID, desiredAmtState)
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("no host ID provided")
	}
	hostID := args[0]

	if (policyFlag == "" || strings.HasPrefix(policyFlag, "--")) && (powerFlag == "" || strings.HasPrefix(powerFlag, "--")) && updFlag == "" && (amtFlag == "" || strings.HasPrefix(amtFlag, "--")) {
		return errors.New("a flag must be provided with the set host command and value cannot be \"\"")
	}

	var power *infra.PowerState
	var policy *infra.PowerCommandPolicy
	var updatePolicy *string
	var amtState *infra.AmtState

	if policyFlag != "" {
		pol, err := resolvePowerPolicy(policyFlag)
		if err != nil {
			return err
		}
		policy = &pol
	}

	if powerFlag != "" {
		pow, err := resolvePower(powerFlag)
		if err != nil {
			return err
		}
		power = &pow
	}

	if updFlag != "" {
		updatePolicy = &updFlag
	}

	if amtFlag != "" {
		amt, err := resolveAmtState(amtFlag)
		if err != nil {
			return err
		}
		amtState = &amt
	}

	ctx, hostClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	// retrieve the host (to check if it has an instance associated with it)
	iresp, err := hostClient.HostServiceGetHostWithResponse(ctx, projectName, hostID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(iresp.HTTPResponse, iresp.Body, "error while retrieving host"); err != nil {
		return err
	}
	host := *iresp.JSON200

	if (powerFlag != "" || policyFlag != "") && host.Instance != nil {
		resp, err := hostClient.HostServicePatchHostWithResponse(ctx, projectName, hostID, infra.HostServicePatchHostJSONRequestBody{
			PowerCommandPolicy: policy,
			DesiredPowerState:  power,
			Name:               host.Name,
		}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error while executing host set for AMT"); err != nil {
			return err
		}
	}

	if updatePolicy != nil && host.Instance != nil && host.Instance.InstanceID != nil && updFlag != "" {
		resp, err := hostClient.InstanceServicePatchInstanceWithResponse(ctx, projectName, *host.Instance.InstanceID, infra.InstanceServicePatchInstanceJSONRequestBody{
			OsUpdatePolicyID: updatePolicy,
		}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error while executing host set OS update policy"); err != nil {
			return err
		}
	}

	if amtState != nil && host.Instance != nil {
		resp, err := hostClient.HostServicePatchHostWithResponse(ctx, projectName, hostID, infra.HostServicePatchHostJSONRequestBody{
			DesiredAmtState: amtState,
			Name:            host.Name,
		}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error while executing host set for AMT"); err != nil {
			return err
		}
	}

	fmt.Printf("Host %s updated successfully\n", hostID)

	return nil
}

// Deauthorizes specific Host - finds a host using resource ID and invalidates it
func runDeauthorizeHostCommand(cmd *cobra.Command, args []string) error {
	hostID := args[0]
	ctx, hostClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := hostClient.HostServiceInvalidateHostWithResponse(ctx, projectName,
		hostID, &infra.HostServiceInvalidateHostParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, resp.Body, "error while invalidating host")
}

// Function containing the logic to register the host and retrieve the host ID
func registerHost(ctx context.Context, hClient infra.ClientWithResponsesInterface, respCache ResponseCache, projectName, hostName, sNo, uuid string, autonboard bool, lvmsize *int) (string, error) {
	// Register host

	resp, err := hClient.HostServiceRegisterHostWithResponse(ctx, projectName,
		infra.HostServiceRegisterHostJSONRequestBody{
			Name:         &hostName,
			SerialNumber: &sNo,
			Uuid:         &uuid,
			AutoOnboard:  &autonboard,
			UserLvmSize:  lvmsize,
		}, auth.AddAuthHeader)
	if err != nil {
		return "", processError(err)
	}
	//Check that valid response was received
	err = checkResponse(resp.HTTPResponse, resp.Body, "error while registering host")
	if err != nil {

		// Check if a host was already registred
		if strings.Contains(string(resp.Body), `"code":"FailedPrecondition"`) {
			//form a filter
			hFilter := fmt.Sprintf("serialNumber='%s' AND uuid='%s'", sNo, uuid)

			//get all the hosts matching the filter
			gresp, err := hClient.HostServiceListHostsWithResponse(ctx, projectName,
				&infra.HostServiceListHostsParams{
					Filter: &hFilter,
				}, auth.AddAuthHeader)
			if err != nil {
				return "", processError(err)
			}

			err = checkResponse(gresp.HTTPResponse, gresp.Body, "error while getting host which failed registration")
			if err != nil {
				return "", err
			}

			if gresp.JSON200.TotalElements != 1 {
				err = e.NewCustomError(e.ErrHostDetailMismatch)
				return "", err
			}

			//If the exact host was already registered cache it - then skip instance creation elsewhere if discovered host has instance assigned
			respCache.HostCache[*(gresp.JSON200.Hosts)[0].ResourceId] = (gresp.JSON200.Hosts)[0]
			return *(gresp.JSON200.Hosts)[0].ResourceId, nil

		}
		return "", err
	}

	//Cache host and save host ID
	if resp.JSON200 != nil && resp.JSON200.ResourceId != nil {
		respCache.HostCache[*resp.JSON200.ResourceId] = *resp.JSON200
		return *resp.JSON200.ResourceId, nil
	}
	return "", errors.New("host not found")

}

// If a valid OE Profile exists creates an instance linking to host resource
func createInstance(ctx context.Context, hClient infra.ClientWithResponsesInterface, respCache ResponseCache,
	projectName, hostID string, rOut *types.HostRecord, rIn types.HostRecord, globalAttr *types.HostRecord) error {

	//Create instance if not already created in a previous run of create host command
	if respCache.HostCache[hostID].Instance == nil {
		// Validate OS profile
		if valErr := validateOSProfile(rOut.OSProfile); valErr != nil {
			return valErr
		}

		cachedProfileIndex := rIn.OSProfile
		if globalAttr.OSProfile != "" {
			cachedProfileIndex = globalAttr.OSProfile
		}
		// Create instance if osProfileID is available
		// Need not notify user of instance ID. Unnecessary detail for user.
		kind := infra.INSTANCEKINDUNSPECIFIED
		osResource, ok := respCache.OSProfileCache[cachedProfileIndex]
		if !ok {
			return e.NewCustomError(e.ErrInternal)
		}

		secFeat := *osResource.SecurityFeature
		if rOut.Secure != types.SecureTrue {
			secFeat = infra.SECURITYFEATURENONE
		}

		var locAcc *string
		if rOut.RemoteUser != "" {
			locAcc = &rOut.RemoteUser
		}

		var cloudInitIDs *[]string
		if rOut.CloudInitMeta != "" {
			cloudInitIDs = breakupCloudInitMetadata(rOut.CloudInitMeta)
		}

		iresp, err := hClient.InstanceServiceCreateInstanceWithResponse(ctx, projectName,
			infra.InstanceServiceCreateInstanceJSONRequestBody{
				HostID:          &hostID,
				OsID:            &rOut.OSProfile,
				LocalAccountID:  locAcc,
				SecurityFeature: &secFeat,
				Kind:            &kind,
				CustomConfigID:  cloudInitIDs,
			}, auth.AddAuthHeader)
		if err != nil {
			err := processError(err)
			return err
		}

		err = checkResponse(iresp.HTTPResponse, iresp.Body, "error while creating instance\n\n")
		if err != nil {
			return err
		}

		return nil
	}
	if respCache.HostCache[hostID].Instance != nil && rOut.K8sEnable != "true" {
		return errors.New("host already registered")
	}
	return nil
}

// Create a cluster
func createCluster(ctx context.Context, cClient cluster.ClientWithResponsesInterface, respCache ResponseCache,
	projectName, hostID string, rOut *types.HostRecord) error {

	clusterTemplateName, clusterTempalteVer, err := decodeK8sTemplate(rOut.K8sClusterTemplate)
	if err != nil {
		return err
	}
	clusterName, clusterRole, clusterLabels, err := decodeK8sConfig(rOut.K8sConfig)
	if err != nil {
		return err
	}

	if clusterName == "" {
		clusterName = hostID
	}

	node := cluster.NodeSpec{
		Id:   rOut.UUID,
		Role: cluster.NodeSpecRole(clusterRole),
	}

	if respCache.K8sClusterNodesCache[clusterName] == nil {
		respCache.K8sClusterNodesCache[clusterName] = []cluster.NodeSpec{}
	}
	respCache.K8sClusterNodesCache[clusterName] = append(respCache.K8sClusterNodesCache[clusterName], node)
	nodes := respCache.K8sClusterNodesCache[clusterName]

	if len(nodes) == 1 {

		template := clusterTemplateName + "-" + clusterTempalteVer
		resp, err := cClient.PostV2ProjectsProjectNameClustersWithResponse(ctx, projectName, cluster.PostV2ProjectsProjectNameClustersJSONRequestBody{
			Name:     &clusterName,
			Nodes:    nodes,
			Template: &template,
			Labels:   &clusterLabels,
		}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}

		if resp.JSON201 != nil {
			return nil
		}

		err = checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error creating cluster %s", clusterName))
		if err != nil {
			if strings.Contains(string(resp.Body), `already exists`) {
				return errors.New("cluster already exists")
			}
			return err
		}
		return err
	}

	//Multinode currently not supported - return error if multiple cluster with same name requested instead
	// if len(nodes) > 1 {
	// 	resp, err := cClient.PutV2ProjectsProjectNameClustersNameNodesWithResponse(ctx, projectName, clusterName, nodes, auth.AddAuthHeader)
	// 	if err != nil {
	// 		return processError(err)
	// 	}
	// 	return checkResponse(resp.HTTPResponse,  resp.Body, fmt.Sprintf("error adding host to a cluster cluster %s", clusterName))
	// }
	if len(nodes) > 1 {
		return errors.New("only single node clusters currently supported - two clusters with same name requested")
	}

	return errors.New("error getting node(s) for cluster creation/expansion")
}

// Decode input metadata and add to host, allocate host to site
func allocateHostToSiteAndAddMetadata(ctx context.Context, hClient infra.ClientWithResponsesInterface,
	projectName, hostID string, rOut *types.HostRecord) error {

	// Update host with Site and metadata
	var metadata *[]infra.MetadataItem
	var err error
	if rOut.Metadata != "" {
		metadata, err = decodeMetadata(rOut.Metadata)
		if err != nil {
			return err
		}
	}

	sresp, err := hClient.HostServicePatchHostWithResponse(ctx, projectName, hostID,
		infra.HostServicePatchHostJSONRequestBody{
			Name:     hostID,
			Metadata: metadata,
			SiteId:   &rOut.Site,
		}, auth.AddAuthHeader)
	if err != nil {
		err := processError(err)
		return err
	}

	err = checkResponse(sresp.HTTPResponse, sresp.Body, "error while linking site and metadata\n\n")
	if err != nil {
		return err
	}

	return nil
}

func resolvePowerPolicy(power string) (infra.PowerCommandPolicy, error) {
	switch power {
	case "immediate":
		return infra.POWERCOMMANDPOLICYIMMEDIATE, nil
	case "ordered":
		return infra.POWERCOMMANDPOLICYORDERED, nil
	default:
		return "", errors.New("incorrect power policy provided with --power-policy flag use one of immediate|ordered")
	}
}

func resolvePower(power string) (infra.PowerState, error) {
	switch power {
	case "on":
		return infra.POWERSTATEON, nil
	case "off":
		return infra.POWERSTATEOFF, nil
	case "cycle":
		return infra.POWERSTATEPOWERCYCLE, nil
	case "hibernate":
		return infra.POWERSTATEHIBERNATE, nil
	case "reset":
		return infra.POWERSTATERESET, nil
	case "sleep":
		return infra.POWERSTATESLEEP, nil
	default:
		return "", errors.New("incorrect power action provided with --power flag use one of on|off|cycle|hibernate|reset|sleep")
	}
}

func resolveAmtState(amt string) (infra.AmtState, error) {
	switch amt {
	case "provisioned", "AMT_STATE_PROVISIONED":
		return infra.AMTSTATEPROVISIONED, nil
	case "unprovisioned", "AMT_STATE_UNPROVISIONED":
		return infra.AMTSTATEUNPROVISIONED, nil
	default:
		return "", errors.New("incorrect AMT state provided with --amt-state flag use one of provisioned|unprovisioned")
	}
}
