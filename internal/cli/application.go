// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/internal/validator"
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
)

const (
	DEFAULT_APPLICATION_FORMAT         = "table{{.Name}}\t{{.DisplayName}}\t{{.Version}}\t{{.Kind}}\t{{.ChartName}}\t{{.ChartVersion}}\t{{.HelmRegistryName}}\t{{.DefaultProfileName}}"
	DEFAULT_APPLICATION_INSPECT_FORMAT = `Name: {{.Name}}
Display Name: {{str .DisplayName}}
Description: {{str .Description}}
Version: {{.Version}}
Kind: {{.Kind}}
Helm Registry Name: {{.HelmRegistryName}}
Image Registry Name: {{str .ImageRegistryName}}
Chart Name: {{.ChartName}}
Chart Version: {{.ChartVersion}}
Create Time: {{.CreateTime}}
Update Time: {{.UpdateTime}}
Profiles:{{- range deref .Profiles}}
  {{.Name}}{{- end}}
Default Profile: {{str .DefaultProfileName}}
`
	APPLICATION_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_APPLICATION_OUTPUT_TEMPLATE"
)

func getCreateApplicationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "application <name> <version> [flags]",
		Aliases: applicationAliases,
		Short:   "Create an application",
		Args:    cobra.ExactArgs(2),
		Example: "orch-cli create application my-app 1.0.0 --chart-name my-chart --chart-version 1.0.0 --chart-registry my-registry --project some-project",
		RunE:    runCreateApplicationCommand,
	}
	addEntityFlags(cmd, "application")
	cmd.Flags().String("chart-name", "", "Helm chart name for deploying the application (required)")
	cmd.Flags().String("chart-version", "", "Helm chart version (required)")
	cmd.Flags().String("chart-registry", "", "Helm chart registry (required)")
	cmd.Flags().String("image-registry", "", "image registry")
	cmd.Flags().String("kind", "normal", "application kind: normal, addon, extension")
	return cmd
}

func getListApplicationsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "applications [flags]",
		Aliases: applicationAliases,
		Short:   "List all applications",
		Example: "orch-cli list applications --project some-project",
		RunE:    runListApplicationsCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "application")
	cmd.Flags().StringSlice("kind", []string{}, "application kind: normal, addon, extension")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getGetApplicationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "application <name> [<version>] [flags]",
		Aliases: applicationAliases,
		Short:   "Get an application",
		Args:    cobra.RangeArgs(1, 2),
		Example: "orch-cli get application my-app --project some-project",
		RunE:    runGetApplicationCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getSetApplicationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "application <name> <version> [flags]",
		Aliases: applicationAliases,
		Short:   "Update an application",
		Args:    cobra.ExactArgs(2),
		Example: "orch-cli set application my-app 1.0.0 --display-name 'My Application' --description 'An example application' --chart-name my-chart --chart-version 1.0.0 --chart-registry my-registry --project some-project",
		RunE:    runSetApplicationCommand,
	}
	addEntityFlags(cmd, "application")
	cmd.Flags().String("chart-name", "", "Helm chart name for deploying the application")
	cmd.Flags().String("chart-version", "", "Helm chart version")
	cmd.Flags().String("chart-registry", "", "Helm chart registry")
	cmd.Flags().String("image-registry", "", "image registry")
	cmd.Flags().String("default-profile", "", "default profile name")
	return cmd
}

func getDeleteApplicationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "application <name> <version> [flags]",
		Aliases: applicationAliases,
		Short:   "Delete an application",
		Args:    cobra.RangeArgs(1, 2),
		Example: "orch-cli delete application my-app 1.0.0 --project some-project",
		RunE:    runDeleteApplicationCommand,
	}
	return cmd
}

func runCreateApplicationCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	version := args[1]

	// Validate version format
	if err := validator.ValidateVersion(version); err != nil {
		return err
	}

	// Validate required flags
	chartName, _ := cmd.Flags().GetString("chart-name")
	chartVersion, _ := cmd.Flags().GetString("chart-version")
	chartRegistry, _ := cmd.Flags().GetString("chart-registry")

	if chartName == "" || chartVersion == "" || chartRegistry == "" {
		return fmt.Errorf("--chart-name, --chart-version, and --chart-registry are required")
	}

	// Validate chart version format
	if err := validator.ValidateVersion(chartVersion); err != nil {
		return fmt.Errorf("invalid chart version: %w", err)
	}

	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}
	displayName, description, err := getEntityFlags(cmd)
	if err != nil {
		return err
	}

	defaultKind := catapi.KINDNORMAL

	resp, err := catalogClient.CatalogServiceCreateApplicationWithResponse(ctx, projectName,
		catapi.CatalogServiceCreateApplicationJSONRequestBody{
			Name:              name,
			Version:           version,
			Kind:              getApplicationKind(cmd, &defaultKind),
			DisplayName:       &displayName,
			Description:       &description,
			ChartName:         *getFlag(cmd, "chart-name"),
			ChartVersion:      *getFlag(cmd, "chart-version"),
			HelmRegistryName:  *getFlag(cmd, "chart-registry"),
			ImageRegistryName: getFlag(cmd, "image-registry"),
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating application %s", name)); err != nil {
		return err
	}
	fmt.Printf("Application '%s:%s' created successfully\n", name, version)
	return nil
}

func applicationKind2String(kind *catapi.CatalogV3Kind) string {
	if kind == nil {
		return "normal"
	}
	switch *kind {
	case catapi.KINDNORMAL:
		return "normal"
	case catapi.KINDADDON:
		return "addon"
	case catapi.KINDEXTENSION:
		return "extension"
	}
	return "normal"
}

func string2ApplicationKind(kind string) catapi.CatalogV3Kind {
	switch kind {
	case "normal":
		return catapi.KINDNORMAL
	case "addon":
		return catapi.KINDADDON
	case "extension":
		return catapi.KINDEXTENSION
	}
	return catapi.KINDNORMAL
}

func getApplicationKind(cmd *cobra.Command, def *catapi.CatalogV3Kind) *catapi.CatalogV3Kind {
	dv := applicationKind2String(def)
	kind := string2ApplicationKind(*getFlagOrDefault(cmd, "kind", &dv))
	return &kind
}

func getApplicationKinds(cmd *cobra.Command) *[]catapi.CatalogV3Kind {
	kinds, _ := cmd.Flags().GetStringSlice("kind")
	if len(kinds) == 0 {
		return nil
	}
	list := make([]catapi.CatalogV3Kind, 0, len(kinds))
	for _, k := range kinds {
		list = append(list, string2ApplicationKind(k))
	}
	return &list
}

func getApplicationOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_APPLICATION_INSPECT_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_APPLICATION_FORMAT, APPLICATION_OUTPUT_TEMPLATE_ENVVAR)
}

func printApplications(cmd *cobra.Command, writer io.Writer, appList *[]catapi.CatalogV3Application, orderBy *string, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getApplicationOutputFormat(cmd, verbose)
	if err != nil {
		return err
	}

	sortSpec := ""
	if outputType == "table" && orderBy != nil {
		sortSpec = *orderBy
	}

	filterSpec := ""
	if outputType == "table" && outputFilter != nil && *outputFilter != "" {
		filterSpec = *outputFilter
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    filterSpec,
		OrderBy:   sortSpec,
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      *appList,
	}

	GenerateOutput(writer, &result)
	return nil
}

func runListApplicationsCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	apiOrderBy := getFlag(cmd, "order-by")
	var clientOrderBy *string
	if outputType == "table" {
		// Table output sorts locally via GenerateOutput(CommandResult.OrderBy).
		// Validate client-side ordering fields.
		if apiOrderBy != nil {
			var sampleApp catapi.CatalogV3Application
			clientOrderBy, err = normalizeOrderByForClientSorting(*apiOrderBy, sampleApp)
			if err != nil {
				return err
			}
		}
		apiOrderBy = nil
	}

	pageSize, offset, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	// Preserve explicit pagination requests as single-page results.
	if cmd.Flags().Changed("page-size") || cmd.Flags().Changed("offset") {
		resp, err := catalogClient.CatalogServiceListApplicationsWithResponse(ctx, projectName,
			&catapi.CatalogServiceListApplicationsParams{
				Kinds:    getApplicationKinds(cmd),
				OrderBy:  apiOrderBy,
				Filter:   getFlag(cmd, "filter"),
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing applications"); !proceed {
			return err
		}
		outputFilter, _ := cmd.Flags().GetString("output-filter")
		if err := printApplications(cmd, writer, &resp.JSON200.Applications, clientOrderBy, &outputFilter, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	allApplications := make([]catapi.CatalogV3Application, 0)

	resp, err := catalogClient.CatalogServiceListApplicationsWithResponse(ctx, projectName,
		&catapi.CatalogServiceListApplicationsParams{
			Kinds:    getApplicationKinds(cmd),
			OrderBy:  apiOrderBy,
			Filter:   getFlag(cmd, "filter"),
			PageSize: &pageSize,
			Offset:   &offset,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
		"error listing applications"); !proceed {
		return err
	}

	allApplications = append(allApplications, resp.JSON200.Applications...)
	totalElements := int(resp.JSON200.TotalElements)

	// When page size is omitted (0), derive increment from the first page length.
	if pageSize <= 0 {
		pageSize = int32(len(resp.JSON200.Applications))
	}

	for len(allApplications) < totalElements {
		if pageSize <= 0 {
			break
		}

		offset += pageSize
		resp, err = catalogClient.CatalogServiceListApplicationsWithResponse(ctx, projectName,
			&catapi.CatalogServiceListApplicationsParams{
				Kinds:    getApplicationKinds(cmd),
				OrderBy:  apiOrderBy,
				Filter:   getFlag(cmd, "filter"),
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing applications"); !proceed {
			return err
		}

		if len(resp.JSON200.Applications) == 0 {
			break
		}
		allApplications = append(allApplications, resp.JSON200.Applications...)
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printApplications(cmd, writer, &allApplications, clientOrderBy, &outputFilter, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

func runGetApplicationCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	var appList []catapi.CatalogV3Application
	if len(args) == 2 {
		version := args[1]
		resp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, name, version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			fmt.Sprintf("error getting application %s:%s", name, version)); !proceed {
			return err
		}
		appList = append(appList, resp.JSON200.Application)
	} else {
		resp, err := catalogClient.CatalogServiceGetApplicationVersionsWithResponse(ctx, projectName, name,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			fmt.Sprintf("error getting application %s versions", name)); !proceed {
			return err
		}
		appList = append(appList, resp.JSON200.Application...)
		if len(appList) == 0 {
			return fmt.Errorf("no versions of application %s found", name)
		}
	}
	if err := printApplications(cmd, writer, &appList, nil, nil, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

func runSetApplicationCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]

	gresp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("application %s:%s not found", name, version)); err != nil {
		return err
	}

	application := gresp.JSON200.Application

	resp, err := catalogClient.CatalogServiceUpdateApplicationWithResponse(ctx, projectName, name, version,
		catapi.CatalogServiceUpdateApplicationJSONRequestBody{
			Name:               name,
			Version:            version,
			Kind:               getApplicationKind(cmd, application.Kind),
			DisplayName:        getFlagOrDefault(cmd, "display-name", application.DisplayName),
			Description:        getFlagOrDefault(cmd, "description", application.Description),
			ChartName:          *getFlagOrDefault(cmd, "chart-name", &application.ChartName),
			ChartVersion:       *getFlagOrDefault(cmd, "chart-version", &application.ChartVersion),
			HelmRegistryName:   *getFlagOrDefault(cmd, "chart-registry", &application.HelmRegistryName),
			ImageRegistryName:  getFlagOrDefault(cmd, "image-registry", application.ImageRegistryName),
			DefaultProfileName: getFlagOrDefault(cmd, "default-profile", application.DefaultProfileName),
			Profiles:           application.Profiles,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while updating application %s:%s", name, version)); err != nil {
		return err
	}
	fmt.Printf("Application '%s:%s' updated successfully\n", name, version)
	return nil
}

func runDeleteApplicationCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]

	// If version was specified, delete just that version
	if len(args) == 2 {
		version := args[1]
		resp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, name, version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err = checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("application %s:%s not found", name, version)); err != nil {
			return err
		}
		deleteResp, err := catalogClient.CatalogServiceDeleteApplicationWithResponse(ctx, projectName, name, version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(deleteResp.HTTPResponse, deleteResp.Body, fmt.Sprintf("error deleting application %s:%s", name, version)); err != nil {
			return err
		}
		fmt.Printf("Application '%s:%s' deleted successfully\n", name, version)
		return nil
	}

	// Otherwise delete all versions
	resp, err := catalogClient.CatalogServiceGetApplicationVersionsWithResponse(ctx, projectName, name,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error getting application versions %s", name)); err != nil {
		return err
	}
	if len(resp.JSON200.Application) == 0 {
		return fmt.Errorf("application %s has no versions", name)
	}

	for _, app := range resp.JSON200.Application {
		deleteResp, err := catalogClient.CatalogServiceDeleteApplicationWithResponse(ctx, projectName, app.Name, app.Version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(deleteResp.HTTPResponse, deleteResp.Body, fmt.Sprintf("error deleting application %s:%s", app.Name, app.Version)); err != nil {
			return err
		}
	}
	fmt.Printf("All versions of application '%s' deleted successfully\n", name)
	return nil
}

func printApplicationEvent(writer io.Writer, _ string, payload []byte, verbose bool) error {
	var item catapi.CatalogV3Application
	if err := json.Unmarshal(payload, &item); err != nil {
		return err
	}
	if !verbose {
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", item.Name,
			valueOrNone(item.DisplayName), item.Version, applicationKind2String(item.Kind),
			item.ChartName, item.ChartVersion, item.HelmRegistryName, valueOrNone(item.DefaultProfileName))
	} else {
		_, _ = fmt.Fprintf(writer, "Name: %s\n", item.Name)
		_, _ = fmt.Fprintf(writer, "Display Name: %s\n", valueOrNone(item.DisplayName))
		_, _ = fmt.Fprintf(writer, "Description: %s\n", valueOrNone(item.Description))
		_, _ = fmt.Fprintf(writer, "Version: %s\n", item.Version)
		_, _ = fmt.Fprintf(writer, "Kind: %s\n", applicationKind2String(item.Kind))
		_, _ = fmt.Fprintf(writer, "Helm Registry Name: %s\n", item.HelmRegistryName)
		_, _ = fmt.Fprintf(writer, "Image Registry Name: %s\n", valueOrNone(item.ImageRegistryName))
		_, _ = fmt.Fprintf(writer, "Chart Name: %s\n", item.ChartName)
		_, _ = fmt.Fprintf(writer, "Chart Version: %s\n", item.ChartVersion)
		creatTime := ""
		if item.CreateTime != nil {
			creatTime = item.CreateTime.Format(timeLayout)
		}
		_, _ = fmt.Fprintf(writer, "Create Time: %s\n", creatTime)
		updateTime := ""
		if item.UpdateTime != nil {
			updateTime = item.UpdateTime.Format(timeLayout)
		}
		_, _ = fmt.Fprintf(writer, "Update Time: %s\n", updateTime)
		profiles := make([]string, 0)
		if item.Profiles != nil {
			profiles = make([]string, 0, len(*item.Profiles))
			for _, p := range *item.Profiles {
				profiles = append(profiles, p.Name)
			}
		}
		_, _ = fmt.Fprintf(writer, "Profiles: %v\n", profiles)
		_, _ = fmt.Fprintf(writer, "Default Profile: %s\n\n", valueOrNone(item.DefaultProfileName))
	}
	return nil
}
