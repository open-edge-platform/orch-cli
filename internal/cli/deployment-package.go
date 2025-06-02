// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
)

func getCreateDeploymentPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package <name> <version> [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "Create a deployment package",
		Args:    cobra.ExactArgs(2),
		RunE:    runCreateDeploymentPackageCommand,
	}
	addEntityFlags(cmd, "deployment-package")
	cmd.Flags().StringSlice("application-reference", []string{}, "<name>:<version>:[<publisher>] constituent application references")
	cmd.Flags().StringToString("application-dependency", map[string]string{},
		"application dependencies expresssed as <app-name>=<required-app-name>,<required-app-name,...")
	cmd.Flags().Bool("visible", true, "mark deployment package as visible or not")
	cmd.Flags().String("kind", "normal", "deployment package kind: normal, addon, extension")
	return cmd
}

func getListDeploymentPackagesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-packages [flags]",
		Aliases: []string{"packages", "bundles", "pkgs"},
		Short:   "Get all deployment packages, optionally filtered by publisher",
		RunE:    runListDeploymentPackagesCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "deployment package")
	cmd.Flags().StringSlice("kind", []string{}, "deployment package kind: normal, addon, extension")
	return cmd
}

func getGetDeploymentPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package <name> [<version>] [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "Get a deployment package",
		Args:    cobra.RangeArgs(1, 2),
		RunE:    runGetDeploymentPackageCommand,
	}
	return cmd
}

func getSetDeploymentPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package <name> <version> [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "Update a deployment package",
		Args:    cobra.ExactArgs(2),
		RunE:    runSetDeploymentPackageCommand,
	}
	addEntityFlags(cmd, "deployment-package")
	cmd.Flags().String("thumbnail-name", "", "name of the application thumbnail artifact")
	cmd.Flags().String("icon-name", "", "name of the application icon artifact")
	cmd.Flags().String("default-profile", "", "default deployment profile")
	cmd.Flags().StringSlice("application-reference", []string{}, "<name>:<version>:[<publisher>] constituent application references")
	cmd.Flags().StringToString("application-dependency", map[string]string{},
		"application dependencies expresssed as <app-name>=<required-app-name>,<required-app-name,...")
	cmd.Flags().Bool("visible", true, "mark deployment package as visible or not")
	cmd.Flags().Bool("deployed", false, "mark deployment package as deployed or not")
	return cmd
}

func getDeleteDeploymentPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package <name> [<version>] [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "Delete a deployment package",
		Args:    cobra.RangeArgs(1, 2),
		RunE:    runDeleteDeploymentPackageCommand,
	}
	return cmd
}

var deploymentPackageHeader = fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
	"Name", "Display Name", "Version", "Kind", "Default Profile", "Is Deployed", "Is Visible", "Application Count")

