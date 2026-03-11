package poweradmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServerWithVersion(t *testing.T, handler http.HandlerFunc, apiVersion string) (*httptest.Server, DNSProvider) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, "test-api-key", apiVersion, false)
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

func TestServerURLNormalization(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/zones" {
			t.Errorf("expected path /api/v2/zones, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Zone{{ID: 1, Name: "example.com"}})
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	tests := []struct {
		name      string
		serverURL string
	}{
		{"no trailing slash", server.URL},
		{"single trailing slash", server.URL + "/"},
		{"multiple trailing slashes", server.URL + "///"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.serverURL, "test-api-key", "v2", false)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			zones, err := client.GetZones(context.Background())
			if err != nil {
				t.Fatalf("GetZones() error = %v", err)
			}
			if len(zones) != 1 {
				t.Errorf("expected 1 zone, got %d", len(zones))
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
			_ = json.NewEncoder(w).Encode(zones)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
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
			_ = json.NewEncoder(w).Encode(records)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
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
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode request body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if body["type"] != "TXT" {
				t.Errorf("expected type=TXT, got %v", body["type"])
			}
			if body["name"] != "_acme-challenge.example.com" {
				t.Errorf("expected name=_acme-challenge.example.com, got %v", body["name"])
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(Record{
				ID: 20, Name: "_acme-challenge.example.com",
				Type: "TXT", Content: "\"test-key\"", TTL: 120,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
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

	_, client := setupTestServerWithVersion(t, handler, "v2")
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
			_ = json.NewEncoder(w).Encode([]Zone{{ID: 1, Name: "example.com"}})
		case r.URL.Path == "/api/v1/zones/1/records" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]Record{})
		case r.URL.Path == "/api/v1/zones/1/records" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(Record{ID: 1})
		case r.URL.Path == "/api/v1/zones/1/records/1" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}

	_, client := setupTestServerWithVersion(t, handler, "v1")
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
		_ = json.NewEncoder(w).Encode([]Zone{})
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
	_, _ = client.GetZones(context.Background())

	if receivedKey != "test-api-key" {
		t.Errorf("expected X-API-Key=test-api-key, got %q", receivedKey)
	}
}

func TestHTTPErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"unauthorized", http.StatusUnauthorized},
		{"forbidden", http.StatusForbidden},
		{"internal server error", http.StatusInternalServerError},
		{"bad gateway", http.StatusBadGateway},
		{"service unavailable", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"error":"test error"}`))
			}

			_, client := setupTestServerWithVersion(t, handler, "v2")
			ctx := context.Background()

			_, err := client.GetZones(ctx)
			if err == nil {
				t.Errorf("expected error for %d response", tt.statusCode)
			}
		})
	}
}

func TestGetZones_EmptyResponse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Zone{})
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
	zones, err := client.GetZones(context.Background())
	if err != nil {
		t.Fatalf("GetZones() error = %v", err)
	}
	if len(zones) != 0 {
		t.Errorf("expected 0 zones, got %d", len(zones))
	}
}

func TestGetZones_InvalidJSON(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not valid json`))
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
	_, err := client.GetZones(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestListTXTRecords_EmptyZone(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Record{})
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
	records, err := client.ListTXTRecords(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListTXTRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestDeleteRecord_HTTPErrors(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
	err := client.DeleteRecord(context.Background(), 1, 999)
	if err == nil {
		t.Error("expected error for 404 response on delete")
	}
}

func TestCreateTXTRecord_ServerError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal error"}`))
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
	_, err := client.CreateTXTRecord(context.Background(), 1, "test", "val", 120)
	if err == nil {
		t.Error("expected error for 500 response on create")
	}
}

func TestNewClient_InsecureTLS(t *testing.T) {
	client, err := NewClient("https://localhost", "key", "v2", true)
	if err != nil {
		t.Fatalf("NewClient() with insecure=true error = %v", err)
	}
	if client == nil {
		t.Error("expected non-nil client with insecure=true")
	}
}

func TestRequestContentType(t *testing.T) {
	var receivedContentType, receivedAccept string
	handler := func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Zone{})
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
	_, _ = client.GetZones(context.Background())

	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q", receivedContentType)
	}
	if receivedAccept != "application/json" {
		t.Errorf("expected Accept=application/json, got %q", receivedAccept)
	}
}
