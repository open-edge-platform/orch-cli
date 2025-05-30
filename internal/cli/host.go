// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	u "github.com/google/uuid"
	e "github.com/open-edge-platform/cli/internal/errors"
	"github.com/open-edge-platform/cli/internal/files"
	"github.com/open-edge-platform/cli/internal/types"
	"github.com/open-edge-platform/cli/internal/validator"
	"github.com/open-edge-platform/cli/pkg/auth"
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

# List hosts using in a specific site uing site ID (--site flag will take precedence over --region flag)
orch-cli list host --project some-project --site site-c69a3c81

# List hosts using in a specific region uing region ID (--site flag will take precedence over --region flag)
orch-cli list host --project some-project --region region-1234abcd
`

const getHostExamples = `# Get detailed information about specific host using the host Resource ID
orch-cli get host host-1234abcd --project some-project`

const registerHostExamples = `# Register a host with a name "my-host" to an Edge Orchestrator using a Serial number of the machine and/or it's UUID.
orch-cli register host my-host --project some-project --serial 12345678 --uuid 0e4ec196-d1c4-4d81-9870-f202ebb498cc

orch-cli register host my-host --project some-project --serial 12345678

orch-cli register host my-host --project some-project --uuid 0e4ec196-d1c4-4d81-9870-f202ebb498cc`

const createHostExamples = "# Provision a host from a CSV file"

const deleteHostExamples = `#Delete a host using it's host Resource ID
orch-cli delete host host-1234abcd  --project itep`

const deauthorizeHostExamples = `#Deauthorize the host and it's access to Edge Orchestrator using the host Resource ID
orch-cli deauthorize host host-1234abcd  --project itep`

var hostHeader = fmt.Sprintf("\n%s\t%s\t%s\t%s\t%s\t%s\t%s", "Resource ID", "Name", "Host Status", "Serial Number", "Operating System", "Site ID", "Workload")
var hostHeaderGet = "\nDetailed Host Information\n"
var filename = "test.csv"

const kVSize = 2

type ResponseCache struct {
	OSProfileCache map[string]infra.OperatingSystemResource
	SiteCache      map[string]infra.Site
	LACache        map[string]infra.LocalAccount
	HostCache      map[string]infra.Host
}

type MetadataItem = struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
func printHosts(writer io.Writer, hosts *[]infra.Host, verbose bool) {

	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "Resource ID", "Name", "Host Status",
			"Serial Number", "Operating System", "Site ID", "Site Name", "Workload", "Host ID", "UUID", "Processor", "Available Update", "Trusted Compute")
	}
	for _, h := range *hosts {
		//TODO clean this up
		os, workload, site, siteName := "Not provisioned", "Not provisioned", "Not provisioned", "Not provisioned"
		host := "Not connected"

		if h.Instance != nil {
			if h.Instance.CurrentOs != nil {
				os = toJSON(h.Instance.CurrentOs.Name)
			}
			if h.Instance.WorkloadMembers != nil {
				workload = toJSON(h.Instance.WorkloadMembers)
			}
		}
		if h.SiteId != nil {
			site = toJSON(h.SiteId)
		}

		if h.Site != nil && h.Site.Name != nil {
			siteName = toJSON(h.Site.Name)
		}

		if *h.HostStatus != "" {
			host = *h.HostStatus
		}
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%v\t%v\t%v\t%v\n", *h.ResourceId, h.Name, host, *h.SerialNumber, os, site, workload)
		} else {
			avupdt := "No update"
			tcomp := "Not compatible"

			//TODO
			//if h.CurrentOs != h.desiredOS avupdt is available
			//if tcomp is set then reflect

			fmt.Fprintf(writer, "%s\t%s\t%s\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", *h.ResourceId, h.Name, host, *h.SerialNumber,
				os, site, siteName, workload, h.Name, h.Uuid, *h.CpuModel, avupdt, tcomp)
		}
	}
}

func printHost(writer io.Writer, host *infra.Host) {

	updatestatus := ""
	hoststatus := "Not connected"
	currentOS := ""
	osprofile := ""

	//TODO Build out the host information
	if host != nil && host.Instance != nil && host.Instance.UpdateStatus != nil {
		updatestatus = toJSON(host.Instance.UpdateStatus)
	}

	if host != nil && host.Instance != nil && host.Instance.CurrentOs.Name != nil {
		currentOS = toJSON(host.Instance.CurrentOs.Name)
	}

	if host != nil && host.Instance != nil && host.Instance.Os.Name != nil {
		osprofile = toJSON(host.Instance.Os.Name)
	}

	if *host.HostStatus != "" {
		hoststatus = *host.HostStatus
	}

	_, _ = fmt.Fprintf(writer, "Host Info: \n\n")
	_, _ = fmt.Fprintf(writer, "-\tHost Resurce ID:\t %s\n", *host.ResourceId)
	_, _ = fmt.Fprintf(writer, "-\tName:\t %s\n", host.Name)
	_, _ = fmt.Fprintf(writer, "-\tOS Profile:\t %v\n\n", osprofile)

	_, _ = fmt.Fprintf(writer, "Status details: \n\n")
	_, _ = fmt.Fprintf(writer, "-\tHost Status:\t %s\n", hoststatus)
	_, _ = fmt.Fprintf(writer, "-\tUpdate Status:\t %s\n\n", updatestatus)

	_, _ = fmt.Fprintf(writer, "Specification: \n\n")
	_, _ = fmt.Fprintf(writer, "-\tSerial Number:\t %s\n", *host.SerialNumber)
	_, _ = fmt.Fprintf(writer, "-\tUUID:\t %s\n", host.Uuid)
	_, _ = fmt.Fprintf(writer, "-\tOS:\t %v\n", currentOS)
	_, _ = fmt.Fprintf(writer, "-\tBIOS Vendor:\t %v\n", *host.BiosVendor)
	_, _ = fmt.Fprintf(writer, "-\tProduct Name:\t %v\n\n", *host.ProductName)

	_, _ = fmt.Fprintf(writer, "CPU Info: \n\n")
	_, _ = fmt.Fprintf(writer, "-\tCPU Model:\t %v\n", *host.CpuModel)
	_, _ = fmt.Fprintf(writer, "-\tCPU Cores:\t %v\n", *host.CpuCores)
	_, _ = fmt.Fprintf(writer, "-\tCPU Architecture:\t %v\n", *host.CpuArchitecture)
	_, _ = fmt.Fprintf(writer, "-\tCPU Threads:\t %v\n", *host.CpuThreads)
	_, _ = fmt.Fprintf(writer, "-\tCPU Sockets:\t %v\n\n", *host.CpuSockets)

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
func doRegister(ctx context.Context, hClient *infra.ClientWithResponses, projectName string,
	rIn types.HostRecord, respCache ResponseCache, erringRecords *[]types.HostRecord) {

	// get the required fields from the record
	sNo := rIn.Serial
	uuid := rIn.UUID
	// predefine other fields
	hostName := ""
	hostID := ""
	autonboard := true

	rOut, err := sanitizeProvisioningFields(ctx, hClient, projectName, rIn, respCache, erringRecords)
	if err != nil {
		return
	}

	hostID, err = registerHost(ctx, hClient, respCache, projectName, hostName, sNo, uuid, autonboard)
	if err != nil {
		rIn.Error = err.Error()
		*erringRecords = append(*erringRecords, rIn)
		return
	}

	err = createInstance(ctx, hClient, respCache, projectName, hostID, rOut, rIn)
	if err != nil {
		rIn.Error = err.Error()
		*erringRecords = append(*erringRecords, rIn)
		return
	}

	err = allocateHostToSiteAndAddMetadata(ctx, hClient, respCache, projectName, hostID, rOut)
	if err != nil {
		rIn.Error = err.Error()
		*erringRecords = append(*erringRecords, rIn)
		return
	}

	// Print host_id from response if successful
	fmt.Printf("✔ Host Serial number : %s  UUID : %s registered. Name : %s\n", sNo, uuid, hostID)
}

// Decodes the provided metadata from input string
func decodeMetadata(metadata string) (*infra.Metadata, error) {
	metadataList := make(infra.Metadata, 0)
	if metadata == "" {
		return &metadataList, nil
	}
	metadataPairs := strings.Split(metadata, "&")
	for _, pair := range metadataPairs {
		kv := strings.Split(pair, "=")
		if len(kv) != kVSize {
			return &metadataList, e.NewCustomError(e.ErrInvalidMetadata)
		}
		mItem := MetadataItem{
			Key:   kv[0],
			Value: kv[1],
		}
		metadataList = append(metadataList, mItem)
	}
	return &metadataList, nil
}

// Sanitize filelds, convert named resources to resource IDs
func sanitizeProvisioningFields(ctx context.Context, hClient *infra.ClientWithResponses, projectName string, record types.HostRecord,
	respCache ResponseCache, erringRecords *[]types.HostRecord) (*types.HostRecord, error) {

	isSecure := record.Secure

	osProfileID, err := resolveOSProfile(ctx, hClient, projectName, record.OSProfile, record, respCache, erringRecords)
	if err != nil {
		return nil, err
	}

	if valErr := validateSecurityFeature(record.OSProfile, isSecure, record, respCache, erringRecords); valErr != nil {
		return nil, valErr
	}

	siteID, err := resolveSite(ctx, hClient, projectName, record.Site, record, respCache, erringRecords)
	if err != nil {
		return nil, err
	}

	laID, err := resolveRemoteUser(ctx, hClient, projectName, record.RemoteUser, record, respCache, erringRecords)
	if err != nil {
		return nil, err
	}

	//TODO implement AMT check - will there be a check if a host is capable of AMT
	// valErr = validateAMT(ctx, hClient, projectName, record.AMTEnable, record, respCache, erringRecords)
	// if err != nil {
	// 	return nil, valErr
	// }

	//TODO implement cloud Init check
	// cloudInitID, err := resolveCloudInit(ctx, hClient, projectName, record.CloudInitMeta, record, respCache, erringRecords)
	// if err != nil {
	// 	return nil, err
	// }

	//TODO implement check for K8s Cluster template
	// K8sTmplID, err := resolveCloudInit(ctx, hClient, projectName, record., record.K8sClusterTemplate, respCache, erringRecords)
	// if err != nil {
	// 	return nil, err
	// }

	return &types.HostRecord{
		OSProfile:  osProfileID,
		RemoteUser: laID,
		Site:       siteID,
		Secure:     isSecure,
		UUID:       record.UUID,
		Serial:     record.Serial,
		Metadata:   record.Metadata,
		//AMTEnable:  isAMT,
		//CloudInitMeta: cloudInitID,
		//K8sClusterTemplate: K8sTmplID,
	}, nil
}

// Ensures that OS profile exists
func resolveOSProfile(ctx context.Context, hClient *infra.ClientWithResponses, projectName string, recordOSProfile string,
	record types.HostRecord, respCache ResponseCache, erringRecords *[]types.HostRecord,
) (string, error) {

	if recordOSProfile == "" {
		record.Error = e.NewCustomError(e.ErrInvalidOSProfile).Error()
		*erringRecords = append(*erringRecords, record)
		return "", e.NewCustomError(e.ErrInvalidOSProfile)
	}

	if osResource, ok := respCache.OSProfileCache[recordOSProfile]; ok {
		return *osResource.ResourceId, nil
	}

	ospfilter := fmt.Sprintf("profileName='%s' OR resourceId='%s'", recordOSProfile, recordOSProfile)
	resp, err := hClient.GetV1ProjectsProjectNameComputeOsWithResponse(ctx, projectName,
		&infra.GetV1ProjectsProjectNameComputeOsParams{
			Filter: &ospfilter,
		}, auth.AddAuthHeader)
	if err != nil {
		record.Error = err.Error()
		*erringRecords = append(*erringRecords, record)
		return "", err
	}
	if resp.JSON200.OperatingSystemResources != nil {
		osResources := *resp.JSON200.OperatingSystemResources
		if len(osResources) > 0 {
			respCache.OSProfileCache[recordOSProfile] = osResources[len(osResources)-1]
			return *osResources[len(osResources)-1].ResourceId, nil
		}
	}
	record.Error = "OS Profile not found"
	*erringRecords = append(*erringRecords, record)
	return "", errors.New(record.Error)
}

// Checks input ecurity feature vs what is capabale by host
func validateSecurityFeature(osProfileID string, isSecure types.RecordSecure,
	record types.HostRecord, respCache ResponseCache, erringRecords *[]types.HostRecord,
) error {
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
func resolveSite(ctx context.Context, hClient *infra.ClientWithResponses, projectName string, recordSite string,
	record types.HostRecord, respCache ResponseCache, erringRecords *[]types.HostRecord,
) (string, error) {

	if record.Site == "" {
		record.Error = e.NewCustomError(e.ErrInvalidSite).Error()
		*erringRecords = append(*erringRecords, record)
		return "", e.NewCustomError(e.ErrInvalidSite)
	}

	if siteResource, ok := respCache.SiteCache[record.Site]; ok {
		return *siteResource.ResourceId, nil
	}

	resp, err := hClient.GetV1ProjectsProjectNameRegionsRegionIDSitesSiteIDWithResponse(ctx, projectName, "regionID", recordSite, auth.AddAuthHeader)
	if err != nil {
		record.Error = err.Error()
		*erringRecords = append(*erringRecords, record)
		return "", err
	}

	err = checkResponse(resp.HTTPResponse, "Error Site not found")
	if err != nil {
		record.Error = err.Error()
		*erringRecords = append(*erringRecords, record)
		return "", err
	}

	respCache.SiteCache[recordSite] = *resp.JSON200
	return *resp.JSON200.ResourceId, nil
}

// Cecks if remote user is valid and exists
func resolveRemoteUser(ctx context.Context, hClient *infra.ClientWithResponses, projectName string, recordRemoteUser string,
	record types.HostRecord, respCache ResponseCache, erringRecords *[]types.HostRecord,
) (string, error) {

	if recordRemoteUser == "" {
		return "", nil
	}

	if lAResource, ok := respCache.LACache[recordRemoteUser]; ok {
		return *lAResource.ResourceId, nil
	}

	lafilter := fmt.Sprintf("username='%s' OR resourceId='%s'", recordRemoteUser, recordRemoteUser)
	resp, err := hClient.GetV1ProjectsProjectNameLocalAccountsWithResponse(ctx, projectName,
		&infra.GetV1ProjectsProjectNameLocalAccountsParams{
			Filter: &lafilter,
		}, auth.AddAuthHeader)
	if err != nil {
		record.Error = err.Error()
		*erringRecords = append(*erringRecords, record)
		return "", err
	}
	if resp.JSON200.LocalAccounts != nil {
		localAccounts := *resp.JSON200.LocalAccounts
		if len(localAccounts) > 0 {
			respCache.LACache[recordRemoteUser] = localAccounts[len(localAccounts)-1]
			return *localAccounts[len(localAccounts)-1].ResourceId, nil
		}
	}
	record.Error = "Remote User not found"
	*erringRecords = append(*erringRecords, record)
	return "", errors.New(record.Error)
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
		RunE:    runListHostCommand,
	}

	// Local persistent flags
	cmd.PersistentFlags().StringP("filter", "f", viper.GetString("filter"), "Optional filter provided as part of host list command\nUsage:\n\tCustom filter: --filter \"<custom filter>\" ie. --filter \"osType=OS_TYPE_IMMUTABLE\" see https://google.aip.dev/160 and API spec. \n\tPredefined filters: --filter provisioned/onboarded/registered/nor connected/deauthorized")
	cmd.PersistentFlags().StringP("site", "s", viper.GetString("site"), "Optional filter provided as part of host list to filter hosts by site")
	cmd.PersistentFlags().StringP("region", "r", viper.GetString("region"), "Optional filter provided as part of host list to filter hosts by region")
	return cmd
}

func getGetHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host <resourceID> [flags]",
		Short:   "Gets a host",
		Example: getHostExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runGetHostCommand,
	}
	return cmd
}

func getCreateHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host --import-from-csv]",
		Short:   "Provisions a host or hosts",
		Example: createHostExamples,
		RunE:    runCreateHostCommand,
	}

	// Local persistent flags
	cmd.PersistentFlags().StringP("import-from-csv", "i", viper.GetString("import-from-csv"), "CSV file containing information about to be provisioned hosts")
	cmd.PersistentFlags().BoolP("dry-run", "d", viper.GetBool("dry-run"), "Verify the validity of input CSV file")
	cmd.PersistentFlags().BoolP("generate-csv", "g", viper.GetBool("generate-csv"), "Generates a template CSV file for host import")
	return cmd
}

func getDeleteHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host <resourceID> [flags]",
		Short:   "Deletes a host",
		Example: deleteHostExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runDeleteHostCommand,
	}
	return cmd
}

func getDeauthorizeHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host <resourceID> [flags]",
		Short:   "Deauthorizes a host",
		Example: deauthorizeHostExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runDeauthorizeHostCommand,
	}
	return cmd
}

// Lists all Hosts - retrieves all hosts and displays selected information in tabular format
func runListHostCommand(cmd *cobra.Command, _ []string) error {

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

	ctx, hostClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	//If all host for a given region are quereied, sites need to be found first
	if siteFlag == "" && regFlag != "" {

		regFilter := fmt.Sprintf("region.resource_id='%s' OR region.parent_region.resource_id='%s' OR region.parent_region.parent_region.resource_id='%s' OR region.parent_region.parent_region.parent_region.resource_id='%s'", regFlag, regFlag, regFlag, regFlag)

		cresp, err := hostClient.GetV1ProjectsProjectNameRegionsRegionIDSitesWithResponse(ctx, projectName, *region,
			&infra.GetV1ProjectsProjectNameRegionsRegionIDSitesParams{
				Filter: &regFilter,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}

		//create site filter
		siteFilter := ""
		if *cresp.JSON200.TotalElements != 0 {
			for i, s := range *cresp.JSON200.Sites {
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

	resp, err := hostClient.GetV1ProjectsProjectNameComputeHostsWithResponse(ctx, projectName,
		&infra.GetV1ProjectsProjectNameComputeHostsParams{
			Filter: filter,
			SiteID: site,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		hostHeader, "error getting Hosts"); !proceed {
		return err
	}

	printHosts(writer, resp.JSON200.Hosts, verbose)

	return writer.Flush()
}

// Gets specific Host - retrieves a host using resource ID and displays detailed information
func runGetHostCommand(cmd *cobra.Command, args []string) error {

	hostID := args[0]
	writer, verbose := getOutputContext(cmd)
	ctx, hostClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := hostClient.GetV1ProjectsProjectNameComputeHostsHostIDWithResponse(ctx, projectName,
		hostID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		hostHeaderGet, "error getting Host"); !proceed {
		return err
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

	generate, _ := cmd.Flags().GetBool("generate-csv")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	csvFilePath, _ := cmd.Flags().GetString("import-from-csv")

	if generate && (dryRun || csvFilePath != "") {
		return fmt.Errorf("cannot use --generate-csv flag with --dry-run and/or --import-from-csv")
	}

	if generate {
		err = generateCSV(fmt.Sprintf("%s/%s", currentPath, filename))
		if err != nil {
			return err
		}
		return nil
	}

	if csvFilePath == "" {
		return fmt.Errorf("--import-from-csv <path/to/file.csv> is required")
	}

	err = verifyCSVInput(csvFilePath)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Println("--dry-run flag provided, validating input, hosts will not be imported")
		_, err := validator.CheckCSV(csvFilePath)
		if err != nil {
			return err
		}
		fmt.Println("CSV validation successful")
		return nil
	}

	validated, err := validator.CheckCSV(csvFilePath)
	if err != nil {
		return err
	}

	respCache := ResponseCache{
		OSProfileCache: make(map[string]infra.OperatingSystemResource),
		SiteCache:      make(map[string]infra.Site),
		LACache:        make(map[string]infra.LocalAccount),
		HostCache:      make(map[string]infra.Host),
	}

	ctx, hostClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}
	erringRecords := []types.HostRecord{}

	for _, record := range validated {
		doRegister(ctx, hostClient, projectName, record, respCache, &erringRecords)
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
	ctx, hostClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	//Get instance associated with host
	resp, err := hostClient.GetV1ProjectsProjectNameComputeHostsHostIDWithResponse(ctx, projectName,
		hostID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	err = checkResponse(resp.HTTPResponse, "error while getting host to delete")
	if err != nil {
		return err
	}

	//If instance associatied delete it
	if resp.JSON200.Instance != nil {
		instanceID := resp.JSON200.Instance.ResourceId

		iresp, err := hostClient.DeleteV1ProjectsProjectNameComputeInstancesInstanceIDWithResponse(ctx, projectName,
			*instanceID, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}

		err = checkResponse(iresp.HTTPResponse, "error while deleting instance during host deletion")
		if err != nil {
			return err
		}
	}

	//Delete host
	dresp, err := hostClient.DeleteV1ProjectsProjectNameComputeHostsHostIDWithResponse(ctx, projectName,
		hostID, infra.DeleteV1ProjectsProjectNameComputeHostsHostIDJSONRequestBody{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(dresp.HTTPResponse, "error while deleting host")
}

// Deauthorizes specific Host - finds a host using resource ID and invalidates it
func runDeauthorizeHostCommand(cmd *cobra.Command, args []string) error {
	hostID := args[0]
	ctx, hostClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := hostClient.PutV1ProjectsProjectNameComputeHostsHostIDInvalidateWithResponse(ctx, projectName,
		hostID, infra.PutV1ProjectsProjectNameComputeHostsHostIDInvalidateJSONRequestBody{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, "error while deleting host")
}

// Function containing the logic to register the host and retrieve the host ID
func registerHost(ctx context.Context, hClient *infra.ClientWithResponses, respCache ResponseCache, projectName, hostName, sNo, uuid string, autonboard bool) (string, error) {
	//convert uuid
	var uuidParsed u.UUID
	if uuid != "" {
		uuidParsed = u.MustParse(uuid)
	}

	// Register host
	resp, err := hClient.PostV1ProjectsProjectNameComputeHostsRegisterWithResponse(ctx, projectName,
		infra.PostV1ProjectsProjectNameComputeHostsRegisterJSONRequestBody{
			Name:         &hostName,
			SerialNumber: &sNo,
			Uuid:         &uuidParsed,
			AutoOnboard:  &autonboard,
		}, auth.AddAuthHeader)
	if err != nil {
		return "", processError(err)
	}
	//Check that valid response was received
	err = checkResponse(resp.HTTPResponse, "error while registering host")
	if err != nil {
		//if host already registered
		if resp.HTTPResponse.StatusCode == http.StatusPreconditionFailed {
			//form a filter
			hFilter := fmt.Sprintf("serialNumber='%s' AND uuid='%s'", sNo, uuid)

			//get all the hosts matching the filter
			gresp, err := hClient.GetV1ProjectsProjectNameComputeHostsWithResponse(ctx, projectName,
				&infra.GetV1ProjectsProjectNameComputeHostsParams{
					Filter: &hFilter,
				}, auth.AddAuthHeader)
			if err != nil {
				processError(err)
			}

			err = checkResponse(gresp.HTTPResponse, "error while getting host which failed registration")
			if err != nil {
				return "", err
			}

			if *gresp.JSON200.TotalElements != 1 {
				err = e.NewCustomError(e.ErrHostDetailMismatch)
				return "", err
			} else if (*gresp.JSON200.Hosts)[0].Instance != nil {
				err = e.NewCustomError(e.ErrAlreadyRegistered)
				return "", err
			} else {
				respCache.HostCache[*(*gresp.JSON200.Hosts)[0].ResourceId] = (*gresp.JSON200.Hosts)[0]
				return *(*gresp.JSON200.Hosts)[0].ResourceId, nil
			}

		} else {
			return "", err
		}
	} else {
		//Cache host and save host ID
		if resp.JSON201 != nil && resp.JSON201.ResourceId != nil {
			respCache.HostCache[*resp.JSON201.ResourceId] = *resp.JSON201
			return *resp.JSON201.ResourceId, nil
		} else {
			return "", errors.New("host not found")
		}
	}
}

// If a valid OE Profile exists creates an instance linking to host resource
func createInstance(ctx context.Context, hClient *infra.ClientWithResponses, respCache ResponseCache,
	projectName, hostID string, rOut *types.HostRecord, rIn types.HostRecord) error {

	// Validate OS profile
	if valErr := validateOSProfile(rOut.OSProfile); valErr != nil {
		return valErr
	}
	// Create instance if osProfileID is available
	// Need not notify user of instance ID. Unnecessary detail for user.
	kind := infra.INSTANCEKINDUNSPECIFIED
	osResource, ok := respCache.OSProfileCache[rIn.OSProfile]
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

	iresp, err := hClient.PostV1ProjectsProjectNameComputeInstancesWithResponse(ctx, projectName,
		infra.PostV1ProjectsProjectNameComputeInstancesJSONRequestBody{
			HostID:          &hostID,
			OsID:            &rOut.OSProfile,
			LocalAccountID:  locAcc,
			SecurityFeature: &secFeat,
			Kind:            &kind,
		}, auth.AddAuthHeader)
	if err != nil {
		err := processError(err)
		return err
	}

	err = checkResponse(iresp.HTTPResponse, "error while creating instance\n\n")
	if err != nil {
		return err
	}

	return nil
}

// Decode input metadata and add to host, allocate host to site
func allocateHostToSiteAndAddMetadata(ctx context.Context, hClient *infra.ClientWithResponses, respCache ResponseCache,
	projectName, hostID string, rOut *types.HostRecord) error {

	// Update host with Site and metadata
	var metadata *infra.Metadata
	var err error
	if rOut.Metadata != "" {
		metadata, err = decodeMetadata(rOut.Metadata)
		if err != nil {
			return err
		}
	}

	sresp, err := hClient.PatchV1ProjectsProjectNameComputeHostsHostIDWithResponse(ctx, projectName, hostID,
		infra.PatchV1ProjectsProjectNameComputeHostsHostIDJSONRequestBody{
			Name:     hostID,
			Metadata: metadata,
			SiteId:   &rOut.Site,
		}, auth.AddAuthHeader)
	if err != nil {
		err := processError(err)
		return err
	}

	err = checkResponse(sresp.HTTPResponse, "error while linking site and metadata\n\n")
	if err != nil {
		return err
	}

	return nil
}
