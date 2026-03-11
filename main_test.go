package main

import (
	"os"
	"testing"

	acmetest "github.com/cert-manager/cert-manager/test/acme"

	"github.com/cert-manager/cert-manager-webhook-poweradmin/internal/solver"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
)

func TestRunsSuite(t *testing.T) {
	// Uncomment and configure for integration testing against a real PowerAdmin instance.
	// Requires TEST_ZONE_NAME environment variable to be set.

	if zone == "" {
		t.Skip("TEST_ZONE_NAME not set, skipping conformance tests")
	}

	fixture := acmetest.NewFixture(solver.New(),
		acmetest.SetResolvedZone(zone),
		acmetest.SetAllowAmbientCredentials(false),
		acmetest.SetManifestPath("testdata/poweradmin-solver"),
	)
	fixture.RunBasic(t)
	fixture.RunExtended(t)
}
