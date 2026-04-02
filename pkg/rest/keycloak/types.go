// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package keycloak

// UserRepresentation mirrors the Keycloak UserRepresentation JSON structure.
type UserRepresentation struct {
	ID        string `json:"id,omitempty"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Enabled   *bool  `json:"enabled,omitempty"`
}

// GroupRepresentation mirrors the Keycloak GroupRepresentation JSON structure.
type GroupRepresentation struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Path string `json:"path,omitempty"`
}

// CredentialRepresentation is used for password reset requests.
type CredentialRepresentation struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Temporary bool   `json:"temporary"`
}
