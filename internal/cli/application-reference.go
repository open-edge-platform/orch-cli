// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
)

func getCreateApplicationReferenceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "application-reference <deployment-package-name> <version> <application-name:version> [flags]",
		Aliases: []string{"app-reference"},
		Short:   "Create an application reference within a deployment package",
		Args:    cobra.ExactArgs(3),
		Example: "orch-cli create application-reference my-package 1.0.0 my-app:1.0.0 --project some-project",
		RunE:    runCreateApplicationReferenceCommand,
	}
	return cmd
}

func getDeleteApplicationReferenceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "application-reference <deployment-package-name> <version> <application-name> [flags]",
		Aliases: []string{"app-reference"},
		Short:   "Delete an application reference within a deployment package",
		Args:    cobra.ExactArgs(3),
		Example: "orch-cli delete application-reference my-package 1.0.0 my-app --project some-project",
		RunE:    runDeleteApplicationReferenceCommand,
	}
	return cmd
}

func runCreateApplicationReferenceCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	pkgName := args[0]
	pkgVersion := args[1]

	applicationFields := strings.SplitN(args[2], ":", 3)

	gresp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, pkgName, pkgVersion,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, fmt.Sprintf("deployment package %s:%s not found", pkgName, pkgVersion)); err != nil {
		return err
	}

	pkg := gresp.JSON200.DeploymentPackage
	pkg.ApplicationReferences = append(pkg.ApplicationReferences, catapi.ApplicationReference{
		Name:    applicationFields[0],
		Version: applicationFields[1],
	})

	resp, err := updateDeploymentPackage(ctx, projectName, catalogClient, pkg)
	if err != nil {
		return err
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating application reference %s", args[2]))
}

func runDeleteApplicationReferenceCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	pkgName := args[0]
	pkgVersion := args[1]
	applicationName := args[2]

	gresp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, pkgName, pkgVersion,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, fmt.Sprintf("deployment package %s:%s not found", pkgName, pkgVersion)); err != nil {
		return err
	}

	pkg := gresp.JSON200.DeploymentPackage
	for i, ref := range pkg.ApplicationReferences {
		if ref.Name == applicationName {
			pkg.ApplicationReferences = append(pkg.ApplicationReferences[:i], pkg.ApplicationReferences[i+1:]...)
			break
		}
	}

	resp, err := updateDeploymentPackage(ctx, projectName, catalogClient, pkg)
	if err != nil {
		return err
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting application reference %s for deployment package %s:%s",
		applicationName, pkgName, pkgVersion))
}

func updateDeploymentPackage(ctx context.Context, projectName string, client catapi.ClientWithResponsesInterface, pkg catapi.DeploymentPackage) (*catapi.CatalogServiceUpdateDeploymentPackageResponse, error) {
	return client.CatalogServiceUpdateDeploymentPackageWithResponse(ctx, projectName, pkg.Name, pkg.Version,
		catapi.CatalogServiceUpdateDeploymentPackageJSONRequestBody{
			Name:                    pkg.Name,
			Version:                 pkg.Version,
			DisplayName:             pkg.DisplayName,
			Description:             pkg.Description,
			ApplicationReferences:   pkg.ApplicationReferences,
			Profiles:                pkg.Profiles,
			DefaultProfileName:      pkg.DefaultProfileName,
			ApplicationDependencies: pkg.ApplicationDependencies,
			Artifacts:               pkg.Artifacts,
			DefaultNamespaces:       pkg.DefaultNamespaces,
			Extensions:              pkg.Extensions,
			IsDeployed:              pkg.IsDeployed,
			IsVisible:               pkg.IsVisible,
		}, auth.AddAuthHeader)
}
