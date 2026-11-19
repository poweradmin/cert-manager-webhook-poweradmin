//go:build integration

package main

import (
	"os"
	"testing"

	acmetest "github.com/cert-manager/cert-manager/test/acme"

	"github.com/cert-manager/cert-manager-webhook-poweradmin/internal/solver"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
	// dnsServer (host:port) checks propagation against a local PowerDNS;
	// the fixture otherwise queries 8.8.8.8, which cannot see private zones.
	dnsServer = os.Getenv("TEST_DNS_SERVER")
)

func TestRunsSuite(t *testing.T) {
	if zone == "" {
		t.Skip("TEST_ZONE_NAME not set, skipping conformance tests")
	}

	opts := []acmetest.Option{
		acmetest.SetResolvedZone(zone),
		acmetest.SetAllowAmbientCredentials(false),
		acmetest.SetManifestPath("testdata/poweradmin-solver"),
	}
	if dnsServer != "" {
		opts = append(opts, acmetest.SetDNSServer(dnsServer), acmetest.SetUseAuthoritative(false))
	}

	fixture := acmetest.NewFixture(solver.New(), opts...)
	fixture.RunBasic(t)
	fixture.RunExtended(t)
}