func printDeploymentPackages(writer io.Writer, caList *[]catapi.DeploymentPackage, verbose bool) {
	for _, ca := range *caList {
		if !verbose {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%t\t%t\t%d\n", ca.Name,
				valueOrNone(ca.DisplayName), ca.Version, deploymentPackageKind2String(ca.Kind),
				valueOrNone(ca.DefaultProfileName), safeBool(ca.IsDeployed), safeBool(ca.IsVisible),
				len(ca.ApplicationReferences))
		} else {
			_, _ = fmt.Fprintf(writer, "Name: %s\n", ca.Name)
			_, _ = fmt.Fprintf(writer, "Display Name: %s\n", valueOrNone(ca.DisplayName))
			_, _ = fmt.Fprintf(writer, "Description: %s\n", valueOrNone(ca.Description))
			_, _ = fmt.Fprintf(writer, "Version: %s\n", ca.Version)
			_, _ = fmt.Fprintf(writer, "Kind: %s\n", deploymentPackageKind2String(ca.Kind))
			_, _ = fmt.Fprintf(writer, "Is Deployed: %t\n", safeBool(ca.IsDeployed))
			_, _ = fmt.Fprintf(writer, "Is Visible: %t\n", safeBool(ca.IsVisible))

			refs := make([]string, 0, len(ca.ApplicationReferences))
			for _, ref := range ca.ApplicationReferences {
				refs = append(refs, fmt.Sprintf("%s:%s", ref.Name, ref.Version))
			}
			_, _ = fmt.Fprintf(writer, "Applications: %v\n", refs)

			deps := make([]string, 0, len(*ca.ApplicationDependencies))
			for _, dep := range *ca.ApplicationDependencies {
				deps = append(deps, fmt.Sprintf("%s->%s", dep.Name, dep.Requires))
			}
			_, _ = fmt.Fprintf(writer, "Application Dependencies: %v\n", deps)

			profiles := make([]string, 0, len(*ca.Profiles))
			for _, p := range *ca.Profiles {
				profiles = append(profiles, p.Name)
			}
			_, _ = fmt.Fprintf(writer, "Profiles: %v\n", profiles)
			_, _ = fmt.Fprintf(writer, "Default Profile: %s\n", *ca.DefaultProfileName)

			extensions := make([]string, 0, len(ca.Extensions))
			for _, ext := range ca.Extensions {
				extensions = append(extensions, ext.Name)
			}
			_, _ = fmt.Fprintf(writer, "Extensions: %v\n", extensions)

			artifacts := make([]string, 0, len(ca.Artifacts))
			for _, ext := range ca.Artifacts {
				artifacts = append(artifacts, fmt.Sprintf("%s:%s", ext.Name, ext.Purpose))
			}
			_, _ = fmt.Fprintf(writer, "Artifacts: %v\n", artifacts)

			_, _ = fmt.Fprintf(writer, "Create Time: %s\n", ca.CreateTime.Format(timeLayout))
			_, _ = fmt.Fprintf(writer, "Update Time: %s\n\n", ca.UpdateTime.Format(timeLayout))
		}
	}
}

// Produces an application reference from the specified <name>:<version>[:<publisher>] string
func parseApplicationReference(refSpec string) (*catapi.ApplicationReference, error) {
	refFields := strings.SplitN(refSpec, ":", 3)
	if len(refFields) < 2 || len(refFields) > 3 {
		return nil, fmt.Errorf("application reference must be in form of <name>:<version>[:<publisher>]")
	}
	return &catapi.ApplicationReference{Name: refFields[0], Version: refFields[1]}, nil
}

func runCreateDeploymentPackageCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}
	displayName, description, err := getEntityFlags(cmd)
	if err != nil {
		return err
	}
	applicationName := args[0]
	applicationVersion := args[1]

	// Collect application references
	applicationReferences := make([]catapi.ApplicationReference, 0)
	appRefs, _ := cmd.Flags().GetStringSlice("application-reference")
	if len(appRefs) > 0 {
		for _, refSpec := range appRefs {
			ref, err := parseApplicationReference(refSpec)
			if err != nil {
				return err
			}
			applicationReferences = append(applicationReferences, *ref)
		}
	}

	// Collect application dependencies
	applicationDependencies := make([]catapi.ApplicationDependency, 0)
	appDeps, _ := cmd.Flags().GetStringToString("application-dependency")
	if len(appDeps) > 0 {
		for app, deps := range appDeps {
			for _, name := range strings.Split(deps, ",") {
				applicationDependencies = append(applicationDependencies, catapi.ApplicationDependency{Name: app, Requires: name})
			}
		}
	}

	defaultKind := catapi.DeploymentPackageKindKINDNORMAL
	defaultVisible := true

	resp, err := catalogClient.CatalogServiceCreateDeploymentPackageWithResponse(ctx, projectName,
		catapi.CatalogServiceCreateDeploymentPackageJSONRequestBody{
			Name:                    applicationName,
			Version:                 applicationVersion,
			Kind:                    getDeploymentPackageKind(cmd, &defaultKind),
			DisplayName:             &displayName,
			Description:             &description,
			ApplicationReferences:   applicationReferences,
			ApplicationDependencies: &applicationDependencies,
			IsVisible:               getBoolFlagOrDefault(cmd, "visible", &defaultVisible),
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating deployment package %s", applicationName))
}

func deploymentPackageKind2String(kind *catapi.DeploymentPackageKind) string {
	if kind == nil {
		return "normal"
	}
	switch *kind {
	case catapi.DeploymentPackageKindKINDNORMAL:
		return "normal"
	case catapi.DeploymentPackageKindKINDADDON:
		return "addon"
	case catapi.DeploymentPackageKindKINDEXTENSION:
		return "extension"
	}
	return "normal"
}

func string2DeploymentPackageKind(kind string) catapi.DeploymentPackageKind {
	switch kind {
	case "normal":
		return catapi.DeploymentPackageKindKINDNORMAL
	case "addon":
		return catapi.DeploymentPackageKindKINDADDON
	case "extension":
		return catapi.DeploymentPackageKindKINDEXTENSION
	}
	return catapi.DeploymentPackageKindKINDNORMAL
}

func getDeploymentPackageKind(cmd *cobra.Command, def *catapi.DeploymentPackageKind) *catapi.DeploymentPackageKind {
	dv := deploymentPackageKind2String(def)
	kind := string2DeploymentPackageKind(*getFlagOrDefault(cmd, "kind", &dv))
	return &kind
}

func getDeploymentPackageKinds(cmd *cobra.Command) *[]catapi.CatalogServiceListDeploymentPackagesParamsKinds {
	kinds, _ := cmd.Flags().GetStringSlice("kind")
	if len(kinds) == 0 {
		return nil
	}
	list := make([]catapi.CatalogServiceListDeploymentPackagesParamsKinds, 0, len(kinds))
	for _, k := range kinds {
		list = append(list, catapi.CatalogServiceListDeploymentPackagesParamsKinds(string2DeploymentPackageKind(k)))
	}
	return &list
}

func runListDeploymentPackagesCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}

	pageSize, offset, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	resp, err := catalogClient.CatalogServiceListDeploymentPackagesWithResponse(ctx, projectName,
		&catapi.CatalogServiceListDeploymentPackagesParams{
			Kinds:    getDeploymentPackageKinds(cmd),
			OrderBy:  getFlag(cmd, "order-by"),
			Filter:   getFlag(cmd, "filter"),
			PageSize: &pageSize,
			Offset:   &offset,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, deploymentPackageHeader,
		"error listing deployment packages"); !proceed {
		return err
	}
	printDeploymentPackages(writer, &resp.JSON200.DeploymentPackages, verbose)
	return writer.Flush()
}

func runGetDeploymentPackageCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}

	name := args[0]

	var deploymentPkgs []catapi.DeploymentPackage
	if len(args) == 2 {
		version := args[1]
		resp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, name, version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, deploymentPackageHeader,
			fmt.Sprintf("error getting deployment package %s:%s", name, version)); !proceed {
			return err
		}
		deploymentPkgs = append(deploymentPkgs, resp.JSON200.DeploymentPackage)
	} else {
		resp, err := catalogClient.CatalogServiceGetDeploymentPackageVersionsWithResponse(ctx, projectName, name,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, deploymentPackageHeader,
			fmt.Sprintf("error getting deployment package %s versions", name)); !proceed {
			return err
		}
		deploymentPkgs = append(deploymentPkgs, resp.JSON200.DeploymentPackages...)
		if len(deploymentPkgs) == 0 {
			return fmt.Errorf("no versions of deployment package %s found", name)
		}
	}
	printDeploymentPackages(writer, &deploymentPkgs, verbose)
	return writer.Flush()
}

func runSetDeploymentPackageCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}
	name := args[0]
	version := args[1]

	gresp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, fmt.Sprintf("deployment package %s:%s not found", name, version)); err != nil {
		return err
	}

	deploymentPackage := gresp.JSON200.DeploymentPackage
	applicationReferences := deploymentPackage.ApplicationReferences
	applicationDependencies := *deploymentPackage.ApplicationDependencies

	// Collect new application references; if any were specified all must be specified
	newApplicationReferences, _ := cmd.Flags().GetStringSlice("application-reference")
	if len(newApplicationReferences) > 0 {
		applicationReferences = make([]catapi.ApplicationReference, 0)
		for _, refSpec := range newApplicationReferences {
			ref, err := parseApplicationReference(refSpec)
			if err != nil {
				return err
			}
			applicationReferences = append(applicationReferences, *ref)
		}
	}

	// Collect new application dependencies; if any were specified all must be specified
	newApplicationDependencies, _ := cmd.Flags().GetStringToString("application-dependency")
	if len(newApplicationDependencies) > 0 {
		applicationDependencies = make([]catapi.ApplicationDependency, 0)
		for app, deps := range newApplicationDependencies {
			if len(deps) > 0 {
				for _, name := range strings.Split(deps, ",") {
					applicationDependencies = append(applicationDependencies, catapi.ApplicationDependency{Name: app, Requires: name})
				}
			}
		}
	}

	resp, _ := catalogClient.CatalogServiceUpdateDeploymentPackageWithResponse(ctx, projectName, name, version,
		catapi.CatalogServiceUpdateDeploymentPackageJSONRequestBody{
			Name:                    name,
			Version:                 version,
			Kind:                    getDeploymentPackageKind(cmd, deploymentPackage.Kind),
			DisplayName:             getFlagOrDefault(cmd, "display-name", deploymentPackage.DisplayName),
			Description:             getFlagOrDefault(cmd, "description", deploymentPackage.Description),
			DefaultProfileName:      getFlagOrDefault(cmd, "default-profile", deploymentPackage.DefaultProfileName),
			IsDeployed:              getBoolFlagOrDefault(cmd, "deployed", deploymentPackage.IsDeployed),
			IsVisible:               getBoolFlagOrDefault(cmd, "visible", deploymentPackage.IsVisible),
			Profiles:                deploymentPackage.Profiles,
			ApplicationReferences:   applicationReferences,
			ApplicationDependencies: &applicationDependencies,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while updating deployment package %s:%s", name, version))
}

func runDeleteDeploymentPackageCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}
	name := args[0]

	// If version was specified, delete just that version
	if len(args) == 2 {
		version := args[1]
		resp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, name, version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err = checkResponse(resp.HTTPResponse, fmt.Sprintf("deployment package %s:%s not found", name, version)); err != nil {
			return err
		}
		deleteResp, err := catalogClient.CatalogServiceDeleteDeploymentPackageWithResponse(ctx, projectName, name, version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		return checkResponse(deleteResp.HTTPResponse, fmt.Sprintf("error deleting deployment package %s:%s", name, version))
	}

	// Otherwise delete all versions
	resp, err := catalogClient.CatalogServiceGetDeploymentPackageVersionsWithResponse(ctx, projectName, name,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(resp.HTTPResponse, fmt.Sprintf("error getting deployment package versions %s", name)); err != nil {
		return err
	}
	if len(resp.JSON200.DeploymentPackages) == 0 {
		return fmt.Errorf("deployment package %s has no versions", name)
	}

	for _, pkg := range resp.JSON200.DeploymentPackages {
		deleteResp, err := catalogClient.CatalogServiceDeleteDeploymentPackageWithResponse(ctx, projectName, pkg.Name, pkg.Version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(deleteResp.HTTPResponse, fmt.Sprintf("error deleting deployment package %s:%s",
			pkg.Name, pkg.Version)); err != nil {
			return err
		}
	}
	return nil
}

func printDeploymentPackageEvent(writer io.Writer, _ string, payload []byte, verbose bool) error {
	var item catapi.DeploymentPackage
	if err := json.Unmarshal(payload, &item); err != nil {
		return err
	}
	printDeploymentPackages(writer, &[]catapi.DeploymentPackage{item}, verbose)
	return nil
}
