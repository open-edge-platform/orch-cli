// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const listSiteExamples = `# List all sites
orch-cli list site --project some-project

# List all sites within specific parent region ID
orch-cli list site --project some-project --region region-aaaa1111"`

const getSiteExamples = `# Get specific site information
orch-cli get site site-aaaa1111 --project some-project`

const createSiteExamples = `# Create specific site

# Create a site in a region (default longitude and latitude set to 0)
orch-cli create site name --project some-project --region region-bbbb1111

# Create a site in a region (default longitude and latitude set to 0)
orch-cli create site name --project some-project --region region-bbbb1111 --longitude 5 --latitude 5
`
const deleteSiteExamples = `# Delete specific site
orch-cli delete site region-aaaa1111 --project some-project`

var queryRegion = "region"

func getListSiteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "site [flags]",
		Short:   "List all sites",
		Example: listSiteExamples,
		RunE:    runListSiteCommand,
	}
	cmd.PersistentFlags().StringP("region", "r", viper.GetString("region"), "Optional filter provided as part of site list to filter sites by parent region")
	return cmd
}

func getGetSiteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "site <resourceid> [flags]",
		Short:   "Get a site",
		Example: getSiteExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runGetSiteCommand,
	}
	return cmd
}

func getCreateSiteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "site name [flags]",
		Short:   "Create a site",
		Example: createSiteExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runCreateSiteCommand,
	}
	cmd.PersistentFlags().StringP("region", "r", viper.GetString("region"), "Region to which the site will be deployed: --region region-aaaa1111")
	cmd.PersistentFlags().StringP("latitude", "l", viper.GetString("latitude"), "Optional flag to provide latitude: --latitude 5")
	cmd.PersistentFlags().StringP("longtitude", "g", viper.GetString("longtitude"), "Optional flag to provide longtitude: longtitude 5 ")
	return cmd
}

func getDeleteSiteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "site <resourceid> [flags]",
		Short:   "Delete a site",
		Example: deleteSiteExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runDeleteSiteCommand,
	}
	return cmd
}

// Lists all sites - retrieves all sites and displays selected information in tabular format
func runListSiteCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	var regFilter *string
	regFlag, _ := cmd.Flags().GetString("region")
	region, err := filterRegionsHelper(regFlag)
	if err != nil {
		return err
	}
	if region != nil {
		filterString := fmt.Sprintf("region.resource_id='%s' OR region.parent_region.resource_id='%s' OR region.parent_region.parent_region.resource_id='%s' OR region.parent_region.parent_region.parent_region.resource_id='%s'", regFlag, regFlag, regFlag, regFlag)
		regFilter = &filterString
	}

	ctx, siteClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	pageSize := 20
	sites := make([]infra.SiteResource, 0)

	for offset := 0; ; offset += pageSize {
		resp, err := siteClient.SiteServiceListSitesWithResponse(ctx, projectName, queryRegion,
			&infra.SiteServiceListSitesParams{
				Filter: regFilter,
				Offset: &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}

		if err := checkResponse(resp.HTTPResponse, "error while retrieving sites"); err != nil {
			return err
		}

		sites = append(sites, resp.JSON200.Sites...)
		if !resp.JSON200.HasNext {
			break // No more hosts to process
		}
	}
	printSites(writer, &sites, verbose, regFlag)

	return writer.Flush()
}

func runCreateSiteCommand(cmd *cobra.Command, args []string) error {
	name := args[0]

	regFlag, _ := cmd.Flags().GetString("region")
	ltdFlag, _ := cmd.Flags().GetString("latitude")
	lngFlag, _ := cmd.Flags().GetString("longtitude")

	if regFlag == "" {
		return errors.New("region flag required")
	}
	region, err := filterRegionsHelper(regFlag)
	if err != nil {
		return err
	}

	ctx, siteClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	err = checkName(name)
	if err != nil {
		return err
	}

	siteLat, err := resolveLatitude(ltdFlag)
	if err != nil {
		return err
	}
	siteLng, err := resolveLongtitude(lngFlag)
	if err != nil {
		return err
	}

	//TODO check if region exists

	resp, err := siteClient.SiteServiceCreateSiteWithResponse(ctx, projectName, *region,
		infra.SiteServiceCreateSiteJSONRequestBody{
			Name:    &name,
			SiteLat: siteLat,
			SiteLng: siteLng,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, "error while creating region")
}

func runGetSiteCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, siteClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	id := args[0]

	resp, err := siteClient.SiteServiceGetSiteWithResponse(ctx, projectName,
		"empty", id, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting site"); !proceed {
		return err
	}

	printSite(writer, resp.JSON200)
	return writer.Flush()
}

