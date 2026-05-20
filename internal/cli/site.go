// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
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

const listSiteExamples = `# List all sites
orch-cli list site --project some-project

# List all sites within specific parent region ID
orch-cli list site --project some-project --region region-aaaa1111"`

const getSiteExamples = `# Get a site by resource ID
orch-cli get site site-aaaa1111 --project some-project

# Get a site by name
orch-cli get site mysite --project some-project`

const createSiteExamples = `# Create specific site

# Create a site in a region by resource ID (default longitude and latitude set to 0)
orch-cli create site name --project some-project --region region-bbbb1111

# Create a site in a region by name
orch-cli create site name --project some-project --region "My Region" --longitude 5 --latitude 5
`
const deleteSiteExamples = `# Delete a site by resource ID
orch-cli delete site site-aaaa1111 --project some-project
# Delete a site by name
orch-cli delete site "my-site" --project some-project`

var queryRegion = "region"

// siteResourceIDPattern matches site resource IDs: "site-" followed by 8 hex chars.
var siteResourceIDPattern = regexp.MustCompile(`^site-[0-9a-f]{8}$`)

func isSiteResourceID(s string) bool {
	return siteResourceIDPattern.MatchString(s)
}

// findSiteByName searches a slice of sites for an exact name match.
// Returns an error if no match is found or if multiple sites share the same name
// (listing the matches so the caller can retry with a resource ID).
func findSiteByName(sites []infra.SiteResource, name string) (infra.SiteResource, error) {
	var matches []infra.SiteResource
	for _, s := range sites {
		if s.Name != nil && *s.Name == name {
			matches = append(matches, s)
		}
	}
	switch len(matches) {
	case 0:
		return infra.SiteResource{}, fmt.Errorf("no site found with name %q", name)
	case 1:
		return matches[0], nil
	default:
		var sb strings.Builder
		fmt.Fprintf(&sb, "multiple sites found with name %q; use a resource ID instead:\n", name)
		for _, m := range matches {
			fmt.Fprintf(&sb, "  name: %s  resource-id: %s\n", derefString(m.Name), derefString(m.ResourceId))
		}
		return infra.SiteResource{}, errors.New(strings.TrimRight(sb.String(), "\n"))
	}
}

func getListSiteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "site [flags]",
		Short:   "List all sites",
		Example: listSiteExamples,
		Aliases: siteAliases,
		RunE:    runListSiteCommand,
	}
	cmd.PersistentFlags().StringP("region", "r", viper.GetString("region"), "Optional filter provided as part of site list to filter sites by parent region")
	addListOrderingFilteringPaginationFlags(cmd, "site")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getGetSiteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "site <name|resourceID> [flags]",
		Short:   "Get a site",
		Example: getSiteExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: siteAliases,
		RunE:    runGetSiteCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getCreateSiteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "site name [flags]",
		Short:   "Create a site",
		Example: createSiteExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: siteAliases,
		RunE:    runCreateSiteCommand,
	}
	cmd.PersistentFlags().StringP("region", "r", viper.GetString("region"), "Region to which the site will be deployed: --region region-aaaa1111 or --region \"My Region\"")
	cmd.PersistentFlags().StringP("latitude", "l", viper.GetString("latitude"), "Optional flag to provide latitude: --latitude 5")
	cmd.PersistentFlags().StringP("longitude", "g", viper.GetString("longitude"), "Optional flag to provide longitude: longitude 5 ")
	return cmd
}

func getDeleteSiteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "site <name|resourceID> [flags]",
		Short:   "Delete a site",
		Example: deleteSiteExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: siteAliases,
		RunE:    runDeleteSiteCommand,
	}
	return cmd
}

