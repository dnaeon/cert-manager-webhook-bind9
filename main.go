package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
)

var GroupName = os.Getenv("GROUP_NAME")

// ErrNoAllowedZonesConfigured is returned when the solver was not
// configured with a list of allowed zones.
var ErrNoAllowedZonesConfigured = errors.New("no allowed zones configured")

// ErrNoTSIGKeyConfigured is returned when the solver was not
// configured with a TSIG key.
var ErrNoTSIGKeyConfigured = errors.New("no TSIG key configured")

// DefaultTTL represents the default TTL value to set for new records,
// unless specified in the configuration
const DefaultTTL = 300

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

	// TTL is the time-to-live to set on the newly created TXT
	// records
	TTL int `json:"ttl"`

	// AllowedZones is the list of zones that the solver is
	// allowed to manage
	AllowedZones []string `json:"allowedZones"`

	// tsigKey represents the raw TSIG key after fetching it from
	// the secret store
	tsigKey []byte
}

// Name implements the webhook.Solver interface
func (b *bindProviderSolver) Name() string {
	return "bind9"
}

// Present implements the webhook.Solver interface by creating the
// respective TXT records
func (b *bindProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	// TODO: Handle duplicate records
	cfg, err := b.loadConfig(ch.Config, ch.ResourceNamespace)
	if err != nil {
		return err
	}

	klog.InfoS("Solving challenge", "dnsName", ch.DNSName, "resolvedZone", ch.ResolvedZone, "resolvedFQDN", ch.ResolvedFQDN)

	// The zone must be in the list of zones we are allowing
	resolvedZone := ch.ResolvedZone
	if !slices.Contains(cfg.AllowedZones, resolvedZone) {
		return fmt.Errorf("Zone %s is not in the allowed-zones list", resolvedZone)
	}

	// TODO: Add the actual logic here
	return nil
}

// CleanUp implements the webhook.Solver interface and deletes the
// respective TXT records
func (b *bindProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	// TODO: Add the actual logic here
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
func (b *bindProviderSolver) loadConfig(cfgJSON *extapi.JSON, namespace string) (bindProviderConfig, error) {
	cfg := bindProviderConfig{
		TTL: DefaultTTL,
	}

	// We require TSIG key and allowed zones to be configured
	if cfgJSON == nil {
		return cfg, errors.New("TSIG key and allowed zones must be configured")
	}

	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	// Validate the configuration and set sane defaults, if
	// needed.
	if cfg.TTL == 0 {
		cfg.TTL = DefaultTTL
	}

	if cfg.AllowedZones == nil {
		return cfg, ErrNoAllowedZonesConfigured
	}

	if cfg.TSIGKeyRef.LocalObjectReference.Name == "" {
		return cfg, ErrNoTSIGKeyConfigured
	}

	// Load the TSIG key
	ctx := context.Background()
	getOpts := metav1.GetOptions{}
	tsigSecret, err := b.client.CoreV1().Secrets(namespace).Get(ctx, cfg.TSIGKeyRef.LocalObjectReference.Name, getOpts)

	if err != nil {
		return cfg, fmt.Errorf("failed to load TSIG key from %s/%s: %v", namespace, cfg.TSIGKeyRef.LocalObjectReference.Name, err)
	}

	secretData, ok := tsigSecret.Data[cfg.TSIGKeyRef.Key]
	if !ok {
		return cfg, fmt.Errorf("TSIG key %s not found in %s/%s", cfg.TSIGKeyRef.Key, cfg.TSIGKeyRef.LocalObjectReference.Name, namespace)
	}

	cfg.tsigKey = secretData

	return cfg, nil
}
