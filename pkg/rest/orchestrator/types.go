// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package orchestrator

import (
	"encoding/json"
	"net/http"
)

// FeatureInfo represents a feature with installation status and optional nested features
type FeatureInfo struct {
	Installed *bool                  `json:"installed,omitempty"`
	Features  map[string]FeatureInfo `json:"-"` // For nested features
}

// UnmarshalJSON custom unmarshaler to handle nested features
func (f *FeatureInfo) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to get all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract installed field
	if installed, ok := raw["installed"].(bool); ok {
		f.Installed = &installed
	}

	// Extract nested features (any field that's not "installed")
	f.Features = make(map[string]FeatureInfo)
	for key, value := range raw {
		if key != "installed" {
			if valueMap, ok := value.(map[string]interface{}); ok {
				// Re-marshal and unmarshal to convert to FeatureInfo
				valueBytes, _ := json.Marshal(valueMap)
				var nestedFeature FeatureInfo
				if err := json.Unmarshal(valueBytes, &nestedFeature); err == nil {
					f.Features[key] = nestedFeature
				}
			}
		}
	}

	return nil
}

// Data represents the orchestrator section of the response
type Data struct {
	Version  *string                `json:"version,omitempty"`
	Features map[string]FeatureInfo `json:"features,omitempty"`
}

// Info defines model for orchestrator information response
type Info struct {
	SchemaVersion *string `json:"schema-version,omitempty"`
	Orchestrator  *Data   `json:"orchestrator,omitempty"`
}

// InfoResponse represents the response from GetOrchestratorInfo
type InfoResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *Info
}

// Status returns HTTPResponse.Status
func (r InfoResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r InfoResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}
