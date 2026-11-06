// Package rules contains the extra lint rules for the vacuum OpenAPI linter.
//
// Every rule lives in its own file and is exposed as a constructor returning a
// [Rule]. A Rule is a vacuum custom function (it implements model.RuleFunction)
// that also knows the metadata a ruleset entry needs: the rule identifier, the
// severity, the JSONPath it is given and the how-to-fix message. The full
// catalogue is returned by [All].
//
// Rule identifiers are kebab-case and prefixed with "extra-" so that they never
// clash with the rules vacuum ships. The name of the custom function that backs
// a rule is identical to the rule identifier.
//
// The ready-to-use ruleset (ruleset.yaml in the repository root) is generated
// from the catalogue with:
//
//	go generate ./...
package rules
