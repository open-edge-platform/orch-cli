// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//TODO handle auto-onboard flag
//TODO handle auto-provision flag

var hostHeader = fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s", "Name", "Host Status", "Serial Number", "Operating System", "Site", "Workload")
var hostHeaderGet = fmt.Sprintf("%s\t%s", "Host Field", "Value")

// Prints Host list in tabular format
func printHosts(writer io.Writer, hosts *[]infra.Host, verbose bool) {
	for _, h := range *hosts {
		if !verbose {
			os, workload := "not set", "not set"
			host := "unknown"
			if h.Instance != nil {
				os = string(toJSON(h.Instance.CurrentOs))
				workload = string(toJSON(h.Instance.WorkloadMembers))
			}
			if *h.HostStatus != "" {
				host = *h.HostStatus
			}
			fmt.Fprintf(writer, "%s\t%s\t%v\t%v\t%v\t%v\n", h.Name, host, *h.SerialNumber, os, h.Site, workload)
		} else {
			// TODO: expand verbose list - perhaps chande to wider tabular with -o wide
			_, _ = fmt.Fprintf(writer, "Name:\t %s\n", h.Name)
			if *h.HostStatus == "" {
				_, _ = fmt.Fprintf(writer, "Host Status:\t unknown\n")
			} else {
				_, _ = fmt.Fprintf(writer, "Host Status:\t %s\n", *h.HostStatus)
			}
			_, _ = fmt.Fprintf(writer, "Serial number:\t %v\n", *h.SerialNumber)
			if h.Instance == nil {
				_, _ = fmt.Fprintf(writer, "Operating System:\t not set\n")
			} else {
				_, _ = fmt.Fprintf(writer, "Operating System:\t %v\n", h.Instance.CurrentOs)
			}
			if h.Site == nil {
				_, _ = fmt.Fprintf(writer, "Site:\t not set\n")
			} else {
				_, _ = fmt.Fprintf(writer, "Site:\t %v\n", *h.Site)
			}
			if h.Instance == nil {
				_, _ = fmt.Fprintf(writer, "Workload:\t not set\n\n")
			} else {
				_, _ = fmt.Fprintf(writer, "Workload:\t %v\n\n", h.Instance.WorkloadMembers)
			}
		}
	}
}

func printHost(writer io.Writer, host *infra.Host) {

	_, _ = fmt.Fprintf(writer, "Name:\t %s\n\n", host.Name)

	_, _ = fmt.Fprintf(writer, "Status details: \n\n")
	_, _ = fmt.Fprintf(writer, "Host Status:\t %s\n", *host.HostStatus)
	//_, _ = fmt.Fprintf(writer, "\tUpdate Status:\t %s\n", *host.Instance.UpdateStatus)

	// _, _ = fmt.Fprintf(writer, "Specification: \n\n")
	// _, _ = fmt.Fprintf(writer, "\tSerial Number:\t %s\n", *host.SerialNumber)
	// _, _ = fmt.Fprintf(writer, "\tUUID:\t %s\n", *host.Uuid)
	// _, _ = fmt.Fprintf(writer, "\tOS:\t %v\n", host.Instance.CurrentOs)
	// _, _ = fmt.Fprintf(writer, "\tBIOS Vendor:\t %v\n", host.BiosVendor)
	// _, _ = fmt.Fprintf(writer, "\tProduct Name:\t %v\n", host.ProductName)

	// _, _ = fmt.Fprintf(writer, "CPU Info: \n\n")
	// _, _ = fmt.Fprintf(writer, "\tCPU Model:\t %v\n", host.CpuModel)
	// _, _ = fmt.Fprintf(writer, "\tCPU Cores:\t %v\n", host.CpuCores)
	// _, _ = fmt.Fprintf(writer, "\tCPU Architecture:\t %v\n", host.CpuArchitecture)
	// _, _ = fmt.Fprintf(writer, "\tCPU Threads:\t %v\n", host.CpuThreads)
	// _, _ = fmt.Fprintf(writer, "\tCPU Sockets:\t %v\n", host.CpuSockets)

	// //TODO enhance GPU display
	// _, _ = fmt.Fprintf(writer, "GPU Info: \n\n")
	// _, _ = fmt.Fprintf(writer, "\tGPU Model:\t %v\n", host.HostGpus)

	// //TODO enhance USB display
	// _, _ = fmt.Fprintf(writer, "I/O Devices Info: \n\n")
	// _, _ = fmt.Fprintf(writer, "\tUSB Model:\t %v\n", host.HostUsbs)

	// //TODO enhance labels
	// _, _ = fmt.Fprintf(writer, "Host Labels: \n\n")

	// //TODO enhance profile name
	// _, _ = fmt.Fprintf(writer, "OS Profile:\t %v\n", host.Instance.Os.Name)

}

