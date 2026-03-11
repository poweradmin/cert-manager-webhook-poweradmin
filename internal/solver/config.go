package solver

import (
	"encoding/json"
	"fmt"

	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type poweradminDNSProviderConfig struct {
	// ServerURL is the base URL of the PowerAdmin instance (e.g., https://poweradmin.example.com).
	ServerURL string `json:"serverURL"`

	// APIKeySecretRef references a Kubernetes Secret containing the PowerAdmin API key.
	APIKeySecretRef cmmeta.SecretKeySelector `json:"apiKeySecretRef"`

	// APIVersion selects the PowerAdmin API version: "v1" or "v2" (default: "v2").
	APIVersion string `json:"apiVersion,omitempty"`

	// TTL for the TXT record in seconds (default: 120).
	TTL int `json:"ttl,omitempty"`

	// Insecure disables TLS certificate verification for the PowerAdmin API.
	Insecure bool `json:"insecure,omitempty"`
}

func loadConfig(cfgJSON *extapi.JSON) (poweradminDNSProviderConfig, error) {
	cfg := poweradminDNSProviderConfig{}
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %w", err)
	}
	return cfg, nil
}
