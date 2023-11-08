package main

import (
	"os"
	"testing"

	dns "github.com/cert-manager/cert-manager/test/acme"
	"github.com/dnaeon/cert-manager-webhook-bind9/bind"
)

func TestRunsSuite(t *testing.T) {
	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.
	//

	// Adjust the path to the ACME helper script during testing.
	// Also, configure the ACME helper script to use the test
	// nameserver during the conformance tests.
	os.Setenv("USE_NAMESERVER", "172.16.0.3")
	solver := bind.NewSolver()
	solver.AcmeHelperScript = "./scripts/acme-challenge-helper.sh"
	fixture := dns.NewFixture(solver,
		dns.SetResolvedZone("example.com."),
		dns.SetAllowAmbientCredentials(false),
		dns.SetManifestPath("testdata/cert-manager-webhook-bind9"),
		dns.SetDNSServer("172.16.0.3:53"),
	)

	fixture.RunConformance(t)
}
