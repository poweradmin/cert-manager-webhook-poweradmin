package poweradmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type v1Client struct {
	*baseClient
}

func (c *v1Client) GetZones(ctx context.Context) ([]Zone, error) {
	body, status, err := c.doRequest(ctx, http.MethodGet, "/api/v1/zones", nil)
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

func (c *v1Client) GetZoneByName(ctx context.Context, name string) (*Zone, error) {
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

func (c *v1Client) ListTXTRecords(ctx context.Context, zoneID int) ([]Record, error) {
	path := fmt.Sprintf("/api/v1/zones/%d/records?type=TXT", zoneID)
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

func (c *v1Client) CreateTXTRecord(ctx context.Context, zoneID int, name, content string, ttl int) (*Record, error) {
	path := fmt.Sprintf("/api/v1/zones/%d/records", zoneID)
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

func (c *v1Client) DeleteRecord(ctx context.Context, zoneID int, recordID int) error {
	path := fmt.Sprintf("/api/v1/zones/%d/records/%d", zoneID, recordID)
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("PowerAdmin API returned HTTP %d: %s", status, string(body))
	}
	return nil
}
