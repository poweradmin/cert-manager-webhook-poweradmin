package poweradmin

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DNSProvider abstracts PowerAdmin API operations needed for DNS-01 challenges.
type DNSProvider interface {
	GetZones(ctx context.Context) ([]Zone, error)
	GetZoneByName(ctx context.Context, name string) (*Zone, error)
	ListTXTRecords(ctx context.Context, zoneID int) ([]Record, error)
	CreateTXTRecord(ctx context.Context, zoneID int, name, content string, ttl int) (*Record, error)
	DeleteRecord(ctx context.Context, zoneID int, recordID int) error
}

// baseClient holds shared HTTP plumbing for v1/v2 clients.
type baseClient struct {
	serverURL  string
	apiKey     string
	httpClient *http.Client
}

func (c *baseClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.serverURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// NewClient creates a DNSProvider for the given API version.
// apiVersion can be "v1", "v2", or "" (defaults to "v2").
func NewClient(serverURL, apiKey, apiVersion string, insecure bool) (DNSProvider, error) {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	if insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	base := &baseClient{
		serverURL:  serverURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}

	switch apiVersion {
	case "v1":
		return &v1Client{baseClient: base}, nil
	case "", "v2":
		return &v2Client{baseClient: base}, nil
	default:
		return nil, fmt.Errorf("unsupported PowerAdmin API version: %q (supported: v1, v2)", apiVersion)
	}
}
