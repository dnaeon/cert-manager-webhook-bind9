// Package bind provides an implementation of DNS-01 Solver as a Cert
// Manager webhook.

package bind

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
)

// ErrNoAllowedZonesConfigured is returned when the solver was not
// configured with a list of allowed zones.
var ErrNoAllowedZonesConfigured = errors.New("no allowed zones configured")

// ErrNoTSIGKeyConfigured is returned when the solver was not
// configured with a TSIG key.
var ErrNoTSIGKeyConfigured = errors.New("no TSIG key configured")

// DefaultTTL represents the default TTL value to set for new records,
// unless specified in the configuration
const DefaultTTL = 300

// BindSolver implements the webhook.Solver interface
type BindProviderSolver struct {
	client *kubernetes.Clientset

	// The helper script we use to create and delete the ACME
	// Challenge TXT records.
	AcmeHelperScript string
}

// NewSolver creates a new BIND9 DNS-01 solver
func NewSolver() *BindProviderSolver {
	b := &BindProviderSolver{
		AcmeHelperScript: "acme-challenge-helper.sh",
	}

	return b
}

// bindProviderConfig represents the configuration for the BIND solver.
type BindProviderConfig struct {
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

// dumpTSIGKey dumps the contents of the TSIG key in the given path
// and returns the file.  Callers of this method must ensure to delete
// the file when no longer needed.
func (bpc *BindProviderConfig) dumpTSIGKey(path string) (*os.File, error) {
	tmpFile, err := os.CreateTemp(path, "tsig-key")
	if err != nil {
		return nil, err
	}

	if _, err := tmpFile.Write(bpc.tsigKey); err != nil {
		return nil, err
	}

	return tmpFile, nil
}

// Name implements the webhook.Solver interface
func (b *BindProviderSolver) Name() string {
	return "bind9"
}

// Present implements the webhook.Solver interface by creating the
// respective TXT records
func (b *BindProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := b.loadConfig(ch.Config, ch.ResourceNamespace)
	if err != nil {
		return err
	}

	// The zone must be in the list of zones we are allowing
	zoneName := ch.ResolvedZone
	if !slices.Contains(cfg.AllowedZones, zoneName) {
		return fmt.Errorf("Zone %s is not in the allowed-zones list", zoneName)
	}

	// Dump the TSIG key locally, so that we can pass it to
	// the helper scripts. Make sure to delete it afterwards.
	tsigFile, err := cfg.dumpTSIGKey("")
	if err != nil {
		return fmt.Errorf("failed to dump TSIG key: %s", err)
	}
	defer os.Remove(tsigFile.Name())

	// Call our helper script here to create the respective TXT
	// records as part of the DNS-01 challenge
	cmd := exec.Command(b.AcmeHelperScript, "create", zoneName, ch.ResolvedFQDN, tsigFile.Name(), strconv.Itoa(cfg.TTL), ch.Key)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create TXT record %s: %s", ch.ResolvedFQDN, err)
	}

	return nil
}

// CleanUp implements the webhook.Solver interface and deletes the
// respective TXT records
func (b *BindProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := b.loadConfig(ch.Config, ch.ResourceNamespace)
	if err != nil {
		return err
	}

	// The zone must be in the list of zones we are allowing
	zoneName := ch.ResolvedZone
	if !slices.Contains(cfg.AllowedZones, zoneName) {
		return fmt.Errorf("Zone %s is not in the allowed-zones list", zoneName)
	}

	// Dump the TSIG key locally, so that we can pass it to
	// the helper scripts. Make sure to delete it afterwards.
	tsigFile, err := cfg.dumpTSIGKey("")
	if err != nil {
		return fmt.Errorf("failed to dump TSIG key: %s", err)
	}
	defer os.Remove(tsigFile.Name())

	// Call our helper script here to delete the respective TXT
	// record
	cmd := exec.Command(b.AcmeHelperScript, "delete", zoneName, ch.ResolvedFQDN, tsigFile.Name(), strconv.Itoa(cfg.TTL), ch.Key)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete TXT record %s: %s", ch.ResolvedFQDN, err)
	}

	return nil
}

// Initialize initializes the BIND solver
func (b *BindProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	b.client = cl

	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func (b *BindProviderSolver) loadConfig(cfgJSON *extapi.JSON, namespace string) (BindProviderConfig, error) {
	cfg := BindProviderConfig{
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
