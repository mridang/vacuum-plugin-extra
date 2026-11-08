package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// ExampleBodyKeyCase returns the extra-example-body-key-case rule.
//
// # What it checks
//
// Property names inside example bodies must follow the configured case style.
// The examples checked are the example and examples[*].value of every JSON media
// type in request bodies and responses, inline or under components. Nested
// objects and arrays are walked, and every mapping key must follow
// the configured convention.
//
// # Why
//
// Examples are copied into client code, tests and documentation. An example
// that spells a property differently from the schema, or from the rest of the
// API, teaches the wrong convention and hides inconsistencies that
// schema-only naming rules cannot see.
//
// # Where it comes from
//
// apilint (body-capitalization).
//
// # Options
//
//   - casing (string or list, default "snake_case"): kebab-case, snake_case,
//     camelCase, PascalCase, SCREAMING_SNAKE_CASE, SCREAMING-KEBAB-CASE or
//     lowercase. A key is accepted when it follows any listed convention, an
//     empty list disables the rule and an unknown value is reported.
//
// # Bad
//
//	responses:
//	  "200":
//	    description: ok
//	    content:
//	      application/json:
//	        example:
//	          firstName: Ada
//
// # Good
//
//	example:
//	  first_name: Ada
//
// # Known false positives and limitations
//
// Words are split on separators and case changes, so keys that are
// case-sensitive identifiers (map keys such as user ids or locale tags) are
// flagged; use a style that fits or suppress the rule. Examples given as a
// string holding JSON, and $ref examples, are not inspected.
func ExampleBodyKeyCase() Rule {
	return New(Spec{
		Name:        "example-body-key-case",
		Description: "Property names in example bodies must follow the configured case style.",
		HowToFix:    "Rename the example property so that it follows the configured case style.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			casingOption("Case convention of the example property names", casing.Snake),
		},
		Check: WithCasing(checkExampleBodyKeyCase),
	})
}

type exampleBodyKeyCaseExample struct {
	node *yaml.Node
	path string
}

// checkExampleBodyKeyCase walks every example body and compares key casing.
func checkExampleBodyKeyCase(call *Call, allowed casing.Set) []Finding {
	return seq.FlatMapSlice(exampleBodyKeyCaseExamples(call.Root), func(example exampleBodyKeyCaseExample) []Finding {
		return exampleBodyKeyCaseWalk(example.node, example.path, allowed)
	})
}

// exampleBodyKeyCaseWalk reports the keys of one example value that do not follow the casing.
func exampleBodyKeyCaseWalk(value *yaml.Node, path string, allowed casing.Set) []Finding {
	return slices.Concat(
		exampleBodyKeyCaseKeys(value, path, allowed),
		seq.FlatMap(oas.Walk(value, path), func(node oas.Node) []Finding {
			return exampleBodyKeyCaseKeys(node.Node, node.Path, allowed)
		}),
	)
}

// exampleBodyKeyCaseKeys reports the direct keys of a mapping that do not follow the casing.
func exampleBodyKeyCaseKeys(mapping *yaml.Node, path string, allowed casing.Set) []Finding {
	return seq.FlatMapPairs(oas.Pairs(mapping), func(key, _ *yaml.Node) []Finding {
		return When(!allowed.Matches(key.Value), func() Finding {
			return Finding{
				Node: key,
				Path: oas.Join(path, key.Value),
				Message: "example body property '" + key.Value + "' must be " + allowed.String() +
					" (for example '" + allowed.Convert(key.Value) + "')",
			}
		})
	})
}

// exampleBodyKeyCaseExamples collects the example values of all JSON media types.
func exampleBodyKeyCaseExamples(root *yaml.Node) []exampleBodyKeyCaseExample {
	media := func(owner *yaml.Node, path string) []exampleBodyKeyCaseExample {
		return seq.FlatMapSlice(
			oas.MediaTypes(oas.Get(owner, "content"), oas.Join(path, "content")),
			func(media oas.MediaType) []exampleBodyKeyCaseExample {
				if !strings.Contains(strings.ToLower(media.Name), "json") {
					return nil
				}

				return exampleBodyKeyCaseMedia(media)
			},
		)
	}

	return slices.Concat(
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []exampleBodyKeyCaseExample {
			return slices.Concat(
				media(oas.Get(operation.Node, "requestBody"), oas.Join(operation.Path, "requestBody")),
				seq.FlatMapPairs(
					oas.Pairs(oas.Get(operation.Node, "responses")),
					func(key, response *yaml.Node) []exampleBodyKeyCaseExample {
						return media(response, oas.JoinAll(operation.Path, "responses", key.Value))
					},
				),
			)
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "requestBodies")),
			func(key, body *yaml.Node) []exampleBodyKeyCaseExample {
				return media(body, oas.JoinAll("$.components.requestBodies", key.Value))
			},
		),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "responses")),
			func(key, response *yaml.Node) []exampleBodyKeyCaseExample {
				return media(response, oas.JoinAll("$.components.responses", key.Value))
			},
		),
	)
}

// exampleBodyKeyCaseMedia returns the example and examples values of one media type.
func exampleBodyKeyCaseMedia(media oas.MediaType) []exampleBodyKeyCaseExample {
	return slices.Concat(
		exampleBodyKeyCaseOf(oas.Get(media.Node, "example"), oas.Join(media.Path, "example")),
		seq.FlatMapPairs(
			oas.Pairs(oas.Get(media.Node, "examples")),
			func(name, entry *yaml.Node) []exampleBodyKeyCaseExample {
				return exampleBodyKeyCaseOf(
					oas.Get(entry, "value"),
					oas.Join(oas.JoinAll(media.Path, "examples", name.Value), "value"),
				)
			},
		),
	)
}

// exampleBodyKeyCaseOf wraps an example value, or returns nothing when it is absent.
func exampleBodyKeyCaseOf(node *yaml.Node, path string) []exampleBodyKeyCaseExample {
	if node == nil {
		return nil
	}

	return []exampleBodyKeyCaseExample{{node: node, path: path}}
}
