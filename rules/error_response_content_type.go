package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// ErrorResponseContentType returns the extra-error-response-content-type rule.
//
// # What it checks
//
// An error response (a response whose status code starts with 4 or 5, which
// includes ranges such as 4XX and 5XX) that declares a content object must
// offer an application/json entry. Media type parameters such as
// "; charset=utf-8" are ignored. Responses without any content, for example a
// bodiless 404, are not reported. Responses defined through a local $ref are
// resolved.
//
// A single, predictable format for error payloads lets clients parse every
// failure the same way, whatever the endpoint returns on success.
//
// # Where it comes from
//
// IBM openapi-validator, rule ibm-error-content-type-is-json.
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
//	        "404":
//	          description: not found
//	          content:
//	            text/plain:
//	              schema:
//	                type: string
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "404":
//	          description: not found
//	          content:
//	            application/json:
//	              schema:
//	                type: object
//
// # Known false positives
//
// Problem Details responses served as application/problem+json are reported
// because only the exact application/json media type is accepted. Add an
// application/json entry or suppress the rule for such responses.
func ErrorResponseContentType() Rule {
	return New(Spec{
		Name:        "error-response-content-type",
		Description: "Error responses that define content must offer an application/json entry.",
		HowToFix:    "Add an application/json entry to the content of the error response.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerResponse(checkErrorResponseContentType),
	})
}

// checkErrorResponseContentType reports error responses that define content
// without an application/json entry.
func checkErrorResponseContentType(_ *Call, _ oas.Operation, response oas.Response) []Finding {
	content := oas.Get(response.Node, "content")

	return When(
		oas.IsClientOrServerError(response.Code) && !oas.Empty(content) && !errorResponseContentTypeHasJSON(content),
		func() Finding {
			return Finding{
				Node:    content,
				Path:    oas.Join(response.Path, "content"),
				Message: "error response " + response.Code + " must define an application/json content entry",
			}
		},
	)
}

// errorResponseContentTypeHasJSON reports whether the content map has a JSON
// media type, ignoring parameters such as the charset.
func errorResponseContentTypeHasJSON(content *yaml.Node) bool {
	for name := range oas.Pairs(content) {
		base, _, _ := strings.Cut(name.Value, ";")
		if strings.EqualFold(strings.TrimSpace(base), "application/json") {
			return true
		}
	}

	return false
}
