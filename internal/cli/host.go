// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/google/uuid"
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

const deleteHostExamples = `#Delete a host using it's host Resource ID
orch-cli delete host host-1234abcd  --project itep`

const deauthorizeHostExamples = `#Deauthorize the host and it's access to Edge Orchestrator using the host Resource ID
orch-cli deauthorize host host-1234abcd  --project itep`

var hostHeaderGet = "\nDetailed Host Information\n"

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
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "Resource ID", "Name", "Host Status",
			"Serial Number", "Operating System", "Site", "Workload", "Host ID", "UUID", "Processor", "Available Update", "Trusted Compute")
	}
	for _, h := range *hosts {
		//TODO clean this up
		os, workload, site := "Not provisioned", "Not provisioned", "Not provisioned"
		host := "Not connected"

		if h.Instance != nil {
			if h.Instance.CurrentOs != nil && h.Instance.CurrentOs.Name != nil {
				os = toJSON(h.Instance.CurrentOs.Name)
			}
			if h.Instance.WorkloadMembers != nil {
				workload = toJSON(h.Instance.WorkloadMembers)
			}
		}
		if h.SiteId != nil {
			site = toJSON(h.SiteId)
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

			fmt.Fprintf(writer, "%s\t%s\t%s\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", *h.ResourceId, h.Name, host, *h.SerialNumber,
				os, site, workload, h.Name, h.Uuid, *h.CpuModel, avupdt, tcomp)
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

func getRegisterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "register",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Register host",
		PersistentPreRunE: auth.CheckAuth,
	}

	cmd.AddCommand(
		getRegisterHostCommand(),
	)
	return cmd
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

func getRegisterHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host <name> [flags, --serial and/or --uuid must be provided]",
		Short:   "Registers a host",
		Example: registerHostExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runRegisterHostCommand,
	}

	// Local persistent flags
	cmd.PersistentFlags().StringP("uuid", "u", viper.GetString("uuid"), "Host UUID to be provided as registration argument")
	cmd.PersistentFlags().StringP("serial", "s", viper.GetString("serial"), "Host Serial Number to be provided as registration argument")

	return cmd
}

func getDeleteHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host <resourceID> [flags]",
		Short:   "Deletes a host and associated instance",
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

	pageSize := 20
	hosts := make([]infra.Host, 0)
	for offset := 0; ; offset += pageSize {
		resp, err := hostClient.GetV1ProjectsProjectNameComputeHostsWithResponse(ctx, projectName,
			&infra.GetV1ProjectsProjectNameComputeHostsParams{
				Filter:   filter,
				SiteID:   site,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}

		if err := checkResponse(resp.HTTPResponse, "error while retrieving hosts"); err != nil {
			return err
		}
		hosts = append(hosts, *resp.JSON200.Hosts...)
		if !*resp.JSON200.HasNext {
			break // No more hosts to process
		}
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

// Registers specific Host - registers sprcific host using SN and/or UUID
func runRegisterHostCommand(cmd *cobra.Command, args []string) error {

	//TODO add autoonboarding and autoprovision??
	hostname := args[0]

	serial, _ := cmd.Flags().GetString("serial")
	uuidString, _ := cmd.Flags().GetString("uuid")

	if serial == "" && uuidString == "" {
		return errors.New("at least one of the flags 'serial' or 'uuid' must be provided")
	}

	ctx, hostClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	var uuidParsed *uuid.UUID
	if uuidString != "" {
		parsedUUID, err := uuid.Parse(uuidString)
		if err != nil {
			return err
		}
		uuidParsed = &parsedUUID
	}

	resp, err := hostClient.PostV1ProjectsProjectNameComputeHostsRegisterWithResponse(ctx, projectName,
		infra.PostV1ProjectsProjectNameComputeHostsRegisterJSONRequestBody{
			Name:         &hostname,
			SerialNumber: &serial,
			Uuid:         uuidParsed,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, "error while registering host")
}

// Deletes specific Host - finds a host using resource ID and deletes it
func runDeleteHostCommand(cmd *cobra.Command, args []string) error {
	hostID := args[0]
	_, verbose := getOutputContext(cmd)
	ctx, hostClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	// retrieve the host (to check if it has an instance associated with it)
	resp1, err := hostClient.GetV1ProjectsProjectNameComputeHostsHostIDWithResponse(ctx, projectName, hostID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp1.HTTPResponse, "error while retrieving host"); err != nil {
		return err
	}
	host := *resp1.JSON200

	// delete the instance if it exists
	instanceID := host.Instance.InstanceID
	if instanceID != nil && *instanceID != "" {
		resp2, err := hostClient.DeleteV1ProjectsProjectNameComputeInstancesInstanceIDWithResponse(ctx, projectName, *instanceID, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(resp2.HTTPResponse, "error while deleting instance"); err != nil {
			return err
		}
		fmt.Printf("Instance %v associated with host deleted successfully\n", *instanceID)
	} else if verbose {
		fmt.Printf("Host %s does not have an instance associated with it, deleting host only\n", hostID)
	}

	// delete the host
	resp3, err := hostClient.DeleteV1ProjectsProjectNameComputeHostsHostIDWithResponse(ctx, projectName,
		hostID, infra.DeleteV1ProjectsProjectNameComputeHostsHostIDJSONRequestBody{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp3.HTTPResponse, "error while deleting host"); err != nil {
		return err
	}
	fmt.Printf("Host %s deleted successfully\n", hostID)
	return nil
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
