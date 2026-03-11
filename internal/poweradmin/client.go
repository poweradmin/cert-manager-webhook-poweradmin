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

// client implements DNSProvider for any PowerAdmin API version.
type client struct {
	serverURL  string
	apiKey     string
	pathPrefix string // "/api/v1" or "/api/v2"
	httpClient *http.Client
}

func (c *client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.serverURL + c.pathPrefix + path
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

func (c *client) GetZones(ctx context.Context) ([]Zone, error) {
	body, status, err := c.doRequest(ctx, http.MethodGet, "/zones", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("PowerAdmin API returned HTTP %d: %s", status, string(body))
	}

	var zones []Zone
	if err := json.Unmarshal(body, &zones); err != nil {
		return nil, fmt.Errorf("failed to parse zones response: %w", err)
	}
	return zones, nil
}

func (c *client) GetZoneByName(ctx context.Context, name string) (*Zone, error) {
	zones, err := c.GetZones(ctx)
	if err != nil {
		return nil, err
	}
	for _, z := range zones {
		if z.Name == name {
			return &z, nil
		}
	}
	return nil, fmt.Errorf("zone %q not found", name)
}

func (c *client) ListTXTRecords(ctx context.Context, zoneID int) ([]Record, error) {
	path := fmt.Sprintf("/zones/%d/records?type=TXT", zoneID)
	body, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("PowerAdmin API returned HTTP %d: %s", status, string(body))
	}

	var records []Record
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, fmt.Errorf("failed to parse records response: %w", err)
	}
	return records, nil
}

func (c *client) CreateTXTRecord(ctx context.Context, zoneID int, name, content string, ttl int) (*Record, error) {
	path := fmt.Sprintf("/zones/%d/records", zoneID)
	reqBody := map[string]interface{}{
		"name":     name,
		"type":     "TXT",
		"content":  content,
		"ttl":      ttl,
		"priority": 0,
		"disabled": 0,
	}

	body, status, err := c.doRequest(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, fmt.Errorf("PowerAdmin API returned HTTP %d: %s", status, string(body))
	}

	var record Record
	if err := json.Unmarshal(body, &record); err != nil {
		return nil, fmt.Errorf("failed to parse record response: %w", err)
	}
	return &record, nil
}

func (c *client) DeleteRecord(ctx context.Context, zoneID int, recordID int) error {
	path := fmt.Sprintf("/zones/%d/records/%d", zoneID, recordID)
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("PowerAdmin API returned HTTP %d: %s", status, string(body))
	}
	return nil
}

// NewClient creates a DNSProvider for the given API version.
// apiVersion can be "v1", "v2", or "" (defaults to "v2").
func NewClient(serverURL, apiKey, apiVersion string, insecure bool) (DNSProvider, error) {
	var pathPrefix string
	switch apiVersion {
	case "v1":
		pathPrefix = "/api/v1"
	case "", "v2":
		pathPrefix = "/api/v2"
	default:
		return nil, fmt.Errorf("unsupported PowerAdmin API version: %q (supported: v1, v2)", apiVersion)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	if insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	return &client{
		serverURL:  serverURL,
		apiKey:     apiKey,
		pathPrefix: pathPrefix,
		httpClient: httpClient,
	}, nil
}
