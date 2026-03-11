package poweradmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, DNSProvider) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, "test-api-key", "v2", false)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return server, client
}

func setupTestServerV1(t *testing.T, handler http.HandlerFunc) (*httptest.Server, DNSProvider) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, "test-api-key", "v1", false)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return server, client
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		wantErr    bool
	}{
		{"default (empty) uses v2", "", false},
		{"explicit v2", "v2", false},
		{"explicit v1", "v1", false},
		{"unsupported version", "v3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient("http://localhost", "key", tt.apiVersion, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetZoneByName(t *testing.T) {
	zones := []Zone{
		{ID: 1, Name: "example.com"},
		{ID: 2, Name: "other.com"},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/api/v2/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(zones)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	_, client := setupTestServer(t, handler)
	ctx := context.Background()

	zone, err := client.GetZoneByName(ctx, "example.com")
	if err != nil {
		t.Fatalf("GetZoneByName() error = %v", err)
	}
	if zone.ID != 1 || zone.Name != "example.com" {
		t.Errorf("GetZoneByName() = %+v, want ID=1 Name=example.com", zone)
	}

	_, err = client.GetZoneByName(ctx, "notfound.com")
	if err == nil {
		t.Error("GetZoneByName() expected error for missing zone")
	}
}

func TestListTXTRecords(t *testing.T) {
	records := []Record{
		{ID: 10, Name: "_acme-challenge.example.com", Type: "TXT", Content: "\"test-key\"", TTL: 120},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/zones/1/records" && r.URL.Query().Get("type") == "TXT" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(records)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	_, client := setupTestServer(t, handler)
	ctx := context.Background()

	result, err := client.ListTXTRecords(ctx, 1)
	if err != nil {
		t.Fatalf("ListTXTRecords() error = %v", err)
	}
	if len(result) != 1 || result[0].ID != 10 {
		t.Errorf("ListTXTRecords() = %+v, want 1 record with ID=10", result)
	}
}

func TestCreateTXTRecord(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v2/zones/1/records" {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			if body["type"] != "TXT" {
				t.Errorf("expected type=TXT, got %v", body["type"])
			}
			if body["name"] != "_acme-challenge.example.com" {
				t.Errorf("expected name=_acme-challenge.example.com, got %v", body["name"])
			}

			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Record{
				ID: 20, Name: "_acme-challenge.example.com",
				Type: "TXT", Content: "\"test-key\"", TTL: 120,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	_, client := setupTestServer(t, handler)
	ctx := context.Background()

	record, err := client.CreateTXTRecord(ctx, 1, "_acme-challenge.example.com", "\"test-key\"", 120)
	if err != nil {
		t.Fatalf("CreateTXTRecord() error = %v", err)
	}
	if record.ID != 20 {
		t.Errorf("CreateTXTRecord() record.ID = %d, want 20", record.ID)
	}
}

func TestDeleteRecord(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v2/zones/1/records/10" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	_, client := setupTestServer(t, handler)
	ctx := context.Background()

	err := client.DeleteRecord(ctx, 1, 10)
	if err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}
}

func TestV1Paths(t *testing.T) {
	var receivedPath string

	handler := func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/zones":
			json.NewEncoder(w).Encode([]Zone{{ID: 1, Name: "example.com"}})
		case r.URL.Path == "/api/v1/zones/1/records" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]Record{})
		case r.URL.Path == "/api/v1/zones/1/records" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Record{ID: 1})
		case r.URL.Path == "/api/v1/zones/1/records/1" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}

	_, client := setupTestServerV1(t, handler)
	ctx := context.Background()

	_, err := client.GetZones(ctx)
	if err != nil {
		t.Fatalf("V1 GetZones() error = %v", err)
	}
	if receivedPath != "/api/v1/zones" {
		t.Errorf("V1 GetZones() path = %s, want /api/v1/zones", receivedPath)
	}

	_, err = client.ListTXTRecords(ctx, 1)
	if err != nil {
		t.Fatalf("V1 ListTXTRecords() error = %v", err)
	}

	_, err = client.CreateTXTRecord(ctx, 1, "test", "\"val\"", 120)
	if err != nil {
		t.Fatalf("V1 CreateTXTRecord() error = %v", err)
	}

	err = client.DeleteRecord(ctx, 1, 1)
	if err != nil {
		t.Fatalf("V1 DeleteRecord() error = %v", err)
	}
}

func TestAuthHeader(t *testing.T) {
	var receivedKey string
	handler := func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Zone{})
	}

	_, client := setupTestServer(t, handler)
	client.GetZones(context.Background())

	if receivedKey != "test-api-key" {
		t.Errorf("expected X-API-Key=test-api-key, got %q", receivedKey)
	}
}

func TestHTTPErrors(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}

	_, client := setupTestServer(t, handler)
	ctx := context.Background()

	_, err := client.GetZones(ctx)
	if err == nil {
		t.Error("expected error for 401 response")
	}
}
