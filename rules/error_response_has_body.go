package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// ErrorResponseHasBody returns the extra-error-response-has-body rule.
//
// # What it checks
//
// Every 4xx response of every operation, including range keys such as 4XX, must
// declare a response body: a content map with at least one media type that has
// a schema. $ref responses are resolved. The finding is reported on the
// response.
//
// # Why
//
// Clients need a machine-readable explanation of what went wrong. A 400 or 404
// with only a description forces them to parse free text, and generated clients
// cannot type the error.
//
// # Where it comes from
//
// apilint (error responses declare a body).
//
// # Options
//
//   - requireSchema (boolean, default true): when false, a media type without a
//     schema is accepted as long as content is not empty.
//
// # Bad
//
//	responses:
//	  "404":
//	    description: Not found
//
// # Good
//
//	responses:
//	  "404":
//	    description: Not found
//	    content:
//	      application/problem+json:
//	        schema:
//	          $ref: "#/components/schemas/Problem"
//
// # Known false positives
//
// 404 responses to HEAD requests, and 4xx responses that intentionally return
// no content, are flagged. A default response is not inspected.
func ErrorResponseHasBody() Rule {
	return New(Spec{
		Name:        "error-response-has-body",
		Description: "4xx responses must declare a response body with a schema.",
		HowToFix:    "Add a content map with an error schema to the response.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        "requireSchema",
			Description: "Require each media type to declare a schema.",
			Default:     true,
		}},
		Check: PerResponse(checkErrorResponseHasBody),
	})
}

// checkErrorResponseHasBody reports 4xx responses without a body.
func checkErrorResponseHasBody(call *Call, _ oas.Operation, response oas.Response) []Finding {
	requireSchema := call.Bool("requireSchema")

	declared := slices.ContainsFunc(
		oas.MediaTypes(oas.Get(response.Node, "content"), ""),
		func(media oas.MediaType) bool {
			return !requireSchema || oas.IsMap(oas.Get(media.Node, "schema"))
		},
	)

	return When(strings.HasPrefix(response.Code, "4") && !declared, func() Finding {
		return Finding{
			Node:    response.Node,
			Path:    response.Path,
			Message: "error response " + response.Code + " must declare a response body with a schema",
		}
	})
}
