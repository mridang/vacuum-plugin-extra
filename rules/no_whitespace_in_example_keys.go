package rules

import (
	"strings"
	"unicode"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// NoWhitespaceInExampleKeys returns the extra-no-whitespace-in-example-keys
// rule.
//
// # What it checks
//
// The keys of the examples map of a media type in a response must not contain
// whitespace. Operation responses and components/responses are inspected, and
// the content of every media type is covered.
//
// The key of a named example is an identifier. Whitespace in it makes the key
// awkward to reference, to use as a generated identifier and to type in tools
// that list the examples; a summary field exists for the human readable label.
//
// # Where it comes from
//
// IBM openapi-validator, rule ibm-no-whitespace-in-example-keys, a pattern
// check that rejects keys with whitespace.
//
// # Options
//
// None.
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
//	                a pet:
//	                  value: {}
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
//	                a-pet:
//	                  summary: A pet
//	                  value: {}
//
// # Known false positives
//
// Only response examples are inspected; examples of request bodies and
// parameters are not.
func NoWhitespaceInExampleKeys() Rule {
	return New(Spec{
		Name:        "no-whitespace-in-example-keys",
		Description: "Keys of response examples maps must not contain whitespace.",
		HowToFix:    "Remove the whitespace from the example key and use summary for the readable label.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkNoWhitespaceInExampleKeys),
	})
}

func checkNoWhitespaceInExampleKeys(call *Call) []Finding {
	return Concat(
		PerOperation(func(_ *Call, operation oas.Operation) []Finding {
			return seq.FlatMapPairs(
				oas.Pairs(oas.Get(operation.Node, "responses")),
				func(key, response *yaml.Node) []Finding {
					return noWhitespaceInExampleKeysInResponse(
						response,
						oas.JoinAll(operation.Path, "responses", key.Value),
					)
				},
			)
		}),
		OnDocument(func(call *Call) []Finding {
			return seq.FlatMapPairs(
				oas.Pairs(oas.Dig(call.Root, "components", "responses")),
				func(key, response *yaml.Node) []Finding {
					return noWhitespaceInExampleKeysInResponse(
						response,
						oas.JoinAll("$.components.responses", key.Value),
					)
				},
			)
		}),
	)(call)
}

func noWhitespaceInExampleKeysInResponse(response *yaml.Node, path string) []Finding {
	return seq.FlatMapSlice(
		oas.MediaTypes(oas.Get(response, "content"), oas.Join(path, "content")),
		func(media oas.MediaType) []Finding {
			return seq.FlatMapPairs(oas.Pairs(oas.Get(media.Node, "examples")), func(key, _ *yaml.Node) []Finding {
				return When(strings.ContainsFunc(key.Value, unicode.IsSpace), func() Finding {
					return Finding{
						Node:    key,
						Path:    oas.JoinAll(media.Path, "examples", key.Value),
						Message: "example key " + key.Value + " must not contain whitespace",
					}
				})
			})
		},
	)
}
