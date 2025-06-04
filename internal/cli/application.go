// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
)

var (
	applicationAliases       = []string{"app"}
	deploymentPackageAliases = []string{"package", "bundle", "pkg"}
	deploymentProfileAliases = []string{"package-profile", "deployment-profile", "bundle-profile"}
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
	_ = cmd.MarkFlagRequired("chart-name")
	cmd.Flags().String("chart-version", "", "Helm chart version (required)")
	_ = cmd.MarkFlagRequired("chart-version")
	cmd.Flags().String("chart-registry", "", "Helm chart registry (required)")
	_ = cmd.MarkFlagRequired("chart-registry")
	cmd.Flags().String("image-registry", "", "image registry")
	cmd.Flags().String("kind", "normal", "application kind: normal, addon, extension")
	return cmd
}

func getListApplicationsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "applications [flags]",
		Aliases: []string{"apps", "applications"},
		Short:   "List all applications",
		Example: "orch-cli list applications --project some-project",
		RunE:    runListApplicationsCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "application")
	cmd.Flags().StringSlice("kind", []string{}, "application kind: normal, addon, extension")
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
	return cmd
}

func getSetApplicationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "application <name> <version> [flags]",
		Aliases: applicationAliases,
		Short:   "Update an application",
		Args:    cobra.ExactArgs(2),
		Example: "orch-cli set application my-app 1.0.0 --chart-name my-chart --chart-version 1.0.0 --chart-registry my-registry --project some-project",
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

var applicationHeader = fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
	"Name", "Display Name", "Version", "Kind", "Chart Name", "Chart Version", "Helm Registry Name", "Default Profile")

func printApplications(writer io.Writer, appList *[]catapi.Application, verbose bool) {
	for _, app := range *appList {
		if !verbose {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", app.Name,
				valueOrNone(app.DisplayName), app.Version, applicationKind2String(app.Kind),
				app.ChartName, app.ChartVersion, app.HelmRegistryName, valueOrNone(app.DefaultProfileName))
		} else {
			_, _ = fmt.Fprintf(writer, "Name: %s\n", app.Name)
			_, _ = fmt.Fprintf(writer, "Display Name: %s\n", valueOrNone(app.DisplayName))
			_, _ = fmt.Fprintf(writer, "Description: %s\n", valueOrNone(app.Description))
			_, _ = fmt.Fprintf(writer, "Version: %s\n", app.Version)
			_, _ = fmt.Fprintf(writer, "Kind: %s\n", applicationKind2String(app.Kind))
			_, _ = fmt.Fprintf(writer, "Helm Registry Name: %s\n", app.HelmRegistryName)
			_, _ = fmt.Fprintf(writer, "Image Registry Name: %s\n", valueOrNone(app.ImageRegistryName))
			_, _ = fmt.Fprintf(writer, "Chart Name: %s\n", app.ChartName)
			_, _ = fmt.Fprintf(writer, "Chart Version: %s\n", app.ChartVersion)
			_, _ = fmt.Fprintf(writer, "Create Time: %s\n", app.CreateTime.Format(timeLayout))
			_, _ = fmt.Fprintf(writer, "Update Time: %s\n", app.UpdateTime.Format(timeLayout))

			profiles := make([]string, 0, len(*app.Profiles))
			for _, p := range *app.Profiles {
				profiles = append(profiles, p.Name)
			}
			_, _ = fmt.Fprintf(writer, "Profiles: %v\n", profiles)
			_, _ = fmt.Fprintf(writer, "Default Profile: %s\n\n", valueOrNone(app.DefaultProfileName))
		}
	}
}

func runCreateApplicationCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}
	displayName, description, err := getEntityFlags(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]
	defaultKind := catapi.ApplicationKindKINDNORMAL

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
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating application %s", name))
}

