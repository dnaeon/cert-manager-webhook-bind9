package main

import (
	"encoding/json"
	"fmt"
	"os"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName,
		&bindProviderSolver{},
	)
}

// bindSolver implements the webhook.Solver interface
type bindProviderSolver struct {
	client *kubernetes.Clientset
}

// bindProviderConfig represents the configuration for the BIND solver.
type bindProviderConfig struct {
	// Change the two fields below according to the format of the configuration
	// to be decoded.
	// These fields will be set by users in the
	// `issuer.spec.acme.dns01.providers.webhook.config` field.

	// Email           string `json:"email"`
	// APIKeySecretRef cmmeta.SecretKeySelector `json:"apiKeySecretRef"`

	// TSIGKeyRef is the shared TSIG key used to dynamically
	// update the DNS records.
	TSIGKeyRef cmmeta.SecretKeySelector `json:"tsigKeyRef"`

	// ZoneName is the DNS zone name we are managing for
	// _acme-challenge TXT records.
	ZoneName string `json:"zoneName"`

	// TTL is the time-to-live to set on the newly created TXT
	// records
	TTL int `json:"ttl"`

	// AllowedZones is the list of zones that the solver is
	// allowed to manage
	AllowedZones []string `json:"allowedZones"`
}

// Name implements the webhook.Solver interface
func (b *bindProviderSolver) Name() string {
	return "bind9"
}

// Present implements the webhook.Solver interface by creating the
// respective TXT records
func (b *bindProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}

	// TODO: do something more useful with the decoded configuration
	fmt.Printf("Decoded configuration %v", cfg)

	// TODO: add code that sets a record in the DNS provider's console
	return nil
}

// CleanUp implements the webhook.Solver interface and deletes the
// respective TXT records
func (b *bindProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	// TODO: add code that deletes a record from the DNS provider's console
	return nil
}

// Initialize initializes the BIND solver
func (b *bindProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	b.client = cl

	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (bindProviderConfig, error) {
	cfg := bindProviderConfig{}
	if cfgJSON == nil {
		return cfg, nil
	}

	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}
