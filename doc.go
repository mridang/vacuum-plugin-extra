// Package extra is the root of the vacuum-plugin-extra rule package: a
// catalogue of extra lint rules for the vacuum OpenAPI linter that vacuum does
// not ship.
//
// Subpackages:
//   - rules: one file per rule, plus the [rules.Rule] type that adapts a check
//     to vacuum's custom function interface.
//
// The ready-to-use ruleset is ruleset.yaml. It is generated from the catalogue
// and embedded in this package; regenerate it with:
//
//	go generate ./...
//
//go:generate go run ./internal/gen/rulesetgen -out ruleset.yaml
package extra
