package rules

import (
	"net/http"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// successResponseHasSchemaIgnored is the option with the codes that carry no body.
const successResponseHasSchemaIgnored = "ignoredCodes"

// SuccessResponseHasSchema returns the extra-success-response-has-schema rule.
//
// # What it checks
//
// For every operation, the first 2xx response must declare a body schema: it
// must have content, and at least one of its media types must define a schema.
// Responses whose status code is listed in ignoredCodes (204 and 205 by
// default, which never carry a body) are skipped, as are HEAD operations.
//
// # Why
//
// A success response without a schema leaves clients and test generators
// guessing at the shape of the data that comes back, and prevents response
// validation.
//
// # Where it comes from
//
// ai-api-lint (success-response-has-schema).
//
// # Options
//
//   - ignoredCodes (list of strings, default [204, 205]): success status codes
//     that are allowed to have no schema.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
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
//	              schema:
//	                type: array
//	                items:
//	                  type: string
//
// # Known false positives / limitations
//
// Only the first success response in document order is examined. Responses
// that redirect with a 3xx code or that stream binary data without a schema
// are reported only if they are 2xx and not ignored; add the code to
// ignoredCodes when it is intentionally schema-less.
func SuccessResponseHasSchema() Rule {
	return New(Spec{
		Name:        "success-response-has-schema",
		Description: "The first success response of an operation must define a body schema.",
		HowToFix:    "Add content with a schema to the success response.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        successResponseHasSchemaIgnored,
				Description: "Success status codes that may have no schema.",
				Default:     []string{"204", "205"},
			},
		},
		Check: OnDocument(checkSuccessResponseHasSchema),
	})
}

// checkSuccessResponseHasSchema reports schema-less success responses.
func checkSuccessResponseHasSchema(call *Call) []Finding {
	ignored := call.Strings(successResponseHasSchemaIgnored)

	return PerOperation(func(call *Call, operation oas.Operation) []Finding {
		if strings.EqualFold(operation.Method, http.MethodHead) {
			return nil
		}

		responses := oas.Responses(call.Root, operation)
		index := slices.IndexFunc(responses, func(response oas.Response) bool {
			return oas.IsSuccess(response.Code)
		})
		if index < 0 {
			return nil
		}

		response := responses[index]

		return When(
			!slices.Contains(ignored, response.Code) && !successResponseHasSchemaDefined(response),
			func() Finding {
				return Finding{
					Node:    response.Node,
					Path:    response.Path,
					Message: "success response " + response.Code + " must define a body schema",
				}
			},
		)
	})(call)
}

// successResponseHasSchemaDefined reports a media type with a schema.
func successResponseHasSchemaDefined(response oas.Response) bool {
	return slices.ContainsFunc(
		oas.MediaTypes(oas.Get(response.Node, "content"), oas.Join(response.Path, "content")),
		func(media oas.MediaType) bool { return oas.Has(media.Node, "schema") },
	)
}
