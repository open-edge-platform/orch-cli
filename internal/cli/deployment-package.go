// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-edge-platform/cli/internal/validator"
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
)

const (
	DEFAULT_DEPLOYMENT_PACKAGE_FORMAT         = "table{{.Name}}\t{{.DisplayName}}\t{{.Version}}\t{{.Kind}}\t{{.DefaultProfileName}}\t{{.IsDeployed}}\t{{len .ApplicationReferences}}"
	DEFAULT_DEPLOYMENT_PACKAGE_INSPECT_FORMAT = `Name: {{.Name}}
Display Name: {{str .DisplayName}}
Description: {{str .Description}}
Version: {{.Version}}
Kind: {{.Kind}}
Is Deployed: {{.IsDeployed}}
Applications:
	{{- range .ApplicationReferences}}
  {{.Name}}:{{.Version}}
	{{- end}}
Application Dependencies:
	{{- range deref .ApplicationDependencies}}
	{{.Name}}:{{.Requires}}
	{{- end}}
Profiles:
	{{- range deref .Profiles}}
  {{.Name}}
	{{- end}}
Default Profile: {{str .DefaultProfileName}}
Default Namespaces:
	{{- range $app, $ns := deref .DefaultNamespaces}}
  {{$app}}:{{$ns}}
	{{- end}}
Extensions:
	{{- range .Extensions}}
  {{.Name}}:{{.Version}}
	{{- end}}
Artifacts:
	{{- range .Artifacts}}
  {{.Name}} ({{.Type}})
	{{- end}}
Create Time: {{.CreateTime}}
Update Time: {{.UpdateTime}}
`
	DEPLOYMENT_PACKAGE_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_DEPLOYMENT_PACKAGE_OUTPUT_TEMPLATE"
)

func getCreateDeploymentPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package <name> <version> [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "Create a deployment package",
		Args:    cobra.ExactArgs(2),
		Example: "orch-cli create deployment-package my-package 1.0.0 --project sample-project --application-reference app1:2.1.0 --application-reference app2:3.17.1 --default-namespace app1=my-namespace --default-profile-name my-profile",
		RunE:    runCreateDeploymentPackageCommand,
	}
	addEntityFlags(cmd, "deployment-package")
	cmd.Flags().StringSlice("application-reference", []string{}, "<name>:<version> constituent application references (required)")
	cmd.Flags().StringToString("application-dependency", map[string]string{},
		"application dependencies expresssed as <app-name>=<required-app-name>,<required-app-name,...")
	cmd.Flags().StringToString("default-namespace", map[string]string{},
		"default namespaces for applications in format '<app-name>=<namespace>'")
	cmd.Flags().String("default-profile-name", "", "default profile name for the deployment package (default: deployment-profile-1)")
	cmd.Flags().String("kind", "normal", "deployment package kind: normal, addon, extension")
	return cmd
}

func getListDeploymentPackagesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-packages [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "List all deployment packages",
		Example: "orch-cli list deployment-packages --project some-project",
		RunE:    runListDeploymentPackagesCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "deployment package")
	cmd.Flags().StringSlice("kind", []string{}, "deployment package kind: normal, addon, extension")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getGetDeploymentPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package <name> [<version>] [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "Get a deployment package",
		Args:    cobra.RangeArgs(1, 2),
		Example: "orch-cli get deployment-package my-package --project some-project",
		RunE:    runGetDeploymentPackageCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getSetDeploymentPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package <name> <version> [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "Update a deployment package",
		Args:    cobra.ExactArgs(2),
		Example: "orch-cli set deployment-package my-package 1.0.0 --project sample-project --application-reference app1:2.1.0 --application-reference app2:3.17.1 --application-reference app3:1.1.1",
		RunE:    runSetDeploymentPackageCommand,
	}
	addEntityFlags(cmd, "deployment-package")
	cmd.Flags().String("thumbnail-name", "", "name of the application thumbnail artifact")
	cmd.Flags().String("icon-name", "", "name of the application icon artifact")
	cmd.Flags().String("default-profile", "", "default deployment profile")
	cmd.Flags().StringSlice("application-reference", []string{}, "<name>:<version> constituent application references")
	cmd.Flags().StringToString("application-dependency", map[string]string{},
		"application dependencies expresssed as <app-name>=<required-app-name>,<required-app-name,...")
	cmd.Flags().StringToString("default-namespace", map[string]string{},
		"default namespaces for applications in format '<app-name>=<namespace>'")
	cmd.Flags().Bool("deployed", false, "mark deployment package as deployed or not")
	return cmd
}

func getDeleteDeploymentPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package <name> [<version>] [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "Delete a deployment package",
		Args:    cobra.RangeArgs(1, 2),
		Example: "orch-cli delete deployment-package my-package --project some-project",
		RunE:    runDeleteDeploymentPackageCommand,
	}
	return cmd
}

func getExportDeploymentPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package <name> [<version>] [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "Export a deployment package as a tarball",
		Args:    cobra.ExactArgs(2),
		Example: "orch-cli export deployment-package my-package 0.1.1 --project some-project",
		RunE:    runExportDeploymentPackageCommand,
	}
	cmd.Flags().StringP("output-file", "o", "", "Override output filename")
	return cmd
}

func getDeploymentPackageOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_DEPLOYMENT_PACKAGE_INSPECT_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_DEPLOYMENT_PACKAGE_FORMAT, DEPLOYMENT_PACKAGE_OUTPUT_TEMPLATE_ENVVAR)
}

func printDeploymentPackages(cmd *cobra.Command, writer io.Writer, caList *[]catapi.CatalogV3DeploymentPackage, orderBy *string, outputFilter *string, verbose bool) error {
	outputFormat, err := getDeploymentPackageOutputFormat(cmd, verbose)
	if err != nil {
		return err
	}

	outputType, _ := cmd.Flags().GetString("output-type")
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
		Data:      *caList,
	}

	GenerateOutput(writer, &result)
	return nil
}

// Produces an application reference from the specified <name>:<version> string
func parseApplicationReference(refSpec string) (*catapi.CatalogV3ApplicationReference, error) {
	refFields := strings.SplitN(refSpec, ":", 2)
	if len(refFields) != 2 {
		return nil, fmt.Errorf("application reference must be in form of <name>:<version>")
	}

	// Validate version format
	if err := validator.ValidateVersion(refFields[1]); err != nil {
		return nil, fmt.Errorf("invalid version in application reference '%s': %w", refSpec, err)
	}

	return &catapi.CatalogV3ApplicationReference{Name: refFields[0], Version: refFields[1]}, nil
}