func runDeleteSiteCommand(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, siteClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := siteClient.SiteServiceDeleteSiteWithResponse(ctx, projectName,
		"empty", id, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	err = checkResponse(resp.HTTPResponse, "error while deleting site")
	if err != nil {
		if strings.Contains(string(resp.Body), `"message":"site_resource not found"`) {
			return errors.New("site does not exist")
		}
	}
	return err
}

func printSites(writer io.Writer, sites *[]infra.SiteResource, verbose bool, region string) {
	if verbose {
		fmt.Fprintf(writer, "%-20s %-20s %-30s %-10s %-10s\n", "Site ID", "Site Name", "Region (Name)", "Longitude", "Latitude")
	} else {
		fmt.Fprintf(writer, "%-20s %-20s %-30s\n", "Site ID", "Site Name", "Region (Name)")
	}

	sitesSlice := *sites
	// Sort sites by RegionId
	sort.Slice(sitesSlice, func(i, j int) bool {
		if *sitesSlice[i].RegionId == region {
			return true
		}
		if *sitesSlice[j].RegionId == region {
			return false
		}
		return *sitesSlice[i].RegionId < *sitesSlice[j].RegionId
	})

	isSubregion := 0
	for _, s := range sitesSlice {
		regionDisplay := fmt.Sprintf("%s (%s)", *s.RegionId, *s.Region.Name)

		if !verbose {
			if region == "" {
				fmt.Fprintf(writer, "%-20s %-20v %-30s\n", *s.ResourceId, *s.Name, regionDisplay)
			} else {
				if *s.RegionId != region {
					isSubregion++
				}
				if isSubregion == 1 {
					fmt.Fprintf(writer, "\nSites in sub-regions:\n\n")
				}
				fmt.Fprintf(writer, "%-20s %-20v %-30s\n", *s.ResourceId, *s.Name, regionDisplay)
			}
		} else {
			if region == "" {
				fmt.Fprintf(writer, "%-20s %-20v %-30s %-10v %-10v\n", *s.ResourceId, *s.Name, regionDisplay, *s.SiteLng, *s.SiteLat)
			} else {
				if *s.RegionId != region {
					isSubregion++
				}
				if isSubregion == 1 {
					fmt.Fprintf(writer, "\nSites in sub-regions:\n\n")
				}
				fmt.Fprintf(writer, "%-20s %-20v %-30s %-10v %-10v\n", *s.ResourceId, *s.Name, regionDisplay, *s.SiteLng, *s.SiteLat)
			}
		}
	}
}

// Prints output details of site
func printSite(writer io.Writer, site *infra.SiteResource) {

	_, _ = fmt.Fprintf(writer, "Name: \t%s\n", *site.Name)
	_, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *site.ResourceId)
	_, _ = fmt.Fprintf(writer, "Region: \t%s %s\n", *site.Region.Name, *site.RegionId)
	_, _ = fmt.Fprintf(writer, "Longtitude: \t%v\n", *site.SiteLng)
	_, _ = fmt.Fprintf(writer, "Latitude: \t%v\n", *site.SiteLat)

}

func resolveLatitude(value string) (*int32, error) {
	defaultVal := int32(0)
	if value == "" {
		return &defaultVal, nil
	}

	parsedValue, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil, errors.New("invalid latitude value")
	}

	int32Value := int32(parsedValue)
	return &int32Value, nil
}

func resolveLongtitude(value string) (*int32, error) {
	defaultVal := int32(0)
	if value == "" {
		return &defaultVal, nil
	}

	parsedValue, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil, errors.New("invalid longtitude value")
	}

	int32Value := int32(parsedValue)
	return &int32Value, nil
}