// Filters list of profiles to find one with specific name
func filterHostsByName(hosts *[]infra.Host, name string) (*infra.Host, error) {
	for _, h := range *hosts {
		if h.Name == name {
			return &h, nil
		}
	}
	return nil, errors.New("no os host matches the given name")
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
		getOnboardHostCommand(),   //TODO is this in worng spot - should I make getOnboardCommand()
		getProvisionHostCommand(), //TODO is this in worng spot - should I make getOnboardCommand()
		getImportHostCommand(),    //TODO is this in worng spot - should I make getOnboardCommand()
	)
	return cmd
}

func getListHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host <name> [flags]",
		Short: "List hosts",
		RunE:  runListHostCommand,
	}
	return cmd
}

func getGetHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host <name> [flags]",
		Short: "Get host",
		Args:  cobra.ExactArgs(1),
		RunE:  runGetHostCommand,
	}
	return cmd
}

func getRegisterHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host <name> [flags]",
		Short: "Register a host",
		Args:  cobra.ExactArgs(1),
		RunE:  runRegisterHostCommand,
	}

	// Local persistent flags
	cmd.PersistentFlags().StringP("uuid", "u", viper.GetString("uuid"), "Host UUID to be provided as registration argument")
	cmd.PersistentFlags().StringP("serial", "s", viper.GetString("serial"), "Host Serial Number to be provided as registration argument")

	return cmd
}

func getOnboardHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host <name> [flags]",
		Short: "Register a host",
		Args:  cobra.ExactArgs(1),
		RunE:  runOnboardHostCommand,
	}
	return cmd
}

func getProvisionHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host <name> [flags]",
		Short: "Register a host",
		Args:  cobra.ExactArgs(1),
		RunE:  runProvisionHostCommand,
	}
	return cmd
}

func getImportHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host <name> [flags]",
		Short: "Register a host",
		Args:  cobra.ExactArgs(1),
		RunE:  runImportHostCommand,
	}
	return cmd
}

// Lists all Hosts - retrieves all hosts and displays selected information in tabular format
func runListHostCommand(cmd *cobra.Command, _ []string) error {

	//TODO: List by flag
	writer, verbose := getOutputContext(cmd)

	ctx, hostClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := hostClient.GetV1ProjectsProjectNameComputeHostsWithResponse(ctx, projectName,
		&infra.GetV1ProjectsProjectNameComputeHostsParams{}, auth.AddAuthHeader)
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

func runGetHostCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, hostClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := hostClient.GetV1ProjectsProjectNameComputeHostsWithResponse(ctx, projectName,
		&infra.GetV1ProjectsProjectNameComputeHostsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		hostHeaderGet, "error getting Host"); !proceed {
		return err
	}

	name := args[0]
	host, err := filterHostsByName(resp.JSON200.Hosts, name)
	if err != nil {
		return err
	}

	printHost(writer, host)
	return writer.Flush()
}

func runRegisterHostCommand(cmd *cobra.Command, args []string) error {

	hostname := args[0]

	//TODO ensure that either serial or uuid is not set to nil
	serial, _ := cmd.Flags().GetString("serial")
	uuidString, _ := cmd.Flags().GetString("uuid")

	//TODO add autoonboarding and autoprovision
	ctx, hostClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	var uuidParsed *uuid.UUID
	if uuidString != "" {
		parsedUUID, err := uuid.Parse(uuidString)
		if err != nil {
			fmt.Println("Invalid UUID format:", err)
			return err
		}
		uuidParsed = &parsedUUID
	}

	//TODO: Ensure that the host does not alrady exist
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

func runOnboardHostCommand(cmd *cobra.Command, _ []string) error {
	return nil
}

func runProvisionHostCommand(cmd *cobra.Command, _ []string) error {
	return nil
}

func runImportHostCommand(cmd *cobra.Command, _ []string) error {
	return nil
}
