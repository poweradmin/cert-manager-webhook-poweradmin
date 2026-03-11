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

// challengeContext holds resolved state shared by Present and CleanUp.
type challengeContext struct {
	cfg     poweradminDNSProviderConfig
	client  poweradmin.DNSProvider
	zone    *poweradmin.Zone
	fqdn    string
	txtKey  string
	records []poweradmin.Record
}

// resolveChallenge performs the common setup for Present and CleanUp:
// decode config, create API client, find zone, resolve FQDN, list TXT records.
func (s *PowerAdminSolver) resolveChallenge(ch *v1alpha1.ChallengeRequest) (*challengeContext, error) {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return nil, err
	}

	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("serverURL must be specified in solver config")
	}

	apiKey, err := s.getAPIKey(cfg, ch.ResourceNamespace)
	if err != nil {
		return nil, err
	}

	client, err := poweradmin.NewClient(cfg.ServerURL, apiKey, cfg.APIVersion, cfg.Insecure)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	zone, err := s.findZone(ctx, client, ch)
	if err != nil {
		return nil, err
	}

	fqdn := strings.TrimSuffix(ch.ResolvedFQDN, ".")
	txtKey := fmt.Sprintf("%q", ch.Key)

	records, err := client.ListTXTRecords(ctx, zone.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list TXT records for zone %q: %w", zone.Name, err)
	}

	return &challengeContext{
		cfg:     cfg,
		client:  client,
		zone:    zone,
		fqdn:    fqdn,
		txtKey:  txtKey,
		records: records,
	}, nil
}

func (s *PowerAdminSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cc, err := s.resolveChallenge(ch)
	if err != nil {
		return err
	}

	// Check idempotency: if a matching record already exists, skip creation.
	for _, r := range cc.records {
		if r.Name == cc.fqdn && r.Content == cc.txtKey {
			return nil
		}
	}

	ttl := cc.cfg.TTL
	if ttl <= 0 {
		ttl = defaultTTL
	}

	_, err = cc.client.CreateTXTRecord(context.Background(), cc.zone.ID, cc.fqdn, cc.txtKey, ttl)
	if err != nil {
		return fmt.Errorf("failed to create TXT record for %q in zone %q: %w", cc.fqdn, cc.zone.Name, err)
	}

	return nil
}

func (s *PowerAdminSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cc, err := s.resolveChallenge(ch)
	if err != nil {
		return err
	}

	for _, r := range cc.records {
		if r.Name == cc.fqdn && r.Content == cc.txtKey {
			if err := cc.client.DeleteRecord(context.Background(), cc.zone.ID, r.ID); err != nil {
				return fmt.Errorf("failed to delete TXT record %d for %q in zone %q: %w", r.ID, cc.fqdn, cc.zone.Name, err)
			}
		}
	}

	return nil
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
