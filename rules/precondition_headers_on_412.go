package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// preconditionHeadersOn412Code is the status code of a failed precondition.
const preconditionHeadersOn412Code = "412"

// PreconditionHeadersOn412 returns the extra-precondition-headers-on-412 rule.
//
// # What it checks
//
// An operation that can return a 412 (Precondition Failed) response must also
// define at least one conditional request header as a header parameter. The
// parameters of the path item are taken into account, and parameters are
// resolved through local $ref. Header names are compared case-insensitively.
//
// A 412 can only be produced when the client sent a precondition. If the
// operation does not accept any, the response is unreachable and the contract
// is misleading.
//
// # Where it comes from
//
// IBM openapi-validator, rule ibm-precondition-header, which requires
// operations that return 412 to define If-Match, If-None-Match,
// If-Modified-Since or If-Unmodified-Since.
//
// # Options
//
//   - headers (list of strings, default [If-Match, If-None-Match,
//     If-Modified-Since, If-Unmodified-Since]): the header names, at least one
//     of which must be defined.
//
// # Bad
//
//	paths:
//	  /pets/{id}:
//	    put:
//	      responses:
//	        "412":
//	          description: precondition failed
//
// # Good
//
//	paths:
//	  /pets/{id}:
//	    put:
//	      parameters:
//	        - name: If-Match
//	          in: header
//	          schema:
//	            type: string
//	      responses:
//	        "412":
//	          description: precondition failed
//
// # Known false positives
//
// Preconditions that are announced in another way (for example a custom header
// that is not in the list) are reported until the headers option is extended.
func PreconditionHeadersOn412() Rule {
	return New(Spec{
		Name:        "precondition-headers-on-412",
		Description: "Operations that return a 412 response must define a conditional request header.",
		HowToFix:    "Add an If-Match, If-None-Match, If-Modified-Since or If-Unmodified-Since header parameter.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        keywordHeaders,
				Description: "The conditional request headers, one of which must be defined.",
				Default:     []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"},
			},
		},
		Check: PerOperation(checkPreconditionHeadersOn412),
	})
}

func checkPreconditionHeadersOn412(call *Call, operation oas.Operation) []Finding {
	wanted := seq.Map(slices.Values(call.Strings("headers")), Fold)
	responses := oas.Responses(call.Root, operation)
	index := slices.IndexFunc(responses, func(response oas.Response) bool {
		return response.Code == preconditionHeadersOn412Code
	})

	return When(
		len(wanted) > 0 && index >= 0 && !preconditionHeadersOn412HasHeader(call, operation, wanted),
		func() Finding {
			return Finding{
				Node: responses[index].Node,
				Path: responses[index].Path,
				Message: "operations that return 412 must define one of the headers: " +
					strings.Join(call.Strings("headers"), ", "),
			}
		},
	)
}

func preconditionHeadersOn412HasHeader(call *Call, operation oas.Operation, wanted []string) bool {
	return slices.ContainsFunc(oas.Parameters(call.Root, operation), func(parameter oas.Parameter) bool {
		return strings.EqualFold(parameter.In(), "header") && slices.Contains(wanted, Fold(parameter.Name()))
	})
}
