package interfaces

import (
	"context"

	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	cluster "github.com/open-edge-platform/cli/pkg/rest/cluster"
	depapi "github.com/open-edge-platform/cli/pkg/rest/deployment"
	infraapi "github.com/open-edge-platform/cli/pkg/rest/infra"
	rpsapi "github.com/open-edge-platform/cli/pkg/rest/rps"
	"github.com/open-edge-platform/orch-library/go/pkg/openidconnect"
	"github.com/spf13/cobra"
)

type InfraFactoryFunc func(cmd *cobra.Command) (context.Context, infraapi.ClientWithResponsesInterface, string, error)
type ClusterFactoryFunc func(cmd *cobra.Command) (context.Context, cluster.ClientWithResponsesInterface, string, error)
type CatalogFactoryFunc func(cmd *cobra.Command) (context.Context, catapi.ClientWithResponsesInterface, string, error)
type KeycloakFactoryFunc func(ctx context.Context, endpoint string) (openidconnect.ClientWithResponsesInterface, error)
type RpsFactoryFunc func(cmd *cobra.Command) (context.Context, rpsapi.ClientWithResponsesInterface, string, error)
type DeploymentFactoryFunc func(cmd *cobra.Command) (context.Context, depapi.ClientWithResponsesInterface, string, error)
