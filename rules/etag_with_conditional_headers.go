package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// EtagWithConditionalHeaders returns the extra-etag-with-conditional-headers rule.
//
// # What it checks
//
// When any operation of a path item accepts an If-Match or If-None-Match
// header parameter (declared on the operation or on the path item, matched
// case-insensitively), every 2xx response of the GET operation of that same
// path item must declare an ETag response header. Clients can only send a
// conditional request when the resource hands out a validator to compare.
// A path item with conditional headers but no GET operation is not reported.
//
// # Where it comes from
//
// IBM's API Handbook and openapi-validator guidance on
// conditional requests.
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets/{id}:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	    put:
//	      parameters:
//	        - name: If-Match
//	          in: header
//	          schema:
//	            type: string
//
// # Good
//
//	paths:
//	  /pets/{id}:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          headers:
//	            ETag:
//	              schema:
//	                type: string
//
// # Known false positives
//
// A 2xx GET response that cannot carry a validator (for example 204 returned
// for an empty collection) is flagged. Responses reached through $ref are
// reported at the operation, not at the component.
func EtagWithConditionalHeaders() Rule {
	return New(Spec{
		Name:        "etag-with-conditional-headers",
		Description: "A path item with If-Match or If-None-Match parameters must return an ETag header from GET.",
		HowToFix:    "Declare an ETag response header on the 2xx responses of the GET operation.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkEtagWithConditionalHeaders),
	})
}

func checkEtagWithConditionalHeaders(call *Call) []Finding {
	byRoute := seq.GroupBy(oas.Operations(call.Root), func(operation oas.Operation) string {
		return operation.Route
	})

	return seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
		if operation.Method != methodGet || !etagWithConditionalHeadersUsed(call, byRoute[operation.Route]) {
			return nil
		}

		return seq.FlatMapSlice(oas.Responses(call.Root, operation), func(response oas.Response) []Finding {
			return When(oas.IsSuccess(response.Code) && !etagWithConditionalHeadersHasETag(response), func() Finding {
				return Finding{
					Node:    response.Node,
					Path:    response.Path,
					Message: "GET response " + response.Code + " must declare an ETag header because conditional headers are used",
				}
			})
		})
	})
}

func etagWithConditionalHeadersUsed(call *Call, operations []oas.Operation) bool {
	return slices.ContainsFunc(operations, func(operation oas.Operation) bool {
		return slices.ContainsFunc(oas.Parameters(call.Root, operation), func(parameter oas.Parameter) bool {
			name := Fold(parameter.Name())

			return parameter.In() == keywordHeader && (name == "if-match" || name == "if-none-match")
		})
	})
}

func etagWithConditionalHeadersHasETag(response oas.Response) bool {
	for name := range oas.Pairs(oas.Get(response.Node, "headers")) {
		if strings.EqualFold(name.Value, "etag") {
			return true
		}
	}

	return false
}
