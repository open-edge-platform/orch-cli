// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"net/http"

	"github.com/open-edge-platform/cli/pkg/auth"
	restapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
)

func getWipeProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "wipe [flags]",
		Args:              cobra.NoArgs,
		Short:             "Wipe all data associated with the specified project",
		PersistentPreRunE: auth.CheckAuth,
		Example:           "orch-cli wipe --project some-project --yes",
		RunE:              runWipeProjectCommand,
	}
	_ = cmd.MarkFlagRequired(project)
	cmd.Flags().BoolP("yes", "y", false, "artifact MIME type (required)")
	return cmd
}

func runWipeProjectCommand(cmd *cobra.Command, _ []string) error {
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}
	w := &wiper{client: *catalogClient, reqEditors: []restapi.RequestEditorFn{auth.AddAuthHeader}}

	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		return fmt.Errorf("you have to say yes")
	}

	errors := w.wipe(ctx, projectName)
	for i, err := range errors {
		fmt.Printf("#%d: %+v\n", i, err)
	}
	return nil
}

type wiper struct {
	client     restapi.ClientWithResponses
	reqEditors []restapi.RequestEditorFn
}

// Deletes all entities (packages, apps, registries, and artifacts) for the given project.
func (w *wiper) wipe(ctx context.Context, projectName string) []error {
	var errors []error

	errors = append(errors, w.preparePackagesForDeletion(ctx, projectName)...)
	errors = append(errors, w.prepareApplicationsForDeletion(ctx, projectName)...)

	errors = append(errors, w.wipePackages(ctx, projectName)...)
	errors = append(errors, w.wipeApplications(ctx, projectName)...)
	errors = append(errors, w.wipeArtifacts(ctx, projectName)...)
	errors = append(errors, w.wipeRegistries(ctx, projectName)...)
	return errors
}

var (
	maxPageSize = int32(500)
	notDeployed = false
)

// Sweeps through all packages, marking them as not deployed
func (w *wiper) preparePackagesForDeletion(ctx context.Context, projectName string) []error {
	var errors []error
	resp, err := w.client.CatalogServiceListDeploymentPackagesWithResponse(ctx, projectName, &restapi.CatalogServiceListDeploymentPackagesParams{PageSize: &maxPageSize}, w.reqEditors...)
	if err != nil {
		return append(errors, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil
	}

	for _, pkg := range resp.JSON200.DeploymentPackages {
		if err = w.preparePackageForDeletion(ctx, projectName, pkg.Name, pkg.Version); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (w *wiper) preparePackageForDeletion(ctx context.Context, projectName string, name string, version string) error {
	gresp, err := w.client.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, name, version, w.reqEditors...)
	if err != nil {
		return err
	}
	if gresp.StatusCode() != http.StatusOK {
		return nil
	}

	// Update package to sever any dependencies from its point of view
	pkg := gresp.JSON200.DeploymentPackage
	pkg.IsDeployed = &notDeployed
	pkg.Profiles = nil
	pkg.ApplicationReferences = nil
	pkg.ApplicationDependencies = nil
	pkg.DefaultNamespaces = nil
	pkg.DefaultProfileName = nil

	if _, err = w.client.CatalogServiceUpdateDeploymentPackageWithResponse(ctx, projectName, name, version, pkg, w.reqEditors...); err != nil {
		return err
	}
	return nil
}

// Sweeps through all applications, severing their dependencies on any deployment packages
func (w *wiper) prepareApplicationsForDeletion(ctx context.Context, projectName string) []error {
	var errors []error
	offset := int32(0)
	hasMorePages := true
	for hasMorePages {
		resp, err := w.client.CatalogServiceListApplicationsWithResponse(ctx, projectName, &restapi.CatalogServiceListApplicationsParams{PageSize: &maxPageSize, Offset: &offset}, w.reqEditors...)
		if resp.StatusCode() != http.StatusOK {
			return nil
		}

		if err != nil {
			return append(errors, err)
		}
		for _, app := range resp.JSON200.Applications {
			if err = w.prepareApplicationForDeletion(ctx, projectName, app.Name, app.Version); err != nil {
				errors = append(errors, err)
			}
		}
		hasMorePages = resp.JSON200.TotalElements > offset+int32(len(resp.JSON200.Applications))
		offset = offset + int32(len(resp.JSON200.Applications))
	}
	return errors
}

func (w *wiper) prepareApplicationForDeletion(ctx context.Context, projectName string, name string, version string) error {
	gresp, err := w.client.CatalogServiceGetApplicationWithResponse(ctx, projectName, name, version, w.reqEditors...)
	if err != nil {
		return err
	}

	// Update app to remove any profiles that might have dependencies on packages
	if gresp.StatusCode() != http.StatusOK {
		return nil
	}

	app := gresp.JSON200.Application
	app.Profiles = nil
	app.DefaultProfileName = nil

	if _, err = w.client.CatalogServiceUpdateApplicationWithResponse(ctx, projectName, name, version, app, w.reqEditors...); err != nil {
		return err
	}
	return nil
}

func (w *wiper) wipePackages(ctx context.Context, projectName string) []error {
	var errors []error
	resp, err := w.client.CatalogServiceListDeploymentPackagesWithResponse(ctx, projectName, &restapi.CatalogServiceListDeploymentPackagesParams{PageSize: &maxPageSize}, w.reqEditors...)
	if err != nil {
		return append(errors, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil
	}

	for _, pkg := range resp.JSON200.DeploymentPackages {
		if _, err = w.client.CatalogServiceDeleteDeploymentPackageWithResponse(ctx, projectName, pkg.Name, pkg.Version, w.reqEditors...); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (w *wiper) wipeApplications(ctx context.Context, projectName string) []error {
	var errors []error
	resp, err := w.client.CatalogServiceListApplicationsWithResponse(ctx, projectName, &restapi.CatalogServiceListApplicationsParams{PageSize: &maxPageSize}, w.reqEditors...)
	if err != nil {
		return append(errors, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil
	}

	for _, app := range resp.JSON200.Applications {
		if _, err = w.client.CatalogServiceDeleteApplicationWithResponse(ctx, projectName, app.Name, app.Version, w.reqEditors...); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (w *wiper) wipeArtifacts(ctx context.Context, projectName string) []error {
	var errors []error
	resp, err := w.client.CatalogServiceListArtifactsWithResponse(ctx, projectName, &restapi.CatalogServiceListArtifactsParams{PageSize: &maxPageSize}, w.reqEditors...)
	if err != nil {
		return append(errors, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil
	}

	for _, artifact := range resp.JSON200.Artifacts {
		if _, err = w.client.CatalogServiceDeleteArtifactWithResponse(ctx, projectName, artifact.Name, w.reqEditors...); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (w *wiper) wipeRegistries(ctx context.Context, projectName string) []error {
	var errors []error
	resp, err := w.client.CatalogServiceListRegistriesWithResponse(ctx, projectName, &restapi.CatalogServiceListRegistriesParams{PageSize: &maxPageSize}, w.reqEditors...)
	if err != nil {
		return append(errors, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil
	}

	for _, registry := range resp.JSON200.Registries {
		if _, err = w.client.CatalogServiceDeleteRegistryWithResponse(ctx, projectName, registry.Name, w.reqEditors...); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}
