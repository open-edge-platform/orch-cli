// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const listRegionExamples = `# List all regions
orch-cli list region --project some-project

# List all regions within specific parent region ID - first level only
orch-cli list region --project some-project --region region-aaaa1111"`

const getRegionExamples = `# Get specific region information
orch-cli get region region-aaaa1111 --project some-project`

const createRegionExamples = `# Create specific region
orch-cli create region name --project some-project --type country

# Create specific region as a subregion to another region
orch-cli create region name --project some-project --parent region-bbbb1111 --type country

--type = country/state/county/region/city`

const deleteRegionExamples = `# Delete specific region
orch-cli delete region region-aaaa1111 --project some-project`

const spaces string = "       "
const spaces2 string = ""

type region2Site struct {
	Sites  map[string][]infra.SiteResource
	Region map[string]infra.RegionResource
}

// RegionNode is a nested representation of a region and its children/sites for machine output
type RegionNode struct {
	Region   infra.RegionResource `json:"region"`
	Sites    []infra.SiteResource `json:"sites"`
	Children []RegionNode         `json:"children"`
}

func getListRegionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "region [flags]",
		Short:   "List all regions in tree",
		Example: listRegionExamples,
		Aliases: regionAliases,
		RunE:    runListRegionCommand,
	}
	cmd.PersistentFlags().StringP("region", "r", viper.GetString("region"), "Optional filter provided as part of region list to filter region by parent region")
	addListOrderingFilteringPaginationFlags(cmd, "region")
	addStandardListOutputFlags(cmd)
	// Override default output-type to "tree" for region list; table/json/yaml are also supported
	if f := cmd.Flags().Lookup("output-type"); f != nil {
		f.DefValue = "tree"
		_ = f.Value.Set("tree")
		f.Usage = "output type: tree (default), table, json, yaml"
	}
	return cmd
}

func getGetRegionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "region <resourceid> [flags]",
		Short:   "Get a region",
		Example: getRegionExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: regionAliases,
		RunE:    runGetRegionCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getCreateRegionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "region name [flags]",
		Short:   "Create a region",
		Example: createRegionExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: regionAliases,
		RunE:    runCreateRegionCommand,
	}
	cmd.PersistentFlags().StringP("parent", "f", viper.GetString("parent"), "Optional parent region used to create a sub region: --parent region-aaaa1111")
	cmd.PersistentFlags().StringP("type", "t", viper.GetString("type"), "Mandatory flag to provide a type of region: --type country/state/county/region/city")
	return cmd
}

func getDeleteRegionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "region <resourceid> [flags]",
		Short:   "Delete a region",
		Example: deleteRegionExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: regionAliases,
		RunE:    runDeleteRegionCommand,
	}
	return cmd
}

// Gets specific Region - retrieves list of regions and then filters and outputs
// specifc region by name
func runGetRegionCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, regionClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	id := args[0]

	resp, err := regionClient.RegionServiceGetRegionWithResponse(ctx, projectName,
		id, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting region"); !proceed {
		return err
	}

	region := resp.JSON200
	// GET endpoint does not populate TotalSites; fetch it via list with ShowTotalSites=true
	showTotalSites := true
	filterStr := fmt.Sprintf("resource_id='%s'", id)
	lresp, lerr := regionClient.RegionServiceListRegionsWithResponse(ctx, projectName,
		&infra.RegionServiceListRegionsParams{
			ShowTotalSites: &showTotalSites,
			Filter:         &filterStr,
		}, auth.AddAuthHeader)
	if lerr == nil && lresp.JSON200 != nil && len(lresp.JSON200.Regions) > 0 {
		region.TotalSites = lresp.JSON200.Regions[0].TotalSites
	}

	printRegion(writer, region)
	return writer.Flush()
}

func runCreateRegionCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	parentFlag, _ := cmd.Flags().GetString("parent")
	typeFlag, _ := cmd.Flags().GetString("type")

	ctx, regionClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	err = checkName(name, REGION)
	if err != nil {
		return err
	}

	typeMeta, err := checkType(name, typeFlag)
	if err != nil {
		return err
	}

	var parentID *string
	if parentFlag != "" {
		err = checkID(parentFlag)
		if err != nil {
			return err
		}
		presp, err := regionClient.RegionServiceGetRegionWithResponse(ctx, projectName, parentFlag, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		err = checkResponse(presp.HTTPResponse, presp.Body, "error while creating region - parent region not found")
		if err != nil {
			return processError(err)
		}
		parentID = &parentFlag
	}

	resp, err := regionClient.RegionServiceCreateRegionWithResponse(ctx, projectName,
		infra.RegionServiceCreateRegionJSONRequestBody{
			Name:     &name,
			ParentId: parentID,
			Metadata: typeMeta,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, "error while creating region")
}

func runDeleteRegionCommand(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, regionClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	err = checkID(id)
	if err != nil {
		return err
	}

	resp, err := regionClient.RegionServiceDeleteRegionWithResponse(ctx, projectName,
		id, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	err = checkResponse(resp.HTTPResponse, resp.Body, "error while deleting region")
	if err != nil {
		if strings.Contains(string(resp.Body), `"message":"region_resource not found"`) {
			return errors.New("region does not exist")
		}
	}
	return err
}

// Lists all Regions - retrieves all regions and prints tree
func runListRegionCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	regFlag, _ := cmd.Flags().GetString("region")
	region, err := filterRegionsHelper(regFlag)
	if err != nil {
		return err
	}

	ctx, regionClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	// Validate order-by early so we can pass it to the region list API when appropriate
	validatedOrderBy, err := getValidatedRegionOrderBy(ctx, cmd, regionClient, projectName)
	if err != nil {
		return err
	}

	enableTotalSite := true

	filterFlag, _ := cmd.Flags().GetString("filter")
	var filterString *string
	var filterParts []string
	if regFlag != "" && region != nil {
		filterParts = append(filterParts, fmt.Sprintf("parent_region.resource_id='%s'", *region))
	}
	if filterFlag != "" {
		filterParts = append(filterParts, filterFlag)
	}
	if len(filterParts) > 0 {
		combined := strings.Join(filterParts, " AND ")
		filterString = &combined
	}

	//Get all regions
	// For table output we will sort client-side; for JSON/YAML allow API ordering
	outputType, _ := cmd.Flags().GetString("output-type")
	apiOrderBy := validatedOrderBy
	if outputType == "table" {
		apiOrderBy = nil
	}

	resp, err := regionClient.RegionServiceListRegionsWithResponse(ctx, projectName,
		&infra.RegionServiceListRegionsParams{
			ShowTotalSites: &enableTotalSite,
			Filter:         filterString,
			OrderBy:        apiOrderBy,
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

	// If JSON/YAML output requested, build nested nodes and use GenerateOutput
	if outputType == "json" || outputType == "yaml" {
		// For JSON/YAML output, print flat API response
		result := CommandResult{
			Format:    format.Format("table"),
			Filter:    "",
			OrderBy:   "",
			OutputAs:  toOutputType(outputType),
			NameLimit: -1,
			Data:      resp.JSON200.Regions,
		}
		GenerateOutput(writer, &result)
		return writer.Flush()
	}

	// If table output requested, render flat list via templates and client-side sorting
	if outputType == "table" {
		outputFormat, err := getRegionOutputFormat(cmd, verbose, true)
		if err != nil {
			return err
		}

		// Build flat list of regions from response
		regions := make([]infra.RegionResource, 0, len(resp.JSON200.Regions))
		for _, r := range resp.JSON200.Regions {
			regions = append(regions, r)
		}

		orderBy := ""
		if validatedOrderBy != nil {
			orderBy = *validatedOrderBy
		}

		outputFilter, _ := cmd.Flags().GetString("output-filter")
		result := CommandResult{
			Format:    format.Format(outputFormat),
			Filter:    outputFilter,
			OrderBy:   orderBy,
			OutputAs:  toOutputType(outputType),
			NameLimit: -1,
			Data:      regions,
		}
		GenerateOutput(writer, &result)
		return writer.Flush()
	}

	// Default: print ASCII tree
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

// Returns a validated order-by string for regions, with API hints when necessary
func getValidatedRegionOrderBy(ctx context.Context, cmd *cobra.Command, regionClient infra.ClientWithResponsesInterface, projectName string) (*string, error) {
	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return nil, err
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	if outputType == "table" {
		return normalizeOrderByForClientSorting(raw, infra.RegionResource{})
	}

	// For JSON/YAML, validate with probe using API
	return normalizeOrderByWithAPIProbe(raw, "regions", infra.RegionResource{}, func(orderBy string) (bool, error) {
		pageSize := 1
		offset := 0
		resp, err := regionClient.RegionServiceListRegionsWithResponse(ctx, projectName,
			&infra.RegionServiceListRegionsParams{
				OrderBy:  &orderBy,
				Filter:   nil,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
			return false, nil
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating region order-by"); err != nil {
			return false, err
		}
		return true, nil
	})
}

// Prints output tree of regions
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

// Prints output details of region
func printRegion(writer io.Writer, region *infra.RegionResource) {

	_, _ = fmt.Fprintf(writer, "Name: \t%s\n", *region.Name)
	_, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *region.ResourceId)
	_, _ = fmt.Fprintf(writer, "Parent region: \t%s %s\n", *region.ParentId, *region.ParentRegion.Name)
	_, _ = fmt.Fprintf(writer, "Metadata: \t%s\n", *region.Metadata)
	_, _ = fmt.Fprintf(writer, "TotalSites: \t%v\n", *region.TotalSites)

}

func getRegionOutputFormat(cmd *cobra.Command, verbose bool, forList bool) (string, error) {
	const DEFAULT_REGION_FORMAT = "table{{.ResourceId}}\t{{.Name}}\t{{.ParentId}}\t{{.TotalSites}}"
	const DEFAULT_REGION_VERBOSE_FORMAT = "table{{.ResourceId}}\t{{.Name}}\t{{.ParentId}}\t{{.ParentRegion.Name}}\t{{.TotalSites}}"
	const DEFAULT_REGION_INSPECT_FORMAT = "Name:\t{{.Name}}\nResource ID:\t{{.ResourceId}}\nParent Region:\t{{.ParentRegion.Name}}\nParent ID:\t{{.ParentId}}\nTotalSites:\t{{.TotalSites}}\n"

	if verbose && forList {
		return DEFAULT_REGION_VERBOSE_FORMAT, nil
	}
	if !forList {
		return DEFAULT_REGION_INSPECT_FORMAT, nil
	}
	return resolveTableOutputTemplate(cmd, DEFAULT_REGION_FORMAT, "ORCH_CLI_REGION_OUTPUT_TEMPLATE")
}

func checkName(name string, resource int) error {
	pattern := `^[a-zA-Z-_0-9./: ]+$`
	re := regexp.MustCompile(pattern)

	//The REGION API regex accepts space, but a name with space is not accepted when metadata is derived from it
	if resource == REGION && strings.Contains(name, " ") {
		return errors.New("invalid region name")
	}

	if re.MatchString(name) {
		return nil
	}
	switch resource {
	case REGION:
		return errors.New("invalid region name")
	case SITE:
		return errors.New("invalid site name")
	default:
		return errors.New("invalid resource name")
	}
}

func checkID(id string) error {
	pattern := `^region-[0-9a-f]{8}$`
	re := regexp.MustCompile(pattern)

	if re.MatchString(id) {
		return nil
	}

	return errors.New("invalid region id")
}

func checkType(name string, loctype string) (*[]infra.MetadataItem, error) {

	if loctype == "" {
		return nil, errors.New("--type flag not provided")
	}
	name = strings.ToLower(name)
	switch loctype {
	case "country", "state", "county", "city", "region":
		items := []infra.MetadataItem{
			{Key: loctype, Value: name},
		}
		return &items, nil
	default:
		// Return an error if loctype is not valid.
		return nil, errors.New("invalid type provided must be one of: country/state/county/region/city")
	}
}
