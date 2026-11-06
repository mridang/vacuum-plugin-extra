package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// noReservedHeaderParameterAccept is the Accept header name.
const noReservedHeaderParameterAccept = "Accept"

// NoReservedHeaderParameter returns the extra-no-reserved-header-parameter rule.
//
// # What it checks
//
// Header parameters must not be named after a header that OpenAPI describes in
// other ways. By default the reserved names are Accept, Authorization,
// Content-Type, If-Modified-Since and If-Unmodified-Since. Names are compared
// case-insensitively. Parameters are read from operations, path items and
// components.parameters.
//
// # Why
//
// The specification states that a parameter named Accept, Content-Type or
// Authorization in the header location is ignored: the content negotiation is
// described by the content maps and the credentials by security schemes. Such a
// parameter therefore documents something tools silently drop. The conditional
// request headers are better modeled as dedicated preconditions.
//
// # Where it comes from
//
// IBM openapi-validator, which has one rule for each of the Accept,
// Authorization, Content-Type, If-Modified-Since and If-Unmodified-Since header
// parameters. They are merged here into one rule with a list option.
//
// # Options
//
//   - headers: the reserved header names. Default: Accept, Authorization,
//     Content-Type, If-Modified-Since, If-Unmodified-Since.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: Accept
//	          in: header
//	          schema:
//	            type: string
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
//	            application/json: {}
//	            text/csv: {}
//
// # Known false positives / limitations
//
// APIs that deliberately expose conditional request headers as parameters will
// see If-Modified-Since and If-Unmodified-Since reported; remove them from the
// headers option.
func NoReservedHeaderParameter() Rule {
	return New(Spec{
		Name:        "no-reserved-header-parameter",
		Description: "Header parameters must not be named Accept, Authorization, Content-Type or similar reserved headers.",
		HowToFix:    "Remove the header parameter and describe the behavior with content, security schemes or preconditions.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        keywordHeaders,
			Description: "The reserved header names, compared case-insensitively.",
			Default: []string{
				noReservedHeaderParameterAccept,
				"Authorization",
				"Content-Type",
				"If-Modified-Since",
				"If-Unmodified-Since",
			},
		}},
		Check: OnDocument(checkNoReservedHeaderParameter),
	})
}

func checkNoReservedHeaderParameter(call *Call) []Finding {
	reserved := seq.Map(slices.Values(call.Strings("headers")), func(header string) string {
		return strings.ToLower(strings.TrimSpace(header))
	})
	parameters := documentParameters(call.Root)

	return seq.FlatMapIndexed(parameters, func(i int, parameter oas.Parameter) []Finding {
		first := slices.IndexFunc(parameters, func(other oas.Parameter) bool {
			return other.Node == parameter.Node
		})

		listed := slices.Contains(reserved, strings.ToLower(parameter.Name()))

		return When(
			first == i && parameter.In() == keywordHeader && listed,
			func() Finding {
				return Finding{
					Node:    parameter.Node,
					Path:    parameter.Path,
					Message: "header parameter '" + parameter.Name() + "' must not be defined, it is described by other parts of the document",
				}
			},
		)
	})
}
