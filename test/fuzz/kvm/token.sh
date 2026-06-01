#!/bin/sh
# SPDX-FileCopyrightText: (C) 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
#
# RESTler token refresh script for KVM fuzz testing.
#
# Outputs the X-Session-Token header line consumed by RESTler's
# authentication engine. The token value must match kvmFuzzToken
# in internal/cli/kvm_fuzz_test.go.
#
# To use a different token, set KVM_FUZZ_TOKEN in the environment
# and restart TestKVMFuzzServer with the same value via KVM_FUZZ_TOKEN.
TOKEN="${KVM_FUZZ_TOKEN:-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef}"
echo "X-Session-Token: ${TOKEN}"
