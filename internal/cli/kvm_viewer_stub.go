// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//go:build !kvm

package cli

import (
	"context"
	"fmt"

	"github.com/open-edge-platform/cli/pkg/rest/infra"
)

// startKVMViewer is a stub used when the binary is built without the 'kvm'
// build tag. Run 'make build-kvm' to produce a KVM-enabled binary.
func startKVMViewer(_ context.Context, _, _, _, _ string, _ infra.ClientWithResponsesInterface, _, _ string) error {
	return fmt.Errorf("KVM viewer not available: binary was built without KVM support — rebuild with 'make build-kvm'")
}
