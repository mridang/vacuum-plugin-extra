package extra

import (
	_ "embed" // Required for the go:embed directive below.
	"maps"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin"

	"github.com/mridang/vacuum-plugin-extra/rules"
)

// rulesetYAML is the generated ruleset that lists every rule of the catalogue.
//
//go:embed ruleset.yaml

var rulesetYAML []byte

// Rules returns every rule of the catalogue, ordered by identifier.
func Rules() []rules.Rule {
	return rules.All()
}

// Functions returns the custom functions of the catalogue keyed by function
// name, ready to be passed to vacuum as RuleSetExecution.CustomFunctions.
func Functions() map[string]model.RuleFunction {
	all := rules.All()

	return maps.Collect(func(yield func(string, model.RuleFunction) bool) {
		for _, rule := range all {
			if !yield(rule.ID(), rule) {
				return
			}
		}
	})
}

// Register adds every rule of the catalogue to a vacuum plugin manager.
func Register(manager *plugin.Manager) {
	for name, function := range Functions() {
		manager.RegisterFunction(name, function)
	}
}

// Ruleset returns the contents of ruleset.yaml, the ruleset that enables every
// rule of the catalogue with its default severity and options.
func Ruleset() []byte {
	return append([]byte(nil), rulesetYAML...)
}
