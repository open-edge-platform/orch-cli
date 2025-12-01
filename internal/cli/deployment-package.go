// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-edge-platform/cli/internal/validator"
	"github.com/open-edge-platform/cli/pkg/auth"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	catutilapi "github.com/open-edge-platform/cli/pkg/rest/catalogutilities"
	"github.com/open-edge-platform/orch-library/go/pkg/loader"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func getCreateDeploymentPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package {<name> <version>|<file-path>} [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "Create a deployment package",
		Args:    cobra.RangeArgs(1, 2),
		Example: "orch-cli create deployment-package my-package 1.0.0 --project sample-project --application-reference app1:2.1.0 --application-reference app2:3.17.1\norch-cli create deployment-package my-package.yaml --project sample-project",
		RunE:    runCreateDeploymentPackageCommand,
	}
	addEntityFlags(cmd, "deployment-package")
	cmd.Flags().StringSlice("application-reference", []string{}, "<name>:<version> constituent application references")
	cmd.Flags().StringToString("application-dependency", map[string]string{},
		"application dependencies expresssed as <app-name>=<required-app-name>,<required-app-name,...")
	cmd.Flags().Bool("visible", true, "mark deployment package as visible or not")
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
	return cmd
}

func getSetDeploymentPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package {<name> <version>|<file-path>} [flags]",
		Aliases: deploymentPackageAliases,
		Short:   "Update a deployment package",
		Args:    cobra.RangeArgs(1, 2),
		Example: "orch-cli set deployment-package my-package 1.0.0 --project sample-project --application-reference app1:2.1.0 --application-reference app2:3.17.1 --application-reference app3:1.1.1\norch-cli set deployment-package my-package.yaml --project sample-project",
		RunE:    runSetDeploymentPackageCommand,
	}
	addEntityFlags(cmd, "deployment-package")
	cmd.Flags().String("thumbnail-name", "", "name of the application thumbnail artifact")
	cmd.Flags().String("icon-name", "", "name of the application icon artifact")
	cmd.Flags().String("default-profile", "", "default deployment profile")
	cmd.Flags().StringSlice("application-reference", []string{}, "<name>:<version> constituent application references")
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

var deploymentPackageHeader = fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
	"Name", "Display Name", "Version", "Kind", "Default Profile", "Is Deployed", "Is Visible", "Application Count")

func printDeploymentPackages(writer io.Writer, caList *[]catapi.CatalogV3DeploymentPackage, verbose bool) {
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
	// Check if a file path was provided (single argument ending with .yaml or .yml)
	if len(args) == 1 && (strings.HasSuffix(args[0], ".yaml") || strings.HasSuffix(args[0], ".yml")) {
		return uploadResourceFile(cmd, args[0])
	}

	// Validate we have name and version
	if len(args) != 2 {
		return fmt.Errorf("requires either a YAML file path or <name> <version> arguments")
	}

	applicationName := args[0]
	applicationVersion := args[1]

	// Validate version format
	if err := validator.ValidateVersion(applicationVersion); err != nil {
		return err
	}

	// Validate required flags when not using YAML file
	appRefs, _ := cmd.Flags().GetStringSlice("application-reference")
	if len(appRefs) == 0 {
		return fmt.Errorf("--application-reference is required when not using a YAML file (at least one application must be referenced)")
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

	defaultKind := catapi.KINDNORMAL
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
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating deployment package %s", applicationName))
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

func runListDeploymentPackagesCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
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
	// Check if a file path was provided (single argument ending with .yaml or .yml)
	if len(args) == 1 && (strings.HasSuffix(args[0], ".yaml") || strings.HasSuffix(args[0], ".yml")) {
		return uploadResourceFile(cmd, args[0])
	}

	// Validate we have name and version
	if len(args) != 2 {
		return fmt.Errorf("requires either a YAML file path or <name> <version> arguments")
	}

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
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while updating deployment package %s:%s", name, version))
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
		return checkResponse(deleteResp.HTTPResponse, deleteResp.Body, fmt.Sprintf("error deleting deployment package %s:%s", name, version))
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
	return nil
}

func runExportDeploymentPackageCommand(cmd *cobra.Command, args []string) error {
	ctx, _, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]

	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return processError(err)
	}

	utilClient, err := catutilapi.NewClientWithResponses(serverAddress)
	if err != nil {
		return processError(err)
	}

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

func printDeploymentPackageEvent(writer io.Writer, _ string, payload []byte, verbose bool) error {
	var item catapi.CatalogV3DeploymentPackage
	if err := json.Unmarshal(payload, &item); err != nil {
		return err
	}
	printDeploymentPackages(writer, &[]catapi.CatalogV3DeploymentPackage{item}, verbose)
	return nil
}

// applicationYAMLSpec represents the structure of an application YAML file
type applicationYAMLSpec struct {
	SpecSchema string `yaml:"specSchema"`
	Profiles   []struct {
		Name           string `yaml:"name"`
		ValuesFileName string `yaml:"valuesFileName"`
	} `yaml:"profiles"`
}

// extractReferencedFiles extracts referenced values files from an application YAML
func extractReferencedFiles(yamlPath string) ([]string, error) {
	// Read the YAML file
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	// Parse the YAML to check if it's an application spec
	var spec applicationYAMLSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		// Not a valid YAML or not an application spec, return empty list
		return nil, nil
	}

	// Check if this is an Application spec
	if spec.SpecSchema != "Application" {
		return nil, nil
	}

	// Extract values file names from profiles
	var referencedFiles []string
	baseDir := filepath.Dir(yamlPath)

	for _, profile := range spec.Profiles {
		if profile.ValuesFileName != "" {
			valuesFilePath := filepath.Join(baseDir, profile.ValuesFileName)
			// Check if the file exists
			if _, err := os.Stat(valuesFilePath); err == nil {
				referencedFiles = append(referencedFiles, valuesFilePath)
			} else {
				// File doesn't exist, but we should warn the user
				fmt.Fprintf(os.Stderr, "Warning: Referenced values file not found: %s\n", valuesFilePath)
			}
		}
	}

	return referencedFiles, nil
}

// uploadResourceFile uploads a YAML file containing resource definitions
// For application YAMLs, it also automatically uploads any referenced values files
func uploadResourceFile(cmd *cobra.Command, filePath string) error {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return err
	}

	projectUUID, err := getProjectName(cmd)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Get the access token
	accessToken, err := auth.GetAccessToken(ctx)
	if err != nil {
		// Log warning but continue with empty token
		accessToken = ""
	}

	// Collect all files to upload
	filesToUpload := []string{filePath}

	// Check if this is an application YAML and extract referenced files
	referencedFiles, err := extractReferencedFiles(filePath)
	if err != nil {
		return fmt.Errorf("failed to extract referenced files: %w", err)
	}

	if len(referencedFiles) > 0 {
		filesToUpload = append(filesToUpload, referencedFiles...)
		fmt.Printf("Uploading application with %d referenced values file(s)\n", len(referencedFiles))
	}

	loader := loader.NewLoader(serverAddress, projectUUID)
	return loader.LoadResources(ctx, accessToken, filesToUpload)
}
