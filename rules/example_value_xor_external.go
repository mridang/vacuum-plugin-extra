package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// ExampleValueXorExternal returns the extra-example-value-xor-external rule.
//
// # What it checks
//
// Every Example Object must define exactly one of value and externalValue.
// Example Objects are the entries of components.examples and of the examples
// maps of media types, parameters and headers, wherever they appear in the
// document. An example that defines both is reported, and so is one that defines
// neither. The OpenAPI 3.2 keywords dataValue and serializedValue count as
// inline values, like value.
//
// # Why
//
// The specification states that value and externalValue are mutually exclusive.
// When both are present tools disagree about which one wins, and an example with
// neither carries no sample at all.
//
// # Where it comes from
//
// Spectral (oas3-examples-value-or-externalValue), Scalar, speccy and the
// OpenAPI Specification linter of oas-kit (example-value-or-externalValue),
// KaiZen (example-single-value) and Redocly (no-example-value-and-externalValue).
// The tools differ in whether an example with neither keyword is an error, which
// is why both directions are options.
//
// # Options
//
//   - forbidBoth: report examples that define both value and externalValue.
//     Default: true.
//   - requireOne: report examples that define neither. Default: true.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              examples:
//	                both:
//	                  value: {name: Rex}
//	                  externalValue: https://example.org/rex.json
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              examples:
//	                inline:
//	                  value: {name: Rex}
//
// # Known false positives / limitations
//
// Examples that use only a $ref are resolved before they are checked; an example
// whose reference cannot be resolved is skipped. Examples nested in vendor
// extensions are not visited.
func ExampleValueXorExternal() Rule {
	return New(Spec{
		Name:        "example-value-xor-external",
		Description: "An Example Object must define exactly one of value and externalValue.",
		HowToFix:    "Keep either value or externalValue in the example, not both and not neither.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Formats:     []string{formatOAS3},
		Options: []Option{
			{
				Name:        "forbidBoth",
				Description: "Report examples that define both value and externalValue.",
				Default:     true,
			},
			{
				Name:        "requireOne",
				Description: "Report examples that define neither value nor externalValue.",
				Default:     true,
			},
		},
		Check: OnDocument(checkExampleValueXorExternal),
	})
}

func checkExampleValueXorExternal(call *Call) []Finding {
	examples := exampleValueXorExternalExamples(call.Root)

	return seq.FlatMapIndexed(examples, func(i int, example oas.Node) []Finding {
		first := slices.IndexFunc(examples, func(other oas.Node) bool {
			return other.Node == example.Node
		}) == i
		message := exampleValueXorExternalMessage(call, example.Node)

		return When(first && message != "", func() Finding {
			return Finding{Node: example.Node, Path: example.Path, Message: message}
		})
	})
}

// exampleValueXorExternalMessage returns the violation message for an Example
// Object, or the empty string when the example is valid.
func exampleValueXorExternalMessage(call *Call, example *yaml.Node) string {
	if !oas.IsMap(example) {
		return ""
	}

	inline := oas.Has(example, "value") || oas.Has(example, "dataValue") || oas.Has(example, "serializedValue")
	external := oas.Has(example, "externalValue")

	switch {
	case inline && external && call.Bool("forbidBoth"):
		return "example must not define both value and externalValue"
	case !inline && !external && call.Bool("requireOne") && oas.Ref(example) == "":
		return "example must define either value or externalValue"
	default:
		return ""
	}
}

// exampleValueXorExternalExamples collects every Example Object with its path.
func exampleValueXorExternalExamples(root *yaml.Node) []oas.Node {
	return slices.Concat(
		exampleValueXorExternalAt(root, oas.Dig(root, "components", "examples"), "$.components.examples"),
		seq.FlatMapSlice(exampleValueXorExternalOwners(root), func(owner oas.Node) []oas.Node {
			return exampleValueXorExternalAt(root, oas.Get(owner.Node, "examples"), oas.Join(owner.Path, "examples"))
		}),
	)
}

