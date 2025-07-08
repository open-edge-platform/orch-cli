// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/open-edge-platform/cli/pkg/rest/rps"
	"github.com/spf13/cobra"
)

const listAmtProfileExamples = `
// # List all sites
// orch-cli list site --project some-project

// # List all sites within specific parent region ID
// orch-cli list site --project some-project --region region-aaaa1111"
`

const getAmtProfileExamples = `
// # Get specific site information
// orch-cli get site site-aaaa1111 --project some-project
`

const createAmtProfileExamples = `
// # Create specific site

// # Create a site in a region (default longitude and latitude set to 0)
// orch-cli create site name --project some-project --region region-bbbb1111

// # Create a site in a region (default longitude and latitude set to 0)
// orch-cli create site name --project some-project --region region-bbbb1111 --longitude 5 --latitude 5
`
const deleteAmtProfileExamples = `
// # Delete specific site
// orch-cli delete site region-aaaa1111 --project some-project
`

func getListAmtProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "amtprofile [flags]",
		Short:   "List all amptprofiles",
		Example: listAmtProfileExamples,
		RunE:    runListAmtProfileCommand,
	}
	//cmd.PersistentFlags().StringP("region", "r", viper.GetString("region"), "Optional filter provided as part of site list to filter sites by parent region")
	return cmd
}

func getGetAmtProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "amtprofile <resourceid> [flags]",
		Short:   "Get an AMT profile",
		Example: getAmtProfileExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runGetAmtProfileCommand,
	}
	return cmd
}

func getCreateAmtProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "amtprofile name [flags]",
		Short:   "Create an AMT profile",
		Example: createAmtProfileExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runCreateAmtProfileCommand,
	}
	return cmd
}

func getDeleteAmtProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "amtprofile <resourceid> [flags]",
		Short:   "Delete an AMT profile",
		Example: deleteAmtProfileExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runDeleteAmtProfileCommand,
	}
	return cmd
}

// Lists all AMT profiles
func runListAmtProfileCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, rpsClient, projectName, err := getRpsServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := rpsClient.GetAllDomainsWithResponse(ctx, projectName, &rps.GetAllDomainsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err := checkResponse(resp.HTTPResponse, "error while retrieving AMT profiles"); err != nil {
		return err
	}

	// /amtprofiles := resp.JSON200

	//printAmtProfiles(writer, amtprofiles, verbose)

	return writer.Flush()
}

func runCreateAmtProfileCommand(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, rpsClient, projectName, err := getRpsServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := rpsClient.CreateDomainWithResponse(ctx, projectName,
		rps.CreateDomainJSONRequestBody{
			DomainSuffix:                  "lol",
			ProfileName:                   name,
			ProvisioningCert:              "cert",
			ProvisioningCertPassword:      "lol",
			ProvisioningCertStorageFormat: "no idea",
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, "error while creating AMT")
}

func runGetAmtProfileCommand(cmd *cobra.Command, args []string) error {
	// writer, verbose := getOutputContext(cmd)
	// ctx, siteClient, projectName, err := getInfraServiceContext(cmd)
	// if err != nil {
	// 	return err
	// }

	// id := args[0]

	// resp, err := siteClient.SiteServiceGetSiteWithResponse(ctx, projectName,
	// 	"empty", id, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	// if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
	// 	"", "error getting site"); !proceed {
	// 	return err
	// }

	// printSite(writer, resp.JSON200)
	//return writer.Flush()
	return nil
}

func runDeleteAmtProfileCommand(cmd *cobra.Command, args []string) error {
	// id := args[0]

	// ctx, siteClient, projectName, err := getInfraServiceContext(cmd)
	// if err != nil {
	// 	return err
	// }

	// resp, err := siteClient.SiteServiceDeleteSiteWithResponse(ctx, projectName,
	// 	"empty", id, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	// err = checkResponse(resp.HTTPResponse, "error while deleting site")
	// if err != nil {
	// 	if strings.Contains(string(resp.Body), `"message":"site_resource not found"`) {
	// 		return errors.New("site does not exist")
	// 	}
	// }
	// return err
	return nil
}

func printAmtProfiles(writer io.Writer, amtprofiles *struct{ union json.RawMessage }, verbose bool) {
	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\n", "AMT Profile Name", "Info")
	} else {
		var shortHeader = fmt.Sprintf("\n%s", "AMT Profile Name")
		fmt.Fprintf(writer, "%s\n", shortHeader)
	}
	if !verbose {
		fmt.Fprintf(writer, "%s\n", amtprofiles.union)
	} else {
		fmt.Fprintf(writer, "%s\t%s\n", amtprofiles.union, "Placeholder")
	}
}

// Prints output details of site
func printAmtProfile(writer io.Writer, site *infra.SiteResource) {

	// _, _ = fmt.Fprintf(writer, "Name: \t%s\n", *site.Name)
	// _, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *site.ResourceId)
	// _, _ = fmt.Fprintf(writer, "Region: \t%s %s\n", *site.Region.Name, *site.RegionId)
	// _, _ = fmt.Fprintf(writer, "Longtitude: \t%v\n", float64(*site.SiteLng)/10000000)
	// _, _ = fmt.Fprintf(writer, "Latitude: \t%v\n", float64(*site.SiteLat)/10000000)

}
