// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const listRegionExamples = `# List all regions
orch-cli list region --project some-project

# List all regions within specific parent region ID - first level only
orch-cli list region --project some-project --region region-aaaa1111"`

const spaces string = "       "
const spaces2 string = ""

type region2Site struct {
	Sites  map[string][]infra.SiteResource
	Region map[string]infra.RegionResource
}

func getListRegionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "region [flags]",
		Short:   "List all regions in tree",
		Example: listRegionExamples,
		RunE:    runListRegionCommand,
	}
	cmd.PersistentFlags().StringP("region", "r", viper.GetString("region"), "Optional filter provided as part of region list to filter region by parent region")
	return cmd
}

// Lists all Regions - retrieves all regions and prints tree
func runListRegionCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	regFlag, _ := cmd.Flags().GetString("region")
	region, err := filterRegionsHelper(regFlag)
	if err != nil {
		return err
	}

	ctx, regionClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	enableTotalSite := true

	var filterString *string
	if regFlag != "" {
		filter := fmt.Sprintf("parent_region.resource_id='%s'", *region)
		filterString = &filter
	}

	//Get all regions
	resp, err := regionClient.RegionServiceListRegionsWithResponse(ctx, projectName,
		&infra.RegionServiceListRegionsParams{
			ShowTotalSites: &enableTotalSite,
			Filter:         filterString,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting regions"); !proceed {
		return err
	}

	regionMap := region2Site{
		Sites:  make(map[string][]infra.SiteResource),
		Region: make(map[string]infra.RegionResource),
	}

	//Map sites to region
	for _, region := range resp.JSON200.Regions {
		regionMap.Region[*region.ResourceId] = region
		//Get all sites per region
		regFilter := fmt.Sprintf("region.resource_id='%s'", *region.ResourceId)

		sresp, err := regionClient.SiteServiceListSitesWithResponse(ctx, projectName, *region.ResourceId,
			&infra.SiteServiceListSitesParams{
				Filter: &regFilter,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		regionMap.Sites[*region.ResourceId] = sresp.JSON200.Sites
	}

	printRegions(writer, regionMap, verbose, region)

	return writer.Flush()
}

func printRegions(writer io.Writer, regions region2Site, verbose bool, regionflag *string) {

	fmt.Fprintf(writer, "Printing regions tree\n\n")

	for _, region := range regions.Region {
		if !verbose {
			if *region.ParentId == "" || (*region.ParentId != "" && regionflag != nil && *region.ParentId == *regionflag) {
				fmt.Fprintf(writer, "Region: %s (%s)\n", *region.ResourceId, *region.Name)
				fmt.Fprintf(writer, "  |\n")
				for _, site := range regions.Sites[*region.ResourceId] {
					fmt.Fprintf(writer, "  └───── Site: %s (%s)\n", *site.ResourceId, *site.Name)
				}
				printSubRegions(writer, regions, *region.RegionID, false, spaces, spaces2)
				fmt.Fprintln(writer)
			}

		} else {
			if *region.ParentId == "" || (*region.ParentId != "" && regionflag != nil && *region.ParentId == *regionflag) {
				fmt.Fprintf(writer, "Region: %s (%s)\n- Total Sites: %v\n", *region.ResourceId, *region.Name, *region.TotalSites)
				fmt.Fprintf(writer, "  |\n")
				for _, site := range regions.Sites[*region.ResourceId] {
					fmt.Fprintf(writer, "  └───── Site: %s (%s)\n", *site.ResourceId, *site.Name)
				}
				printSubRegions(writer, regions, *region.RegionID, true, spaces, spaces2)
				fmt.Fprintln(writer)
			}
		}
	}
}

func printSubRegions(writer io.Writer, regions region2Site, parentRegion string, verbose bool, spaces string, spaces2 string) {

	//totalRegions := len(regions.Region)
	totalSites := ""
	//totalSites := "\n- Total Sites:" + *region.TotalSites
	for _, region := range regions.Region {
		if region.ParentId != nil && *region.ParentId == parentRegion {
			if verbose {
				totalSites = "\n         " + spaces2 + "- Total Sites: " + strconv.FormatInt(int64(*region.TotalSites), 10)
			}
			fmt.Fprintf(writer, "\n  %s└───── Region: %s (%s)%s\n", spaces2, *region.ResourceId, *region.Name, totalSites)
			if len(regions.Sites[*region.ResourceId]) > 0 {
				fmt.Fprintf(writer, "  %s|\n", spaces)
			}
			for _, site := range regions.Sites[*region.ResourceId] {
				fmt.Fprintf(writer, "  %s└───── Site: %s (%s)\n", spaces, *site.ResourceId, *site.Name)
			}
			spaces = spaces + "       "
			spaces2 = spaces2 + "       "
			printSubRegions(writer, regions, *region.RegionID, verbose, spaces, spaces2)
			spaces = spaces[:len(spaces)-7]
			spaces2 = spaces2[:len(spaces2)-7]
		}
	}
}
