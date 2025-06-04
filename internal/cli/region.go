// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const listRegionExamples = `# List all regions
orch-cli list region --project some-project

# List all regions within specific parent region ID - first level only
orch-cli list region --project some-project --region region-aaaa1111"`

const getRegionExamples = `# Get specific region information using regions resource ID
orch-cli get region region-aaaa1111 --project some-project"`

const spaces string = "       "
const spaces2 string = ""

type region2Site struct {
	Sites  map[string][]infra.Site
	Region map[string]infra.Region
}

func getGetRegionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "region <region ID> [flags]",
		Short:   "Get a region",
		Example: getRegionExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runGetRegionCommand,
	}
	return cmd
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

// Gets specific OS Profile - retrieves list of profiles and then filters and outputs
// specifc profile by name
func runGetRegionCommand(cmd *cobra.Command, args []string) error {
	// writer, verbose := getOutputContext(cmd)
	// ctx, OSProfileClient, projectName, err := getInfraServiceContext(cmd)
	// if err != nil {
	// 	return err
	// }

	// resp, err := OSProfileClient.GetV1ProjectsProjectNameComputeOsWithResponse(ctx, projectName,
	// 	&infra.GetV1ProjectsProjectNameComputeOsParams{}, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	// if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
	// 	OSProfileHeaderGet, "error getting OS Profile"); !proceed {
	// 	return err
	// }

	// name := args[0]
	// profile, err := filterProfilesByName(resp.JSON200.OperatingSystemResources, name)
	// if err != nil {
	// 	return err
	// }

	// printOSProfile(writer, profile)
	// return writer.Flush()
	return nil
}

// Lists all OS Profiles - retrieves all profiles and displays selected information in tabular format
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

	//Get all regions
	resp, err := regionClient.GetV1ProjectsProjectNameRegionsWithResponse(ctx, projectName,
		&infra.GetV1ProjectsProjectNameRegionsParams{
			ShowTotalSites: &enableTotalSite,
			Parent:         region,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting regions"); !proceed {
		return err
	}

	regionMap := region2Site{
		Sites:  make(map[string][]infra.Site),
		Region: make(map[string]infra.Region),
	}

	//Map sites to region
	for _, region := range *resp.JSON200.Regions {
		regionMap.Region[*region.ResourceId] = region
		//Get all sites per region
		regFilter := fmt.Sprintf("region.resource_id='%s'", *region.ResourceId)

		sresp, err := regionClient.GetV1ProjectsProjectNameRegionsRegionIDSitesWithResponse(ctx, projectName, *region.ResourceId,
			&infra.GetV1ProjectsProjectNameRegionsRegionIDSitesParams{
				Filter: &regFilter,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		regionMap.Sites[*region.ResourceId] = *sresp.JSON200.Sites
	}

	printRegions(writer, regionMap, verbose)

	return writer.Flush()
}

func printRegions(writer io.Writer, regions region2Site, verbose bool) {
	if !verbose {
		//fmt.Fprintf(writer, "%s\t%s\n", "Region ID", "Region name")
	} else {
		//fmt.Fprintf(writer, "%s\t%s\t%s\n", "Region ID", "Region name", "Total sites")
	}

	//var levelList []string

	for _, region := range regions.Region {
		if !verbose {
			if region.ParentId == nil {
				fmt.Fprintf(writer, "Region: %s (%s)\n", *region.ResourceId, *region.Name)
				fmt.Fprintf(writer, "  |\n")
				for _, site := range regions.Sites[*region.ResourceId] {
					fmt.Fprintf(writer, "  └───── Site: %s (%s)\n", *site.ResourceId, *site.Name)
				}
				printSubRegions(writer, regions, *region.RegionID, false, spaces, spaces2)
			}
			fmt.Fprintln(writer)

		} else {
			if region.ParentId == nil {
				fmt.Fprintf(writer, "Region: %s (%s)\n - Total Sites: %v\n", *region.ResourceId, *region.Name, *region.TotalSites)
				fmt.Fprintf(writer, "  |\n")
				for _, site := range regions.Sites[*region.ResourceId] {
					fmt.Fprintf(writer, "  └───── Site: %s (%s)\n", *site.ResourceId, *site.Name)
				}
				printSubRegions(writer, regions, *region.RegionID, true, spaces, spaces2)
			}

			fmt.Fprintln(writer)
		}
	}
}

func printSubRegions(writer io.Writer, regions region2Site, parentRegion string, verbose bool, spaces string, spaces2 string) {

	track := 0
	totalRegions := len(regions.Region)

	for _, region := range regions.Region {
		track++
		if region.ParentId != nil && *region.ParentId == parentRegion {
			fmt.Fprintf(writer, "  %s└───── Region: %s (%s)\n", spaces2, *region.ResourceId, *region.Name)
			fmt.Fprintf(writer, "  %s|\n", spaces)
			for _, site := range regions.Sites[*region.ResourceId] {
				fmt.Fprintf(writer, "  %s└───── Site: %s (%s)\n", spaces, *site.ResourceId, *site.Name)
			}
			if track == totalRegions-1 {
				fmt.Fprintf(writer, "  |\n")
			}
			spaces = spaces + "       "
			spaces2 = spaces2 + "       "
			printSubRegions(writer, regions, *region.RegionID, false, spaces, spaces2)
		}
	}
}
