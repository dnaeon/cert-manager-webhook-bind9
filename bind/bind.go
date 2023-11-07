package bind

import (
	restclient "k8s.io/client-go/rest"

	whapi "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
)

// BINDSolver is an implementation of webhook.Solver
type BINDSolver struct {
}

// Name implements the webhook.Solver interface
func (b *BINDSolver) Name() string {

	// TODO
	return ""
}

// Present creates the respective TXT records as part of the challenge
// request.
func (b *BINDSolver) Present(ch *whapi.ChallengeRequest) error {

	// TODO
	return nil
}

// CleanUp removes the respective TXT records for the given challenge
// request
func (b *BINDSolver) CleanUp(ch *whapi.ChallengeRequest) error {

	return nil
	// TODO
}

// Initialize initializes the webhook
func (b *BINDSolver) Initialize(kubeClientConfig *restclient.Config, stopCh <-chan struct{}) error {

	// TODO
	return nil
}