// exampleValueXorExternalAt lists the entries of one examples map.
func exampleValueXorExternalAt(root, examples *yaml.Node, path string) []oas.Node {
	return seq.FlatMapPairs(oas.Pairs(examples), func(key, value *yaml.Node) []oas.Node {
		return []oas.Node{{Key: key.Value, Node: oas.Resolve(root, value), Path: oas.Join(path, key.Value)}}
	})
}

// exampleValueXorExternalOwners lists the objects that can carry an examples
// map: parameters, headers and media types.
func exampleValueXorExternalOwners(root *yaml.Node) []oas.Node {
	return slices.Concat(
		seq.FlatMapPairs(oas.Pairs(oas.Dig(root, "components", "parameters")), func(key, value *yaml.Node) []oas.Node {
			return exampleValueXorExternalOwner(root, value, oas.JoinAll("$.components.parameters", key.Value))
		}),
		seq.FlatMapPairs(oas.Pairs(oas.Dig(root, "components", "headers")), func(key, value *yaml.Node) []oas.Node {
			return exampleValueXorExternalOwner(root, value, oas.JoinAll("$.components.headers", key.Value))
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "requestBodies")),
			func(key, value *yaml.Node) []oas.Node {
				path := oas.Join(oas.JoinAll("$.components.requestBodies", key.Value), "content")

				return exampleValueXorExternalContent(root, oas.Get(oas.Resolve(root, value), "content"), path)
			},
		),
		seq.FlatMapPairs(oas.Pairs(oas.Dig(root, "components", "responses")), func(key, value *yaml.Node) []oas.Node {
			return exampleValueXorExternalResponse(root, value, oas.JoinAll("$.components.responses", key.Value))
		}),
		seq.FlatMap(oas.PathItems(root), func(item oas.PathItem) []oas.Node {
			return exampleValueXorExternalParameters(
				root,
				oas.Get(item.Node, "parameters"),
				oas.Join(item.Path, "parameters"),
			)
		}),
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []oas.Node {
			body := oas.Resolve(root, oas.Get(operation.Node, "requestBody"))

			return slices.Concat(
				exampleValueXorExternalParameters(
					root,
					oas.Get(operation.Node, "parameters"),
					oas.Join(operation.Path, "parameters"),
				),
				exampleValueXorExternalContent(
					root,
					oas.Get(body, "content"),
					oas.JoinAll(operation.Path, "requestBody", "content"),
				),
				seq.FlatMapSlice(oas.Responses(root, operation), func(response oas.Response) []oas.Node {
					return exampleValueXorExternalResponse(root, response.Node, response.Path)
				}),
			)
		}),
	)
}

// exampleValueXorExternalOwner wraps one resolved owner node.
func exampleValueXorExternalOwner(root, node *yaml.Node, path string) []oas.Node {
	return []oas.Node{{Node: oas.Resolve(root, node), Path: path}}
}

// exampleValueXorExternalParameters lists the parameters of a parameter list.
func exampleValueXorExternalParameters(root, list *yaml.Node, path string) []oas.Node {
	return seq.FlatMapPairs(oas.Items(list), func(i int, parameter *yaml.Node) []oas.Node {
		return exampleValueXorExternalOwner(root, parameter, oas.Index(path, i))
	})
}

// exampleValueXorExternalContent lists the media types of a content map.
func exampleValueXorExternalContent(root, content *yaml.Node, path string) []oas.Node {
	return seq.FlatMapPairs(oas.Pairs(content), func(key, value *yaml.Node) []oas.Node {
		return exampleValueXorExternalOwner(root, value, oas.Join(path, key.Value))
	})
}

// exampleValueXorExternalResponse lists the media types and headers of a response.
func exampleValueXorExternalResponse(root, response *yaml.Node, path string) []oas.Node {
	resolved := oas.Resolve(root, response)

	return slices.Concat(
		exampleValueXorExternalContent(root, oas.Get(resolved, "content"), oas.Join(path, "content")),
		seq.FlatMapPairs(oas.Pairs(oas.Get(resolved, "headers")), func(key, value *yaml.Node) []oas.Node {
			return exampleValueXorExternalOwner(root, value, oas.JoinAll(path, "headers", key.Value))
		}),
	)
}
