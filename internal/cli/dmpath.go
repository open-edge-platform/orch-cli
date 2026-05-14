// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	mpsapi "github.com/open-edge-platform/cli/pkg/rest/mps"
	rpsapi "github.com/open-edge-platform/cli/pkg/rest/rps"
)

// rewriteDmPath rewrites a Device-Management (RPS/MPS) client request path
// from the upstream service-internal form ("/api/v1/<rest>") to the
// orch-gateway external form ("/v1/projects/{projectName}/<dmSubPath>/<rest>").
//
// The orch-gateway Traefik rewrite-rps / rewrite-mps middlewares then strip
// the project prefix back to "/api/v1/<rest>" before forwarding to the
// upstream service container, so the upstream service sees the path it
// expects from its OpenAPI spec.
//
// Per-service mappings (see traefik-extra-objects IngressRoutes):
//
//   RPS  →  dmSubPath = "dm/amt"
//        client  /api/v1/admin/domains
//        gateway /v1/projects/{p}/dm/amt/admin/domains
//
//   MPS  →  dmSubPath = "dm"
//        client  /api/v1/devices
//        gateway /v1/projects/{p}/dm/devices
//        client  /api/v1/amt/generalSettings
//        gateway /v1/projects/{p}/dm/amt/generalSettings
//
// projectName == "" leaves the request unchanged so unit tests and other
// callers without a project context still work.
func rewriteDmPath(projectName, dmSubPath string, req *http.Request) {
	if projectName == "" {
		return
	}
	const prefix = "/api/v1/"
	if !strings.HasPrefix(req.URL.Path, prefix) {
		return
	}
	rest := strings.TrimPrefix(req.URL.Path, prefix)
	req.URL.Path = fmt.Sprintf("/v1/projects/%s/%s/%s", projectName, dmSubPath, rest)
	req.URL.RawPath = ""
}

// rewriteRpsAmtPath returns a request editor for the generated RPS client.
func rewriteRpsAmtPath(projectName string) rpsapi.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		rewriteDmPath(projectName, "dm/amt", req)
		return nil
	}
}

// rewriteMpsPath returns a request editor for the generated MPS client.
func rewriteMpsPath(projectName string) mpsapi.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		rewriteDmPath(projectName, "dm", req)
		return nil
	}
}
