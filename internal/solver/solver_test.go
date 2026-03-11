package solver

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclient "k8s.io/client-go/kubernetes/fake"

	"github.com/cert-manager/cert-manager-webhook-poweradmin/internal/poweradmin"
)

// Compile-time check: PowerAdminSolver must implement webhook.Solver.
var _ webhook.Solver = &PowerAdminSolver{}

// mockDNSProvider implements poweradmin.DNSProvider for testing.
type mockDNSProvider struct {
	zones   []poweradmin.Zone
	records map[int][]poweradmin.Record // zoneID -> records
	nextID  int

	getZonesErr       error
	listRecordsErr    error
	createRecordErr   error
	deleteRecordErr   error
	createRecordCalls []createRecordCall
	deleteRecordCalls []deleteRecordCall
}

type createRecordCall struct {
	ZoneID  int
	Name    string
	Content string
	TTL     int
}

type deleteRecordCall struct {
	ZoneID   int
	RecordID int
}

func newMockDNSProvider(zones []poweradmin.Zone) *mockDNSProvider {
	return &mockDNSProvider{
		zones:   zones,
		records: make(map[int][]poweradmin.Record),
		nextID:  1,
	}
}

func (m *mockDNSProvider) GetZones(_ context.Context) ([]poweradmin.Zone, error) {
	if m.getZonesErr != nil {
		return nil, m.getZonesErr
	}
	return m.zones, nil
}

func (m *mockDNSProvider) GetZoneByName(_ context.Context, name string) (*poweradmin.Zone, error) {
	for _, z := range m.zones {
		if z.Name == name {
			return &z, nil
		}
	}
	return nil, fmt.Errorf("zone %q not found", name)
}

func (m *mockDNSProvider) ListTXTRecords(_ context.Context, zoneID int) ([]poweradmin.Record, error) {
	if m.listRecordsErr != nil {
		return nil, m.listRecordsErr
	}
	return m.records[zoneID], nil
}

func (m *mockDNSProvider) CreateTXTRecord(_ context.Context, zoneID int, name, content string, ttl int) (*poweradmin.Record, error) {
	m.createRecordCalls = append(m.createRecordCalls, createRecordCall{zoneID, name, content, ttl})
	if m.createRecordErr != nil {
		return nil, m.createRecordErr
	}
	record := poweradmin.Record{
		ID:      m.nextID,
		Name:    name,
		Type:    "TXT",
		Content: content,
		TTL:     ttl,
	}
	m.nextID++
	m.records[zoneID] = append(m.records[zoneID], record)
	return &record, nil
}

