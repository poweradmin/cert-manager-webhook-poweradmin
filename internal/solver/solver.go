package solver

import (
	"context"
	"fmt"
	"strings"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/cert-manager/cert-manager-webhook-poweradmin/internal/poweradmin"
)

const defaultTTL = 120

// PowerAdminSolver implements the cert-manager webhook.Solver interface
// for PowerAdmin DNS provider.
type PowerAdminSolver struct {
	kubeClient kubernetes.Interface
}

// New creates a new PowerAdminSolver.
func New() *PowerAdminSolver {
	return &PowerAdminSolver{}
}

func (s *PowerAdminSolver) Name() string {
	return "poweradmin"
}

func (s *PowerAdminSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	s.kubeClient = cl
	return nil
}

func (s *PowerAdminSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, client, err := s.newClientFromChallenge(ch)
	if err != nil {
		return err
	}

	ctx := context.Background()

	zone, err := s.findZone(ctx, client, ch)
	if err != nil {
		return err
	}

	fqdn := strings.TrimSuffix(ch.ResolvedFQDN, ".")
	quotedKey := fmt.Sprintf("%q", ch.Key)

	// Check idempotency: if a matching record already exists, skip creation.
	records, err := client.ListTXTRecords(ctx, zone.ID)
	if err != nil {
		return fmt.Errorf("failed to list TXT records for zone %q: %w", zone.Name, err)
	}
	for _, r := range records {
		if r.Name == fqdn && r.Content == quotedKey {
			return nil
		}
	}

	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = defaultTTL
	}

	_, err = client.CreateTXTRecord(ctx, zone.ID, fqdn, quotedKey, ttl)
	if err != nil {
		return fmt.Errorf("failed to create TXT record for %q in zone %q: %w", fqdn, zone.Name, err)
	}

	return nil
}

func (s *PowerAdminSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	_, client, err := s.newClientFromChallenge(ch)
	if err != nil {
		return err
	}

	ctx := context.Background()

	zone, err := s.findZone(ctx, client, ch)
	if err != nil {
		return err
	}

	fqdn := strings.TrimSuffix(ch.ResolvedFQDN, ".")
	quotedKey := fmt.Sprintf("%q", ch.Key)

	records, err := client.ListTXTRecords(ctx, zone.ID)
	if err != nil {
		return fmt.Errorf("failed to list TXT records for zone %q: %w", zone.Name, err)
	}

	for _, r := range records {
		if r.Name == fqdn && r.Content == quotedKey {
			if err := client.DeleteRecord(ctx, zone.ID, r.ID); err != nil {
				return fmt.Errorf("failed to delete TXT record %d for %q in zone %q: %w", r.ID, fqdn, zone.Name, err)
			}
		}
	}

	return nil
}

// newClientFromChallenge decodes config, fetches the API key from K8s Secret,
// and creates a PowerAdmin API client.
func (s *PowerAdminSolver) newClientFromChallenge(ch *v1alpha1.ChallengeRequest) (poweradminDNSProviderConfig, poweradmin.DNSProvider, error) {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return cfg, nil, err
	}

	if cfg.ServerURL == "" {
		return cfg, nil, fmt.Errorf("serverURL must be specified in solver config")
	}

	apiKey, err := s.getAPIKey(cfg, ch.ResourceNamespace)
	if err != nil {
		return cfg, nil, err
	}

	client, err := poweradmin.NewClient(cfg.ServerURL, apiKey, cfg.APIVersion, cfg.Insecure)
	if err != nil {
		return cfg, nil, err
	}

	return cfg, client, nil
}

// getAPIKey fetches the API key from the Kubernetes Secret referenced in the config.
func (s *PowerAdminSolver) getAPIKey(cfg poweradminDNSProviderConfig, namespace string) (string, error) {
	secret, err := s.kubeClient.CoreV1().Secrets(namespace).Get(
		context.Background(),
		cfg.APIKeySecretRef.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to read API key secret %s/%s: %w", namespace, cfg.APIKeySecretRef.Name, err)
	}

	keyBytes, ok := secret.Data[cfg.APIKeySecretRef.Key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %s/%s", cfg.APIKeySecretRef.Key, namespace, cfg.APIKeySecretRef.Name)
	}

	return strings.TrimSpace(string(keyBytes)), nil
}

// findZone resolves the DNS zone in PowerAdmin for the given challenge.
func (s *PowerAdminSolver) findZone(ctx context.Context, client poweradmin.DNSProvider, ch *v1alpha1.ChallengeRequest) (*poweradmin.Zone, error) {
	// Fetch all zones once and search locally to avoid redundant API calls
	// and to properly surface API errors (auth failures, network issues).
	zones, err := client.GetZones(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list zones from PowerAdmin: %w", err)
	}

	zoneMap := make(map[string]*poweradmin.Zone, len(zones))
	for i := range zones {
		zoneMap[zones[i].Name] = &zones[i]
	}

	// Try ch.ResolvedZone first (provided by cert-manager).
	zoneName := strings.TrimSuffix(ch.ResolvedZone, ".")
	if zone, ok := zoneMap[zoneName]; ok {
		return zone, nil
	}

	// Fallback: walk up domain labels of ch.ResolvedFQDN.
	fqdn := strings.TrimSuffix(ch.ResolvedFQDN, ".")
	parts := strings.Split(fqdn, ".")
	for i := 1; i < len(parts); i++ {
		candidate := strings.Join(parts[i:], ".")
		if zone, ok := zoneMap[candidate]; ok {
			return zone, nil
		}
	}

	return nil, fmt.Errorf("could not find zone for domain %q in PowerAdmin", fqdn)
}
