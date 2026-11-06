package rules

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	exampleBodyKeyCaseOption  = "style"
	exampleBodyKeyCaseDefault = "snake-case"
)

// ExampleBodyKeyCase returns the extra-example-body-key-case rule.
//
// # What it checks
//
// Property names inside example bodies must follow the configured case style.
// The examples checked are the example and examples[*].value of every JSON media
// type in request bodies and responses, inline or under components. Nested
// objects and arrays are walked, and every mapping key is compared with the
// re-cased form of itself.
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
//   - style (string, default "snake-case"): one of snake-case, dash-case,
//     camel-case (UpperCamelCase) or lower-camel-case. Any other value makes
//     the rule do nothing.
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
			{
				Name:        exampleBodyKeyCaseOption,
				Description: "snake-case, dash-case, camel-case or lower-camel-case.",
				Default:     exampleBodyKeyCaseDefault,
			},
		},
		Check: OnDocument(checkExampleBodyKeyCase),
	})
}

type exampleBodyKeyCaseExample struct {
	node *yaml.Node
	path string
}

// checkExampleBodyKeyCase walks every example body and compares key casing.
func checkExampleBodyKeyCase(call *Call) []Finding {
	style := call.String(exampleBodyKeyCaseOption)
	if !exampleBodyKeyCaseKnownStyle(style) {
		return nil
	}

	return seq.FlatMapSlice(exampleBodyKeyCaseExamples(call.Root), func(example exampleBodyKeyCaseExample) []Finding {
		return exampleBodyKeyCaseWalk(example.node, example.path, style)
	})
}

// exampleBodyKeyCaseKnownStyle reports whether the style is supported.
func exampleBodyKeyCaseKnownStyle(style string) bool {
	switch style {
	case "snake-case", "dash-case", "camel-case", "lower-camel-case":
		return true
	default:
		return false
	}
}

// exampleBodyKeyCaseWalk reports the keys of one example value that are not in the style.
func exampleBodyKeyCaseWalk(value *yaml.Node, path string, style string) []Finding {
	return slices.Concat(
		exampleBodyKeyCaseKeys(value, path, style),
		seq.FlatMap(oas.Walk(value, path), func(node oas.Node) []Finding {
			return exampleBodyKeyCaseKeys(node.Node, node.Path, style)
		}),
	)
}

// exampleBodyKeyCaseKeys reports the direct keys of a mapping that are not in the style.
func exampleBodyKeyCaseKeys(mapping *yaml.Node, path string, style string) []Finding {
	return seq.FlatMapPairs(oas.Pairs(mapping), func(key, _ *yaml.Node) []Finding {
		expected := exampleBodyKeyCaseRecase(key.Value, style)

		return When(key.Value != expected, func() Finding {
			return Finding{
				Node:    key,
				Path:    oas.Join(path, key.Value),
				Message: "example body property '" + key.Value + "' should be cased like '" + expected + "'",
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

// exampleBodyKeyCaseRecase rewrites key in the given style.
func exampleBodyKeyCaseRecase(key string, style string) string {
	words := exampleBodyKeyCaseSplit(key)
	if len(words) == 0 {
		return key
	}

	switch style {
	case "snake-case":
		return strings.ToLower(strings.Join(words, "_"))
	case "dash-case":
		return strings.ToLower(strings.Join(words, "-"))
	default:
		return exampleBodyKeyCaseCamel(words, style == "camel-case")
	}
}

// exampleBodyKeyCaseCamel joins words in camel case, capitalising the first word when upper is set.
func exampleBodyKeyCaseCamel(words []string, upper bool) string {
	return strings.Join(seq.FlatMapIndexed(words, func(i int, word string) []string {
		return []string{exampleBodyKeyCaseCamelWord(word, upper || i > 0)}
	}), "")
}

// exampleBodyKeyCaseCamelWord capitalises or lowers one word; all-capital words are kept as acronyms.
func exampleBodyKeyCaseCamelWord(word string, capitalise bool) string {
	if word == strings.ToUpper(word) {
		if capitalise {
			return word
		}

		return strings.ToLower(word)
	}

	lower := strings.ToLower(word)
	if !capitalise {
		return lower
	}

	first, size := utf8.DecodeRuneInString(lower)

	return string(unicode.ToUpper(first)) + lower[size:]
}

// exampleBodyKeyCaseSplit splits a key into words on separators and case changes.
func exampleBodyKeyCaseSplit(key string) []string {
	return seq.FlatMapSlice(
		strings.FieldsFunc(key, func(r rune) bool { return r == '-' || r == '_' }),
		func(segment string) []string {
			runes := []rune(segment)
			bounds := slices.Concat(
				[]int{0},
				seq.FlatMapIndexed(runes, func(i int, _ rune) []int {
					if !exampleBodyKeyCaseBoundary(runes, i) {
						return nil
					}

					return []int{i}
				}),
				[]int{len(runes)},
			)

			return seq.FlatMapIndexed(bounds[:len(bounds)-1], func(n int, start int) []string {
				return []string{string(runes[start:bounds[n+1]])}
			})
		},
	)
}

// exampleBodyKeyCaseBoundary reports whether a new word starts at index i.
func exampleBodyKeyCaseBoundary(runes []rune, i int) bool {
	if i == 0 || !unicode.IsUpper(runes[i]) {
		return false
	}

	previous := runes[i-1]
	if unicode.IsLower(previous) || unicode.IsDigit(previous) {
		return true
	}

	return unicode.IsUpper(previous) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
}