func runCreateDeploymentPackageCommand(cmd *cobra.Command, args []string) error {
	applicationName := args[0]
	applicationVersion := args[1]

	// Validate version format
	if err := validator.ValidateVersion(applicationVersion); err != nil {
		return err
	}

	// Validate required flags
	appRefs, _ := cmd.Flags().GetStringSlice("application-reference")
	if len(appRefs) == 0 {
		return fmt.Errorf("--application-reference is required (at least one application must be referenced)")
	}

	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}
	displayName, description, err := getEntityFlags(cmd)
	if err != nil {
		return err
	}

	// Collect application references and validate they exist
	applicationReferences := make([]catapi.CatalogV3ApplicationReference, 0)
	appDefaultProfiles := make(map[string]string)
	for _, refSpec := range appRefs {
		ref, err := parseApplicationReference(refSpec)
		if err != nil {
			return err
		}

		// Verify the application exists
		appResp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, ref.Name, ref.Version, auth.AddAuthHeader)
		if err != nil {
			return fmt.Errorf("failed to verify application %s:%s exists: %w", ref.Name, ref.Version, err)
		}
		if appResp.StatusCode() != 200 {
			return fmt.Errorf("application %s:%s does not exist. Please create the application before referencing it in the deployment package", ref.Name, ref.Version)
		}

		// Store the application's default profile name (fallback to "default" if not set)
		if appResp.JSON200 != nil && appResp.JSON200.Application.DefaultProfileName != nil {
			appDefaultProfiles[ref.Name] = *appResp.JSON200.Application.DefaultProfileName
		} else {
			appDefaultProfiles[ref.Name] = "default"
		}

		applicationReferences = append(applicationReferences, *ref)
	}

	// Collect application dependencies
	applicationDependencies := make([]catapi.CatalogV3ApplicationDependency, 0)
	appDeps, _ := cmd.Flags().GetStringToString("application-dependency")
	if len(appDeps) > 0 {
		for app, deps := range appDeps {
			for _, name := range strings.Split(deps, ",") {
				applicationDependencies = append(applicationDependencies, catapi.CatalogV3ApplicationDependency{Name: app, Requires: name})
			}
		}
	}

	// Collect default namespaces
	defaultNamespaces, _ := cmd.Flags().GetStringToString("default-namespace")
	var defaultNamespacesPtr *map[string]string
	if len(defaultNamespaces) > 0 {
		defaultNamespacesPtr = &defaultNamespaces
	}

	// Set up default profile name - use "deployment-profile-1" to match UI behavior
	defaultProfileName, _ := cmd.Flags().GetString("default-profile-name")
	if defaultProfileName == "" {
		defaultProfileName = "deployment-profile-1"
	}

	// Create an initial deployment profile to match UI behavior
	initialProfile := catapi.CatalogV3DeploymentProfile{
		Name:                defaultProfileName,
		ApplicationProfiles: appDefaultProfiles,
	}
	initialProfiles := []catapi.CatalogV3DeploymentProfile{initialProfile}

	defaultKind := catapi.KINDNORMAL

	resp, err := catalogClient.CatalogServiceCreateDeploymentPackageWithResponse(ctx, projectName,
		catapi.CatalogServiceCreateDeploymentPackageJSONRequestBody{
			Name:                    applicationName,
			Version:                 applicationVersion,
			Kind:                    getDeploymentPackageKind(cmd, &defaultKind),
			DisplayName:             &displayName,
			Description:             &description,
			ApplicationReferences:   applicationReferences,
			ApplicationDependencies: &applicationDependencies,
			DefaultNamespaces:       defaultNamespacesPtr,
			DefaultProfileName:      &defaultProfileName,
			Profiles:                &initialProfiles,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating deployment package %s", applicationName)); err != nil {
		return err
	}
	fmt.Printf("Deployment package '%s:%s' created successfully\n", applicationName, applicationVersion)
	return nil
}

