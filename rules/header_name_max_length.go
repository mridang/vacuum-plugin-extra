package rules

import (
	"slices"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	headerNameMaxLengthOption  = "maxLength"
	headerNameMaxLengthDefault = 50
)

// HeaderNameMaxLength returns the extra-header-name-max-length rule.
//
// # What it checks
//
// The names of response headers, in operation responses and in
// components.responses, must be at most maxLength characters long. Response
// definitions that are $refs are checked where they are declared.
//
// # Why
//
// Very long header names inflate every response, run into proxy and server
// header limits, and are awkward to read in logs and generated code.
//
// # Where it comes from
//
// rate-my-openapi (response header names at most 50 characters).
//
// # Options
//
//   - maxLength (integer, default 50): the longest allowed header name,
//     counted in characters.
//
// # Bad
//
//	responses:
//	  "200":
//	    description: ok
//	    headers:
//	      X-Some-Extremely-Long-Header-Name-That-Keeps-Going-And-Going:
//	        schema: {type: string}
//
// # Good
//
//	responses:
//	  "200":
//	    description: ok
//	    headers:
//	      X-Request-Id:
//	        schema: {type: string}
//
// # Known false positives and limitations
//
// Headers declared as parameters with in: header, and reusable headers under
// components.headers, are not response headers on their own and are not
// checked. A maxLength of zero or less disables the rule.
func HeaderNameMaxLength() Rule {
	return New(Spec{
		Name:        "header-name-max-length",
		Description: "Response header names must not exceed the maximum length.",
		HowToFix:    "Shorten the response header name.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        headerNameMaxLengthOption,
				Description: "The longest allowed header name, in characters.",
				Default:     headerNameMaxLengthDefault,
			},
		},
		Check: OnDocument(checkHeaderNameMaxLength),
	})
}

// checkHeaderNameMaxLength reports response header names that are too long.
func checkHeaderNameMaxLength(call *Call) []Finding {
	limit := call.Int(headerNameMaxLengthOption)
	if limit <= 0 {
		return nil
	}

	return slices.Concat(
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
			return seq.FlatMapPairs(
				oas.Pairs(oas.Get(operation.Node, "responses")),
				func(key, response *yaml.Node) []Finding {
					path := oas.JoinAll(operation.Path, "responses", key.Value)

					return headerNameMaxLengthResponse(response, path, limit)
				},
			)
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(call.Root, "components", "responses")),
			func(key, response *yaml.Node) []Finding {
				return headerNameMaxLengthResponse(response, oas.JoinAll("$.components.responses", key.Value), limit)
			},
		),
	)
}

// headerNameMaxLengthResponse checks the headers of one response object.
func headerNameMaxLengthResponse(response *yaml.Node, path string, limit int) []Finding {
	return seq.FlatMapPairs(oas.Pairs(oas.Get(response, "headers")), func(name, _ *yaml.Node) []Finding {
		return When(utf8.RuneCountInString(name.Value) > limit, func() Finding {
			return Finding{
				Node:    name,
				Path:    oas.JoinAll(path, "headers", name.Value),
				Message: "response header name is longer than the allowed maximum",
			}
		})
	})
}