// Lists all sites - retrieves all sites and displays selected information in tabular format
func runListSiteCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, siteClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	// Validate order-by flag
	validatedOrderBy, err := getValidatedSiteOrderBy(ctx, cmd, siteClient, projectName)
	if err != nil {
		return err
	}

	// Paging
	pageSize32, offset32, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}
	var pageSize *int
	var offset *int
	if pageSize32 > 0 {
		v := int(pageSize32)
		pageSize = &v
	}
	if offset32 > 0 {
		v := int(offset32)
		offset = &v
	}

	// Filtering
	filterSpec := getNonEmptyFlag(cmd, "filter")
	regFlag, _ := cmd.Flags().GetString("region")
	region, err := filterRegionsHelper(regFlag)
	var regFilter *string
	if err != nil {
		return err
	}
	if region != nil {
		filterString := fmt.Sprintf("region.resource_id='%s' OR region.parent_region.resource_id='%s' OR region.parent_region.parent_region.resource_id='%s' OR region.parent_region.parent_region.parent_region.resource_id='%s'", regFlag, regFlag, regFlag, regFlag)
		regFilter = &filterString
	}
	// Combine region filter and user filter if both present
	var combinedFilter *string
	if regFilter != nil && filterSpec != nil {
		combined := fmt.Sprintf("(%s) AND (%s)", *regFilter, *filterSpec)
		combinedFilter = &combined
	} else if regFilter != nil {
		combinedFilter = regFilter
	} else if filterSpec != nil {
		combinedFilter = filterSpec
	}

	// Validate combined filter with API probe so callers get friendly hints
	var validatedFilter *string
	if combinedFilter != nil {
		vf, err := normalizeFilterWithAPIProbe(*combinedFilter, "sites", infra.SiteResource{}, func(filter string) (bool, error) {
			pageSize := 1
			offset := 0
			resp, err := siteClient.SiteServiceListSitesWithResponse(ctx, projectName, queryRegion,
				&infra.SiteServiceListSitesParams{
					Filter:   &filter,
					PageSize: &pageSize,
					Offset:   &offset,
				}, auth.AddAuthHeader)
			if err != nil {
				return false, processError(err)
			}
			if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
				return false, &api400Error{string(resp.Body)}
			}
			if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating site filter"); err != nil {
				return false, err
			}
			return true, nil
		})
		if err != nil {
			return err
		}
		validatedFilter = vf
	}

	sites := make([]infra.SiteResource, 0)
	outputType, _ := cmd.Flags().GetString("output-type")
	apiOrderBy := validatedOrderBy
	if outputType == "table" {
		// For table output, do not send order-by to API (client-side sort)
		apiOrderBy = nil
	}
	for {
		resp, err := siteClient.SiteServiceListSitesWithResponse(ctx, projectName, queryRegion,
			&infra.SiteServiceListSitesParams{
				Filter:   validatedFilter,
				OrderBy:  apiOrderBy,
				PageSize: pageSize,
				Offset:   offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error while retrieving sites"); err != nil {
			return err
		}
		sites = append(sites, resp.JSON200.Sites...)
		if !resp.JSON200.HasNext {
			break
		}
		// Advance offset for next page
		if offset == nil {
			v := len(sites)
			offset = &v
		} else {
			v := *offset + len(resp.JSON200.Sites)
			offset = &v
		}
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printSites(cmd, writer, &sites, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

func runCreateSiteCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	name := args[0]

	regFlag, _ := cmd.Flags().GetString("region")
	ltdFlag, _ := cmd.Flags().GetString("latitude")
	lngFlag, _ := cmd.Flags().GetString("longitude")

	if regFlag == "" || strings.HasPrefix(regFlag, "--") {
		return errors.New("region flag required")
	}

	ctx, siteClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	// Resolve --region: accept resource ID or region name.
	var regionID string
	if isRegionResourceID(regFlag) {
		regionID = regFlag
	} else {
		lresp, err := siteClient.RegionServiceListRegionsWithResponse(ctx, projectName,
			&infra.RegionServiceListRegionsParams{}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(lresp.HTTPResponse, lresp.Body, "error while retrieving regions"); err != nil {
			return err
		}
		r, err := findRegionByName(lresp.JSON200.Regions, regFlag)
		if err != nil {
			return err
		}
		regionID = derefString(r.ResourceId)
	}

	err = checkName(name, SITE)
	if err != nil {
		return err
	}

	siteLat, err := resolveLatitude(ltdFlag)
	if err != nil {
		return err
	}
	siteLng, err := resolveLongitude(lngFlag)
	if err != nil {
		return err
	}

	rresp, err := siteClient.RegionServiceGetRegionWithResponse(ctx, projectName,
		regionID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(rresp.HTTPResponse, rresp.Body, writer, verbose,
		"", "the region for site creation does not exist"); !proceed {
		return err
	}

	resp, err := siteClient.SiteServiceCreateSiteWithResponse(ctx, projectName, "empty",
		infra.SiteServiceCreateSiteJSONRequestBody{
			Name:     &name,
			SiteLat:  siteLat,
			SiteLng:  siteLng,
			RegionId: &regionID,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, "error while creating site")
}

func runGetSiteCommand(cmd *cobra.Command, args []string) error {
	writer, _ := getOutputContext(cmd)
	ctx, siteClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	query := args[0]

	if isSiteResourceID(query) {
		resp, err := siteClient.SiteServiceGetSiteWithResponse(ctx, projectName,
			"empty", query, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, false,
			"", "error getting site"); !proceed {
			return err
		}
		if err := printSite(cmd, writer, resp.JSON200); err != nil {
			return err
		}
		return writer.Flush()
	}

	// Name-based lookup: list all sites and filter by name.
	resp, err := siteClient.SiteServiceListSitesWithResponse(ctx, projectName, queryRegion,
		&infra.SiteServiceListSitesParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, "error while retrieving sites"); err != nil {
		return err
	}

	site, err := findSiteByName(resp.JSON200.Sites, query)
	if err != nil {
		return err
	}
	if err := printSite(cmd, writer, &site); err != nil {
		return err
	}
	return writer.Flush()
}

func runDeleteSiteCommand(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, siteClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	if !isSiteResourceID(id) {
		// Name-based lookup: list all sites and filter by name.
		resp, err := siteClient.SiteServiceListSitesWithResponse(ctx, projectName, queryRegion,
			&infra.SiteServiceListSitesParams{}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error while retrieving sites"); err != nil {
			return err
		}
		site, err := findSiteByName(resp.JSON200.Sites, id)
		if err != nil {
			return err
		}
		id = derefString(site.ResourceId)
	}

	resp, err := siteClient.SiteServiceDeleteSiteWithResponse(ctx, projectName,
		"empty", id, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	err = checkResponse(resp.HTTPResponse, resp.Body, "error while deleting site")
	if err != nil {
		if strings.Contains(string(resp.Body), `"message":"site_resource not found"`) {
			return errors.New("site does not exist")
		}
	}
	return err
}

func printSites(cmd *cobra.Command, writer io.Writer, sites *[]infra.SiteResource, orderBy *string, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getSiteOutputFormat(cmd, verbose, true)
	if err != nil {
		return err
	}

	sortSpec := ""
	filterSpec := ""
	if outputType == "table" && orderBy != nil {
		sortSpec = *orderBy
	}
	if outputType == "table" && outputFilter != nil && *outputFilter != "" {
		filterSpec = *outputFilter
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    filterSpec,
		OrderBy:   sortSpec,
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      *sites,
	}
	GenerateOutput(writer, &result)
	return nil
}

func getSiteOutputFormat(cmd *cobra.Command, verbose bool, forList bool) (string, error) {
	const DEFAULT_SITE_FORMAT = "table{{.ResourceId}}\t{{.Name}}\t{{.RegionId}}\t{{.Region.Name}}"
	const DEFAULT_SITE_VERBOSE_FORMAT = "table{{.ResourceId}}\t{{.Name}}\t{{.RegionId}}\t{{.Region.Name}}\t{{.SiteLng}}\t{{.SiteLat}}"
	const DEFAULT_SITE_INSPECT_FORMAT = "Name:\t{{.Name}}\nResource ID:\t{{.ResourceId}}\nRegion Name:\t{{.Region.Name}}\nRegion ID:\t{{.RegionId}}\nLongitude:\t{{.SiteLng}}\nLatitude:\t{{.SiteLat}}\n"

	if verbose && forList {
		return DEFAULT_SITE_VERBOSE_FORMAT, nil
	}
	if !forList {
		return resolveTableOutputTemplate(cmd, DEFAULT_SITE_INSPECT_FORMAT, "ORCH_CLI_SITE_INSPECT_TEMPLATE")
	}
	return resolveTableOutputTemplate(cmd, DEFAULT_SITE_FORMAT, "ORCH_CLI_SITE_OUTPUT_TEMPLATE")
}

// Prints output details of site using template-based output
func printSite(cmd *cobra.Command, writer io.Writer, site *infra.SiteResource) error {
	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getSiteOutputFormat(cmd, true, false)
	if err != nil {
		return err
	}
	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    "",
		OrderBy:   "",
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      *site,
	}
	GenerateOutput(writer, &result)
	return nil
}

func resolveLatitude(value string) (*int32, error) {
	defaultVal := int32(0)
	if value == "" {
		return &defaultVal, nil
	}

	parsedValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, errors.New("invalid latitude value")
	}

	scaling := 10000000
	int32Value := int32(parsedValue * float64(scaling))
	return &int32Value, nil
}

func resolveLongitude(value string) (*int32, error) {
	defaultVal := int32(0)
	if value == "" {
		return &defaultVal, nil
	}

	parsedValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, errors.New("invalid longitude value")
	}

	scaling := 10000000
	int32Value := int32(parsedValue * float64(scaling))
	return &int32Value, nil
}

// Returns a validated order-by string for the site resource, with hints for valid fields
func getValidatedSiteOrderBy(_ interface{}, cmd *cobra.Command, siteClient infra.ClientWithResponsesInterface, projectName string) (*string, error) {
	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return nil, err
	}
	outputType, _ := cmd.Flags().GetString("output-type")
	// For table output, allow any struct field (client-side sort)
	if outputType == "table" {
		return normalizeOrderByForClientSorting(raw, infra.SiteResource{})
	}
	// For JSON/YAML, normalize and validate with API probe
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	// Normalize order direction for API: -field -> 'field desc', +field -> 'field asc', field -> 'field'
	normalized := raw
	if len(normalized) > 0 {
		switch normalized[0] {
		case '-':
			normalized = normalized[1:] + " desc"
		case '+':
			normalized = normalized[1:] + " asc"
		}
	}
	pageSize := 1
	offset := 0
	resp, err := siteClient.SiteServiceListSitesWithResponse(context.Background(), projectName, queryRegion,
		&infra.SiteServiceListSitesParams{
			OrderBy:  &normalized,
			PageSize: &pageSize,
			Offset:   &offset,
		}, auth.AddAuthHeader)
	if err != nil {
		return nil, processError(err)
	}
	if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == 400 {
		// Try to extract error message and provide a hint
		msg := string(resp.Body)
		return nil, fmt.Errorf("invalid --order-by field '%s': %s\nValid fields: name, resourceId, regionId, region.name", raw, msg)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating site order-by"); err != nil {
		return nil, err
	}
	return &normalized, nil
}
