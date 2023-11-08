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

	cmd.RunWebhookServer(GroupName,
		&bind.BindProviderSolver{},
	)
}
