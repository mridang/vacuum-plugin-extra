package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// responseRequestContentRequiredIgnoreOption is the name of the option listing
// status codes that need no content.
const responseRequestContentRequiredIgnoreOption = "ignoreCodes"

// ResponseRequestContentRequired returns the extra-response-request-content-required rule.
//
// # What it checks
//
// Every request body and every response of an operation must have a content
// map. A response without content cannot be validated or deserialized by
// clients, and a request body without content gives the server nothing to
// parse. The check follows local $ref references, so a response shared in
// components is checked in the context of the status code that uses it.
// Responses that by definition carry no body are exempt: 204 and 304 (the
// ignoreCodes option), any 1xx informational code, and every response of a
// HEAD operation.
//
// # Where it comes from
//
// IBM's openapi-validator (ibm-request-and-response-content), which requires
// content on request bodies and on responses other than 204.
//
// # Options
//
//   - ignoreCodes: status codes exempt from the check. Default: 204, 304.
//
// # Bad
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        description: a pet
//	      responses:
//	        "201":
//	          description: created
//
// # Good
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        content:
//	          application/json:
//	            schema:
//	              type: object
//	      responses:
//	        "201":
//	          description: created
//	          content:
//	            application/json:
//	              schema:
//	                type: object
//
// # Known false positives
//
// Error responses such as 404 that intentionally return an empty body are
// flagged; add their codes to ignoreCodes or document an error body.
func ResponseRequestContentRequired() Rule {
	return New(Spec{
		Name:        "response-request-content-required",
		Description: "Request bodies and responses other than 204 must have a content field.",
		HowToFix:    "Add a content map describing the payload, or use a status code that has no body.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{{
			Name:        responseRequestContentRequiredIgnoreOption,
			Description: "Status codes whose responses need no content.",
			Default:     []string{"204", "304"},
		}},
		Check: PerOperation(checkResponseRequestContentRequired),
	})
}

// checkResponseRequestContentRequired reports request bodies and responses
// without content. HEAD operations never have response content.
func checkResponseRequestContentRequired(call *Call, operation oas.Operation) []Finding {
	body := oas.Get(operation.Node, "requestBody")
	resolved := oas.Resolve(call.Root, body)
	requestFindings := When(oas.IsMap(resolved) && !oas.Has(resolved, "content"), func() Finding {
		return Finding{
			Node:    body,
			Path:    oas.Join(operation.Path, "requestBody"),
			Message: "request body must have content",
		}
	})
	if operation.Method == methodHead {
		return requestFindings
	}

	ignored := call.Strings(responseRequestContentRequiredIgnoreOption)
	responseFindings := seq.FlatMapSlice(oas.Responses(call.Root, operation), func(response oas.Response) []Finding {
		return When(
			!oas.Has(response.Node, "content") && !responseRequestContentRequiredExempt(response.Code, ignored),
			func() Finding {
				return Finding{
					Node:    response.Node,
					Path:    response.Path,
					Message: "response " + response.Code + " must have content",
				}
			},
		)
	})

	return slices.Concat(requestFindings, responseFindings)
}

// responseRequestContentRequiredExempt reports whether the status code needs no
// content: informational codes and the configured ones.
func responseRequestContentRequiredExempt(code string, ignored []string) bool {
	return strings.HasPrefix(code, "1") || slices.Contains(ignored, code)
}
