// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"
	"sort"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const listSiteExamples = `# List all sites
orch-cli list site --project some-project

# List all sites within specific parent region ID
orch-cli list site --project some-project --region region-aaaa1111"`

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