func applicationKind2String(kind *catapi.ApplicationKind) string {
	if kind == nil {
		return "normal"
	}
	switch *kind {
	case catapi.ApplicationKindKINDNORMAL:
		return "normal"
	case catapi.ApplicationKindKINDADDON:
		return "addon"
	case catapi.ApplicationKindKINDEXTENSION:
		return "extension"
	}
	return "normal"
}

func string2ApplicationKind(kind string) catapi.ApplicationKind {
	switch kind {
	case "normal":
		return catapi.ApplicationKindKINDNORMAL
	case "addon":
		return catapi.ApplicationKindKINDADDON
	case "extension":
		return catapi.ApplicationKindKINDEXTENSION
	}
	return catapi.ApplicationKindKINDNORMAL
}

func getApplicationKind(cmd *cobra.Command, def *catapi.ApplicationKind) *catapi.ApplicationKind {
	dv := applicationKind2String(def)
	kind := string2ApplicationKind(*getFlagOrDefault(cmd, "kind", &dv))
	return &kind
}

func getApplicationKinds(cmd *cobra.Command) *[]catapi.CatalogServiceListApplicationsParamsKinds {
	kinds, _ := cmd.Flags().GetStringSlice("kind")
	if len(kinds) == 0 {
		return nil
	}
	list := make([]catapi.CatalogServiceListApplicationsParamsKinds, 0, len(kinds))
	for _, k := range kinds {
		list = append(list, catapi.CatalogServiceListApplicationsParamsKinds(string2ApplicationKind(k)))
	}
	return &list
}

func runListApplicationsCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}

	pageSize, offset, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	resp, err := catalogClient.CatalogServiceListApplicationsWithResponse(ctx, projectName,
		&catapi.CatalogServiceListApplicationsParams{
			Kinds:    getApplicationKinds(cmd),
			OrderBy:  getFlag(cmd, "order-by"),
			PageSize: &pageSize,
			Offset:   &offset,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, applicationHeader,
		"error listing applications"); !proceed {
		return err
	}
	printApplications(writer, &resp.JSON200.Applications, verbose)
	return writer.Flush()
}

func runGetApplicationCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	var appList []catapi.Application
	if len(args) == 2 {
		version := args[1]
		resp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, name, version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, applicationHeader,
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
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, applicationHeader,
			fmt.Sprintf("error getting application %s versions", name)); !proceed {
			return err
		}
		appList = append(appList, resp.JSON200.Application...)
		if len(appList) == 0 {
			return fmt.Errorf("no versions of application %s found", name)
		}
	}
	printApplications(writer, &appList, verbose)
	return writer.Flush()
}

func runSetApplicationCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
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
	if err = checkResponse(gresp.HTTPResponse, fmt.Sprintf("application %s:%s not found", name, version)); err != nil {
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
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while updating application %s:%s", name, version))
}

func runDeleteApplicationCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
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
		if err = checkResponse(resp.HTTPResponse, fmt.Sprintf("application %s:%s not found", name, version)); err != nil {
			return err
		}
		deleteResp, err := catalogClient.CatalogServiceDeleteApplicationWithResponse(ctx, projectName, name, version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		return checkResponse(deleteResp.HTTPResponse, fmt.Sprintf("error deleting application %s:%s", name, version))
	}

	// Otherwise delete all versions
	resp, err := catalogClient.CatalogServiceGetApplicationVersionsWithResponse(ctx, projectName, name,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(resp.HTTPResponse, fmt.Sprintf("error getting application versions %s", name)); err != nil {
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
		if err := checkResponse(deleteResp.HTTPResponse, fmt.Sprintf("error deleting application %s:%s", app.Name, app.Version)); err != nil {
			return err
		}
	}
	return nil
}

func printApplicationEvent(writer io.Writer, _ string, payload []byte, verbose bool) error {
	var item catapi.Application
	if err := json.Unmarshal(payload, &item); err != nil {
		return err
	}
	printApplications(writer, &[]catapi.Application{item}, verbose)
	return nil
}
