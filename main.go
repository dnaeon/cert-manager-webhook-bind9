// TODO: handle duplicate records in Present
package main

import (
	"fmt"
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	"github.com/dnaeon/cert-manager-webhook-bind9/bind"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		fmt.Fprintf(os.Stderr, "GROUP_NAME must be specified\n")
		os.Exit(1)
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName,
		&bind.BindProviderSolver{},
	)
}