func deploymentPackageKind2String(kind *catapi.CatalogV3Kind) string {
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

func string2DeploymentPackageKind(kind string) catapi.CatalogV3Kind {
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

func getDeploymentPackageKind(cmd *cobra.Command, def *catapi.CatalogV3Kind) *catapi.CatalogV3Kind {
	dv := deploymentPackageKind2String(def)
	kind := string2DeploymentPackageKind(*getFlagOrDefault(cmd, "kind", &dv))
	return &kind
}

func getDeploymentPackageKinds(cmd *cobra.Command) *[]catapi.CatalogV3Kind {
	kinds, _ := cmd.Flags().GetStringSlice("kind")
	if len(kinds) == 0 {
		return nil
	}
	list := make([]catapi.CatalogV3Kind, 0, len(kinds))
	for _, k := range kinds {
		list = append(list, string2DeploymentPackageKind(k))
	}
	return &list
}

func getValidatedDeploymentPackageOrderBy(
	ctx context.Context,
	cmd *cobra.Command,
	catalogClient catapi.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return nil, err
	}

	outputType, _ := cmd.Flags().GetString("output-type")

	// For table format (default), use client-side sorting which supports any field in the model
	if outputType == "table" {
		return normalizeOrderByForClientSorting(raw, catapi.CatalogV3DeploymentPackage{})
	}

	// For JSON/YAML, use API ordering (only API-supported fields)
	return normalizeOrderByWithAPIProbe(raw, "deployment-packages", catapi.CatalogV3DeploymentPackage{}, func(orderBy string) (bool, error) {
		pageSize := int32(1)
		offset := int32(0)
		// Validate ordering in isolation. Reusing the caller's --filter here can turn
		// filter errors into misleading "invalid --order-by field" errors.
		resp, err := catalogClient.CatalogServiceListDeploymentPackagesWithResponse(ctx, projectName,
			&catapi.CatalogServiceListDeploymentPackagesParams{
				Kinds:    getDeploymentPackageKinds(cmd),
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
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating deployment package order-by"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func runListDeploymentPackagesCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	validatedOrderBy, err := getValidatedDeploymentPackageOrderBy(ctx, cmd, catalogClient, projectName)
	if err != nil {
		return err
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	apiOrderBy := validatedOrderBy
	if outputType == "table" {
		// Table output sorts locally via GenerateOutput(CommandResult.OrderBy).
		apiOrderBy = nil
	}

	pageSize, offset, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	// Preserve explicit pagination requests as single-page results.
	if cmd.Flags().Changed("page-size") || cmd.Flags().Changed("offset") {
		resp, err := catalogClient.CatalogServiceListDeploymentPackagesWithResponse(ctx, projectName,
			&catapi.CatalogServiceListDeploymentPackagesParams{
				Kinds:    getDeploymentPackageKinds(cmd),
				OrderBy:  apiOrderBy,
				Filter:   getFlag(cmd, "filter"),
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing deployment packages"); !proceed {
			return err
		}

		outputFilter, _ := cmd.Flags().GetString("output-filter")
		if err := printDeploymentPackages(cmd, writer, &resp.JSON200.DeploymentPackages, validatedOrderBy, &outputFilter, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	allDeploymentPackages := make([]catapi.CatalogV3DeploymentPackage, 0)

	resp, err := catalogClient.CatalogServiceListDeploymentPackagesWithResponse(ctx, projectName,
		&catapi.CatalogServiceListDeploymentPackagesParams{
			Kinds:    getDeploymentPackageKinds(cmd),
			OrderBy:  apiOrderBy,
			Filter:   getFlag(cmd, "filter"),
			PageSize: &pageSize,
			Offset:   &offset,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
		"error listing deployment packages"); !proceed {
		return err
	}

	allDeploymentPackages = append(allDeploymentPackages, resp.JSON200.DeploymentPackages...)
	totalElements := int(resp.JSON200.TotalElements)

	// When page size is omitted (0), derive increment from the first page length.
	if pageSize <= 0 {
		pageSize = int32(len(resp.JSON200.DeploymentPackages))
	}

	for len(allDeploymentPackages) < totalElements {
		if pageSize <= 0 {
			break
		}

		offset += pageSize
		resp, err = catalogClient.CatalogServiceListDeploymentPackagesWithResponse(ctx, projectName,
			&catapi.CatalogServiceListDeploymentPackagesParams{
				Kinds:    getDeploymentPackageKinds(cmd),
				OrderBy:  apiOrderBy,
				Filter:   getFlag(cmd, "filter"),
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing deployment packages"); !proceed {
			return err
		}

		if len(resp.JSON200.DeploymentPackages) == 0 {
			break
		}
		allDeploymentPackages = append(allDeploymentPackages, resp.JSON200.DeploymentPackages...)
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printDeploymentPackages(cmd, writer, &allDeploymentPackages, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

func runGetDeploymentPackageCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]

	var deploymentPkgs []catapi.CatalogV3DeploymentPackage
	if len(args) == 2 {
		version := args[1]
		resp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, name, version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, "",
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
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, "",
			fmt.Sprintf("error getting deployment package %s versions", name)); !proceed {
			return err
		}
		deploymentPkgs = append(deploymentPkgs, resp.JSON200.DeploymentPackages...)
		if len(deploymentPkgs) == 0 {
			return fmt.Errorf("no versions of deployment package %s found", name)
		}
	}
	if err := printDeploymentPackages(cmd, writer, &deploymentPkgs, nil, nil, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

func runSetDeploymentPackageCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
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
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("deployment package %s:%s not found", name, version)); err != nil {
		return err
	}

	deploymentPackage := gresp.JSON200.DeploymentPackage
	applicationReferences := deploymentPackage.ApplicationReferences
	applicationDependencies := *deploymentPackage.ApplicationDependencies

	// Collect new application references; if any were specified all must be specified
	newApplicationReferences, _ := cmd.Flags().GetStringSlice("application-reference")
	if len(newApplicationReferences) > 0 {
		applicationReferences = make([]catapi.CatalogV3ApplicationReference, 0)
		for _, refSpec := range newApplicationReferences {
			ref, err := parseApplicationReference(refSpec)
			if err != nil {
				return err
			}

			// Verify the application exists
			appResp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, ref.Name, ref.Version, auth.AddAuthHeader)
			if err != nil {
				return fmt.Errorf("failed to verify application %s:%s exists: %w", ref.Name, ref.Version, err)
			}
			if appResp.StatusCode() != 200 {
				return fmt.Errorf("application %s:%s does not exist. Please create the application before referencing it in the deployment package", ref.Name, ref.Version)
			}

			applicationReferences = append(applicationReferences, *ref)
		}
	}

	// Collect new application dependencies; if any were specified all must be specified
	newApplicationDependencies, _ := cmd.Flags().GetStringToString("application-dependency")
	if len(newApplicationDependencies) > 0 {
		applicationDependencies = make([]catapi.CatalogV3ApplicationDependency, 0)
		for app, deps := range newApplicationDependencies {
			if len(deps) > 0 {
				for _, name := range strings.Split(deps, ",") {
					applicationDependencies = append(applicationDependencies, catapi.CatalogV3ApplicationDependency{Name: app, Requires: name})
				}
			}
		}
	}

	// Collect default namespaces; merge with existing if any
	defaultNamespaces := deploymentPackage.DefaultNamespaces
	newDefaultNamespaces, _ := cmd.Flags().GetStringToString("default-namespace")
	if len(newDefaultNamespaces) > 0 {
		if defaultNamespaces == nil {
			defaultNamespaces = &newDefaultNamespaces
		} else {
			for k, v := range newDefaultNamespaces {
				(*defaultNamespaces)[k] = v
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
			Profiles:                deploymentPackage.Profiles,
			ApplicationReferences:   applicationReferences,
			ApplicationDependencies: &applicationDependencies,
			DefaultNamespaces:       defaultNamespaces,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while updating deployment package %s:%s", name, version)); err != nil {
		return err
	}
	fmt.Printf("Deployment package '%s:%s' updated successfully\n", name, version)
	return nil
}

func runDeleteDeploymentPackageCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
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
		if err = checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("deployment package %s:%s not found", name, version)); err != nil {
			return err
		}
		deleteResp, err := catalogClient.CatalogServiceDeleteDeploymentPackageWithResponse(ctx, projectName, name, version,
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(deleteResp.HTTPResponse, deleteResp.Body, fmt.Sprintf("error deleting deployment package %s:%s", name, version)); err != nil {
			return err
		}
		fmt.Printf("Deployment package '%s:%s' deleted successfully\n", name, version)
		return nil
	}

	// Otherwise delete all versions
	resp, err := catalogClient.CatalogServiceGetDeploymentPackageVersionsWithResponse(ctx, projectName, name,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error getting deployment package versions %s", name)); err != nil {
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
		if err := checkResponse(deleteResp.HTTPResponse, deleteResp.Body, fmt.Sprintf("error deleting deployment package %s:%s",
			pkg.Name, pkg.Version)); err != nil {
			return err
		}
	}
	fmt.Printf("All versions of deployment package '%s' deleted successfully\n", name)
	return nil
}

func runExportDeploymentPackageCommand(cmd *cobra.Command, args []string) error {
	ctx, utilClient, projectName, err := CatalogUtilitiesFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]

	resp, err := utilClient.CatalogServiceDownloadDeploymentPackageWithResponse(ctx, projectName, name, version, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error downloading deployment package %s:%s",
		name, version)); err != nil {
		return err
	}

	filename := ""
	if fileNameFlag := getFlag(cmd, "output-file"); fileNameFlag != nil {
		filename = *fileNameFlag
	}

	// No filename given on command line, so try to get it from the API response
	if filename == "" {
		// NOTE: This is untested in production due to an issue in nexus-api-gw that is
		// stripping the Content-Disposition headers.
		contentDisposition := resp.HTTPResponse.Header.Get("Content-Disposition")
		if contentDisposition != "" {
			_, params, err := mime.ParseMediaType(contentDisposition)
			if err == nil {
				if fname, ok := params["filename"]; ok {
					filename = fname
				}
			}
		}

		// We got this from the server, so there should be no path.
		// But just in case... ensure filename is not a path
		if filename != "" {
			filename = filepath.Base(filename)
		}
	}

	// If after all that, we still don't have a filename, make up a default one
	if filename == "" {
		filename = fmt.Sprintf("%s-%s.tar.gz", name, version)
	}

	outFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer outFile.Close()

	if _, err := outFile.Write(resp.Body); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filename, err)
	}
	fmt.Printf("Deployment package exported to %s\n", filename)

	return nil
}
