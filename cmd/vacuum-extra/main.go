// Command vacuum-extra lints an OpenAPI document with the extra rules, using
// vacuum's engine. It is a single static binary: no Go plugin is needed.
//
//	vacuum-extra openapi.yaml
//	vacuum-extra -format json -fail-on warn openapi.yaml
//	vacuum-extra -list
package main

import (
	"os"

	"github.com/mridang/vacuum-plugin-extra/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