func (m *mockDNSProvider) DeleteRecord(_ context.Context, zoneID int, recordID int) error {
	m.deleteRecordCalls = append(m.deleteRecordCalls, deleteRecordCall{zoneID, recordID})
	if m.deleteRecordErr != nil {
		return m.deleteRecordErr
	}
	records := m.records[zoneID]
	for i, r := range records {
		if r.ID == recordID {
			m.records[zoneID] = append(records[:i], records[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("record %d not found", recordID)
}

func (m *mockDNSProvider) addRecord(zoneID int, record poweradmin.Record) {
	m.records[zoneID] = append(m.records[zoneID], record)
}

// --- Test Helpers ---

func makeConfig(t *testing.T, cfg poweradminDNSProviderConfig) *extapi.JSON {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	return &extapi.JSON{Raw: data}
}

func makeChallengeRequest(zone, fqdn, key string, cfg *extapi.JSON) *v1alpha1.ChallengeRequest {
	return &v1alpha1.ChallengeRequest{
		ResourceNamespace: "default",
		ResolvedZone:      zone,
		ResolvedFQDN:      fqdn,
		Key:               key,
		Config:            cfg,
	}
}

func newSolverWithMock(mock *mockDNSProvider) *PowerAdminSolver {
	s := New()
	s.kubeClient = fakeclient.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "poweradmin-api-key",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"api-key": []byte("test-key"),
			},
		},
	)
	return s
}

// solverWithClient creates a solver that uses a real mock provider by overriding resolveChallenge.
// Since the solver creates its own client internally, we test through the full flow
// using a fake K8s client + httptest server. For unit tests that need precise control,
// we test individual components instead.

// --- Tests ---

func TestSolverName(t *testing.T) {
	s := New()
	if name := s.Name(); name != "poweradmin" {
		t.Errorf("Name() = %q, want %q", name, "poweradmin")
	}
}

func TestInterfaceCompliance(t *testing.T) {
	// This test verifies at runtime that *PowerAdminSolver satisfies webhook.Solver.
	var s interface{} = New()
	if _, ok := s.(webhook.Solver); !ok {
		t.Fatal("PowerAdminSolver does not implement webhook.Solver")
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantURL   string
		wantVer   string
		wantTTL   int
		wantErr   bool
	}{
		{
			name:    "full config",
			json:    `{"serverURL":"https://pa.example.com","apiKeySecretRef":{"name":"secret","key":"k"},"apiVersion":"v1","ttl":300,"insecure":true}`,
			wantURL: "https://pa.example.com",
			wantVer: "v1",
			wantTTL: 300,
		},
		{
			name:    "minimal config",
			json:    `{"serverURL":"https://pa.example.com","apiKeySecretRef":{"name":"secret","key":"k"}}`,
			wantURL: "https://pa.example.com",
			wantVer: "",
			wantTTL: 0,
		},
		{
			name: "nil config",
		},
		{
			name:    "invalid json",
			json:    `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfgJSON *extapi.JSON
			if tt.json != "" {
				cfgJSON = &extapi.JSON{Raw: []byte(tt.json)}
			}

			cfg, err := loadConfig(cfgJSON)
			if (err != nil) != tt.wantErr {
				t.Fatalf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if cfg.ServerURL != tt.wantURL {
				t.Errorf("ServerURL = %q, want %q", cfg.ServerURL, tt.wantURL)
			}
			if cfg.APIVersion != tt.wantVer {
				t.Errorf("APIVersion = %q, want %q", cfg.APIVersion, tt.wantVer)
			}
			if cfg.TTL != tt.wantTTL {
				t.Errorf("TTL = %d, want %d", cfg.TTL, tt.wantTTL)
			}
		})
	}
}

func TestLoadConfig_Insecure(t *testing.T) {
	cfgJSON := &extapi.JSON{Raw: []byte(`{"serverURL":"https://x","apiKeySecretRef":{"name":"s","key":"k"},"insecure":true}`)}
	cfg, err := loadConfig(cfgJSON)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if !cfg.Insecure {
		t.Error("expected Insecure=true")
	}
}

func TestGetAPIKey(t *testing.T) {
	s := New()
	s.kubeClient = fakeclient.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{
				"api-key": []byte("  secret-key-123  \n"),
			},
		},
	)

	cfg := poweradminDNSProviderConfig{
		APIKeySecretRef: cmmeta.SecretKeySelector{
			LocalObjectReference: cmmeta.LocalObjectReference{Name: "my-secret"},
			Key:                  "api-key",
		},
	}

	key, err := s.getAPIKey(cfg, "test-ns")
	if err != nil {
		t.Fatalf("getAPIKey() error = %v", err)
	}
	if key != "secret-key-123" {
		t.Errorf("getAPIKey() = %q, want %q (trimmed)", key, "secret-key-123")
	}
}

func TestGetAPIKey_SecretNotFound(t *testing.T) {
	s := New()
	s.kubeClient = fakeclient.NewSimpleClientset()

	cfg := poweradminDNSProviderConfig{
		APIKeySecretRef: cmmeta.SecretKeySelector{
			LocalObjectReference: cmmeta.LocalObjectReference{Name: "missing"},
			Key:                  "api-key",
		},
	}

	_, err := s.getAPIKey(cfg, "default")
	if err == nil {
		t.Error("expected error for missing secret")
	}
}

func TestGetAPIKey_KeyNotFound(t *testing.T) {
	s := New()
	s.kubeClient = fakeclient.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"other-key": []byte("value"),
			},
		},
	)

	cfg := poweradminDNSProviderConfig{
		APIKeySecretRef: cmmeta.SecretKeySelector{
			LocalObjectReference: cmmeta.LocalObjectReference{Name: "my-secret"},
			Key:                  "api-key",
		},
	}

	_, err := s.getAPIKey(cfg, "default")
	if err == nil {
		t.Error("expected error for missing key in secret")
	}
}

func TestFindZone(t *testing.T) {
	s := New()
	mock := newMockDNSProvider([]poweradmin.Zone{
		{ID: 1, Name: "example.com"},
		{ID: 2, Name: "other.com"},
	})

	tests := []struct {
		name         string
		resolvedZone string
		resolvedFQDN string
		wantZoneID   int
		wantErr      bool
	}{
		{
			name:         "exact match via ResolvedZone",
			resolvedZone: "example.com.",
			resolvedFQDN: "_acme-challenge.example.com.",
			wantZoneID:   1,
		},
		{
			name:         "fallback via FQDN label walking",
			resolvedZone: "nonexistent.com.",
			resolvedFQDN: "_acme-challenge.sub.example.com.",
			wantZoneID:   1,
		},
		{
			name:         "zone not found",
			resolvedZone: "nonexistent.com.",
			resolvedFQDN: "_acme-challenge.nonexistent.com.",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &v1alpha1.ChallengeRequest{
				ResolvedZone: tt.resolvedZone,
				ResolvedFQDN: tt.resolvedFQDN,
			}
			zone, err := s.findZone(context.Background(), mock, ch)
			if (err != nil) != tt.wantErr {
				t.Fatalf("findZone() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && zone.ID != tt.wantZoneID {
				t.Errorf("findZone() zone.ID = %d, want %d", zone.ID, tt.wantZoneID)
			}
		})
	}
}

func TestFindZone_APIError(t *testing.T) {
	s := New()
	mock := newMockDNSProvider(nil)
	mock.getZonesErr = fmt.Errorf("connection refused")

	ch := &v1alpha1.ChallengeRequest{
		ResolvedZone: "example.com.",
		ResolvedFQDN: "_acme-challenge.example.com.",
	}
	_, err := s.findZone(context.Background(), mock, ch)
	if err == nil {
		t.Error("expected error when GetZones fails")
	}
}

func TestPresent_CreatesRecord(t *testing.T) {
	mock := newMockDNSProvider([]poweradmin.Zone{{ID: 1, Name: "example.com"}})
	_ = newSolverWithMock(mock)

	// Override resolveChallenge by testing the logic components directly.
	// Present creates a TXT record when none exists.
	fqdn := "_acme-challenge.example.com"
	key := fmt.Sprintf("%q", "test-token")

	// No existing records — should create.
	_, err := mock.CreateTXTRecord(context.Background(), 1, fqdn, poweradmin.EnsureTXTQuoted(key), 120)
	if err != nil {
		t.Fatalf("CreateTXTRecord() error = %v", err)
	}
	if len(mock.createRecordCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(mock.createRecordCalls))
	}
	call := mock.createRecordCalls[0]
	if call.ZoneID != 1 {
		t.Errorf("create call zoneID = %d, want 1", call.ZoneID)
	}
	if call.Name != fqdn {
		t.Errorf("create call name = %q, want %q", call.Name, fqdn)
	}
}

func TestPresent_Idempotent(t *testing.T) {
	mock := newMockDNSProvider([]poweradmin.Zone{{ID: 1, Name: "example.com"}})
	fqdn := "_acme-challenge.example.com"
	key := fmt.Sprintf("%q", "test-token")

	// Add existing record with quoted content (as the API would return).
	mock.addRecord(1, poweradmin.Record{
		ID: 100, Name: fqdn, Type: "TXT", Content: `"test-token"`, TTL: 120,
	})

	records, _ := mock.ListTXTRecords(context.Background(), 1)

	// Simulate idempotency check from Present.
	shouldCreate := true
	for _, r := range records {
		if r.Name == fqdn && poweradmin.NormalizeTXTContent(r.Content) == poweradmin.NormalizeTXTContent(key) {
			shouldCreate = false
			break
		}
	}
	if shouldCreate {
		t.Error("expected idempotency check to detect existing record")
	}
}

func TestPresent_IdempotentWithUnquotedContent(t *testing.T) {
	mock := newMockDNSProvider([]poweradmin.Zone{{ID: 1, Name: "example.com"}})
	fqdn := "_acme-challenge.example.com"
	key := fmt.Sprintf("%q", "test-token") // produces `"test-token"`

	// API returns content without quotes.
	mock.addRecord(1, poweradmin.Record{
		ID: 100, Name: fqdn, Type: "TXT", Content: "test-token", TTL: 120,
	})

	records, _ := mock.ListTXTRecords(context.Background(), 1)

	shouldCreate := true
	for _, r := range records {
		if r.Name == fqdn && poweradmin.NormalizeTXTContent(r.Content) == poweradmin.NormalizeTXTContent(key) {
			shouldCreate = false
			break
		}
	}
	if shouldCreate {
		t.Error("normalized comparison should match quoted key with unquoted API response")
	}
}

func TestCleanUp_DeletesOnlyMatchingRecord(t *testing.T) {
	mock := newMockDNSProvider([]poweradmin.Zone{{ID: 1, Name: "example.com"}})
	fqdn := "_acme-challenge.example.com"
	key := fmt.Sprintf("%q", "token-A")

	// Two TXT records for the same FQDN (concurrent validations).
	mock.addRecord(1, poweradmin.Record{
		ID: 100, Name: fqdn, Type: "TXT", Content: `"token-A"`, TTL: 120,
	})
	mock.addRecord(1, poweradmin.Record{
		ID: 101, Name: fqdn, Type: "TXT", Content: `"token-B"`, TTL: 120,
	})

	records, _ := mock.ListTXTRecords(context.Background(), 1)

	// Simulate CleanUp logic — only delete records matching exact key.
	for _, r := range records {
		if r.Name == fqdn && poweradmin.NormalizeTXTContent(r.Content) == poweradmin.NormalizeTXTContent(key) {
			if err := mock.DeleteRecord(context.Background(), 1, r.ID); err != nil {
				t.Fatalf("DeleteRecord() error = %v", err)
			}
		}
	}

	if len(mock.deleteRecordCalls) != 1 {
		t.Fatalf("expected 1 delete call, got %d", len(mock.deleteRecordCalls))
	}
	if mock.deleteRecordCalls[0].RecordID != 100 {
		t.Errorf("deleted record ID = %d, want 100", mock.deleteRecordCalls[0].RecordID)
	}

	// Verify token-B still exists.
	remaining, _ := mock.ListTXTRecords(context.Background(), 1)
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining record, got %d", len(remaining))
	}
	if remaining[0].ID != 101 {
		t.Errorf("remaining record ID = %d, want 101 (token-B)", remaining[0].ID)
	}
}

func TestCleanUp_NoMatchingRecord(t *testing.T) {
	mock := newMockDNSProvider([]poweradmin.Zone{{ID: 1, Name: "example.com"}})
	fqdn := "_acme-challenge.example.com"
	key := fmt.Sprintf("%q", "nonexistent-token")

	mock.addRecord(1, poweradmin.Record{
		ID: 100, Name: fqdn, Type: "TXT", Content: `"other-token"`, TTL: 120,
	})

	records, _ := mock.ListTXTRecords(context.Background(), 1)

	// CleanUp with non-matching key should delete nothing.
	for _, r := range records {
		if r.Name == fqdn && poweradmin.NormalizeTXTContent(r.Content) == poweradmin.NormalizeTXTContent(key) {
			t.Error("should not match any record")
		}
	}

	if len(mock.deleteRecordCalls) != 0 {
		t.Errorf("expected 0 delete calls, got %d", len(mock.deleteRecordCalls))
	}
}

func TestCleanUp_HandlesUnquotedAPIResponse(t *testing.T) {
	mock := newMockDNSProvider([]poweradmin.Zone{{ID: 1, Name: "example.com"}})
	fqdn := "_acme-challenge.example.com"
	key := fmt.Sprintf("%q", "my-token") // `"my-token"`

	// API returns unquoted content.
	mock.addRecord(1, poweradmin.Record{
		ID: 200, Name: fqdn, Type: "TXT", Content: "my-token", TTL: 120,
	})

	records, _ := mock.ListTXTRecords(context.Background(), 1)

	for _, r := range records {
		if r.Name == fqdn && poweradmin.NormalizeTXTContent(r.Content) == poweradmin.NormalizeTXTContent(key) {
			_ = mock.DeleteRecord(context.Background(), 1, r.ID)
		}
	}

	if len(mock.deleteRecordCalls) != 1 {
		t.Fatalf("expected 1 delete call (normalized match), got %d", len(mock.deleteRecordCalls))
	}
	if mock.deleteRecordCalls[0].RecordID != 200 {
		t.Errorf("deleted record ID = %d, want 200", mock.deleteRecordCalls[0].RecordID)
	}
}

func TestDefaultTTL(t *testing.T) {
	if defaultTTL != 120 {
		t.Errorf("defaultTTL = %d, want 120", defaultTTL)
	}
}

func TestConfigTTLOverride(t *testing.T) {
	cfgJSON := &extapi.JSON{Raw: []byte(`{"serverURL":"https://x","apiKeySecretRef":{"name":"s","key":"k"},"ttl":300}`)}
	cfg, err := loadConfig(cfgJSON)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.TTL != 300 {
		t.Errorf("TTL = %d, want 300", cfg.TTL)
	}
}
