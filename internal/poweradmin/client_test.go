package poweradmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testV2RecordsPath = "/api/v2/zones/1/records"

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

// writeV2ZonesResponse writes a wrapped v2 zones response.
func writeV2ZonesResponse(w http.ResponseWriter, zones []Zone) {
	w.Header().Set("Content-Type", "application/json")
	zonesData, _ := json.Marshal(zones)
	resp := apiResponse{
		Success: true,
		Data:    json.RawMessage(`{"zones":` + string(zonesData) + `}`),
		Message: "Zones retrieved successfully",
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// writeV1ZonesResponse writes a wrapped v1 zones response.
func writeV1ZonesResponse(w http.ResponseWriter, zones []Zone) {
	w.Header().Set("Content-Type", "application/json")
	zonesData, _ := json.Marshal(zones)
	resp := apiResponse{
		Success: true,
		Data:    zonesData,
		Message: "Zones retrieved successfully",
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// writeV2RecordsResponse writes a wrapped v2 records response (4.3.0+: {"records": [...]}).
func writeV2RecordsResponse(w http.ResponseWriter, records []Record) {
	w.Header().Set("Content-Type", "application/json")
	recordsData, _ := json.Marshal(records)
	resp := apiResponse{
		Success: true,
		Data:    json.RawMessage(`{"records":` + string(recordsData) + `}`),
		Message: "Records retrieved successfully",
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// writeV1RecordsResponse writes a flat v1 records response (data is [...]).
func writeV1RecordsResponse(w http.ResponseWriter, records []Record) {
	w.Header().Set("Content-Type", "application/json")
	recordsData, _ := json.Marshal(records)
	resp := apiResponse{
		Success: true,
		Data:    recordsData,
		Message: "Records retrieved successfully",
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// writeV2LegacyRecordsResponse writes a flat v2 records response (pre-4.3.0: data is [...]).
func writeV2LegacyRecordsResponse(w http.ResponseWriter, records []Record) {
	w.Header().Set("Content-Type", "application/json")
	recordsData, _ := json.Marshal(records)
	resp := apiResponse{
		Success: true,
		Data:    recordsData,
		Message: "Records retrieved successfully",
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// writeV2CreateRecordResponse writes a wrapped v2 create-record response.
func writeV2CreateRecordResponse(w http.ResponseWriter, record Record) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	recordData, _ := json.Marshal(record)
	resp := apiResponse{
		Success: true,
		Data:    json.RawMessage(`{"record":` + string(recordData) + `}`),
		Message: "Record created successfully",
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// writeV1CreateRecordResponse writes a wrapped v1 create-record response
// using "record_id" instead of "id", matching the real PowerAdmin v1 API.
func writeV1CreateRecordResponse(w http.ResponseWriter, record Record) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	v1data := v1RecordData{
		RecordID: record.ID,
		Name:     record.Name,
		Type:     record.Type,
		Content:  record.Content,
		TTL:      record.TTL,
		Priority: record.Priority,
		Disabled: record.Disabled,
	}
	recordData, _ := json.Marshal(v1data)
	resp := apiResponse{
		Success: true,
		Data:    recordData,
		Message: "Record created successfully",
	}
	_ = json.NewEncoder(w).Encode(resp)
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
		writeV2ZonesResponse(w, []Zone{{ID: 1, Name: "example.com"}})
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
			writeV2ZonesResponse(w, zones)
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

	// Test v2 wrapped response (4.3.0+)
	t.Run("v2 wrapped", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == testV2RecordsPath && r.URL.Query().Get("type") == "TXT" {
				writeV2RecordsResponse(w, records)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}

		_, client := setupTestServerWithVersion(t, handler, "v2")
		result, err := client.ListTXTRecords(context.Background(), 1)
		if err != nil {
			t.Fatalf("ListTXTRecords() error = %v", err)
		}
		if len(result) != 1 || result[0].ID != 10 {
			t.Errorf("ListTXTRecords() = %+v, want 1 record with ID=10", result)
		}
	})

	// Test v2 legacy flat response (pre-4.3.0)
	t.Run("v2 legacy flat", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == testV2RecordsPath && r.URL.Query().Get("type") == "TXT" {
				writeV2LegacyRecordsResponse(w, records)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}

		_, client := setupTestServerWithVersion(t, handler, "v2")
		result, err := client.ListTXTRecords(context.Background(), 1)
		if err != nil {
			t.Fatalf("ListTXTRecords() error = %v", err)
		}
		if len(result) != 1 || result[0].ID != 10 {
			t.Errorf("ListTXTRecords() = %+v, want 1 record with ID=10", result)
		}
	})
}

func TestCreateTXTRecord(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == testV2RecordsPath {
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

			writeV2CreateRecordResponse(w, Record{
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
			writeV1ZonesResponse(w, []Zone{{ID: 1, Name: "example.com"}})
		case r.URL.Path == "/api/v1/zones/1/records" && r.Method == http.MethodGet:
			writeV1RecordsResponse(w, []Record{})
		case r.URL.Path == "/api/v1/zones/1/records" && r.Method == http.MethodPost:
			writeV1CreateRecordResponse(w, Record{ID: 1})
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

	rec, err := client.CreateTXTRecord(ctx, 1, "test", "\"val\"", 120)
	if err != nil {
		t.Fatalf("V1 CreateTXTRecord() error = %v", err)
	}
	if rec.ID != 1 {
		t.Errorf("V1 CreateTXTRecord() record.ID = %d, want 1 (from record_id)", rec.ID)
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
		writeV2ZonesResponse(w, []Zone{})
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
		writeV2ZonesResponse(w, []Zone{})
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
		writeV2RecordsResponse(w, []Record{})
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
		writeV2ZonesResponse(w, []Zone{})
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

func TestFlexBool_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    FlexBool
		wantErr bool
	}{
		{"bool true", "true", true, false},
		{"bool false", "false", false, false},
		{"int 1", "1", true, false},
		{"int 0", "0", false, false},
		{"invalid string", `"yes"`, false, true},
		{"invalid number", "2", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b FlexBool
			err := b.UnmarshalJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && b != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %v, want %v", tt.input, b, tt.want)
			}
		})
	}
}

func TestFlexBool_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		val  FlexBool
		want string
	}{
		{"true", true, "true"},
		{"false", false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.val.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(data) != tt.want {
				t.Errorf("MarshalJSON() = %s, want %s", string(data), tt.want)
			}
		})
	}
}

func TestFlexBool_RecordUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		json string
		want FlexBool
	}{
		{"disabled as bool false", `{"id":1,"name":"test","type":"TXT","content":"val","ttl":120,"priority":0,"disabled":false}`, false},
		{"disabled as bool true", `{"id":1,"name":"test","type":"TXT","content":"val","ttl":120,"priority":0,"disabled":true}`, true},
		{"disabled as int 0", `{"id":1,"name":"test","type":"TXT","content":"val","ttl":120,"priority":0,"disabled":0}`, false},
		{"disabled as int 1", `{"id":1,"name":"test","type":"TXT","content":"val","ttl":120,"priority":0,"disabled":1}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r Record
			if err := json.Unmarshal([]byte(tt.json), &r); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if r.Disabled != tt.want {
				t.Errorf("Record.Disabled = %v, want %v", r.Disabled, tt.want)
			}
		})
	}
}

func TestEnsureTXTQuoted(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"unquoted", "test-value", `"test-value"`},
		{"already quoted", `"test-value"`, `"test-value"`},
		{"empty", "", `""`},
		{"with spaces", "hello world", `"hello world"`},
		{"already double-quoted", `""test""`, `"test"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnsureTXTQuoted(tt.content)
			if got != tt.want {
				t.Errorf("EnsureTXTQuoted(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestNormalizeTXTContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"quoted", `"test-value"`, "test-value"},
		{"unquoted", "test-value", "test-value"},
		{"empty", "", ""},
		{"empty quotes", `""`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTXTContent(tt.content)
			if got != tt.want {
				t.Errorf("NormalizeTXTContent(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestCreateTXTRecord_AutoQuotes(t *testing.T) {
	var receivedContent string
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == testV2RecordsPath {
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode request body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			receivedContent = body["content"].(string)

			writeV2CreateRecordResponse(w, Record{
				ID: 30, Name: "_acme-challenge.example.com",
				Type: "TXT", Content: receivedContent, TTL: 120,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
	ctx := context.Background()

	// Send unquoted content — should be auto-quoted
	_, err := client.CreateTXTRecord(ctx, 1, "_acme-challenge.example.com", "unquoted-value", 120)
	if err != nil {
		t.Fatalf("CreateTXTRecord() error = %v", err)
	}
	if receivedContent != `"unquoted-value"` {
		t.Errorf("expected quoted content %q, got %q", `"unquoted-value"`, receivedContent)
	}

	// Send already-quoted content — should not double-quote
	_, err = client.CreateTXTRecord(ctx, 1, "_acme-challenge.example.com", `"already-quoted"`, 120)
	if err != nil {
		t.Fatalf("CreateTXTRecord() error = %v", err)
	}
	if receivedContent != `"already-quoted"` {
		t.Errorf("expected quoted content %q, got %q", `"already-quoted"`, receivedContent)
	}
}

func TestListTXTRecords_DisabledAsBool(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Simulate v2 4.3.0+ wrapped response with disabled as bool
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"records":[{"id":10,"name":"_acme.example.com","type":"TXT","content":"\"test\"","ttl":120,"priority":0,"disabled":false}]},"message":"ok"}`))
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
	records, err := client.ListTXTRecords(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListTXTRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Disabled != false {
		t.Errorf("expected Disabled=false, got %v", records[0].Disabled)
	}
}

func TestListTXTRecords_DisabledAsInt(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Simulate v2 legacy flat response with disabled as int
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":[{"id":10,"name":"_acme.example.com","type":"TXT","content":"\"test\"","ttl":120,"priority":0,"disabled":1}],"message":"ok"}`))
	}

	_, client := setupTestServerWithVersion(t, handler, "v2")
	records, err := client.ListTXTRecords(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListTXTRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Disabled != true {
		t.Errorf("expected Disabled=true, got %v", records[0].Disabled)
	}
}
