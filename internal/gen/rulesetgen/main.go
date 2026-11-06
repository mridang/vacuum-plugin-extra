// Command rulesetgen writes ruleset.yaml, the vacuum ruleset that lists every
// rule of the catalogue.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mridang/vacuum-plugin-extra/internal/ruleset"
)

const fileMode = 0o644

func main() {
	out := flag.String("out", "ruleset.yaml", "path of the ruleset file to write")
	flag.Parse()

	if err := write(*out); err != nil {
		fmt.Fprintln(os.Stderr, "rulesetgen:", err)
		os.Exit(1)
	}
}

func write(path string) error {
	data, err := ruleset.Render()
	if err != nil {
		return fmt.Errorf("render ruleset: %w", err)
	}

	if err = os.WriteFile(path, data, fileMode); err != nil {
		return fmt.Errorf("write ruleset: %w", err)
	}

	return nil
}
