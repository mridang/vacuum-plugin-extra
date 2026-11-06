package rules

import (
	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

const requestBodyJSONMediaTypeJSON = "application/json"

const requestBodyJSONMediaTypeOption = "mediaType"

// RequestBodyJSONMediaType returns the extra-request-body-json-media-type rule.
//
// # What it checks
//
// An operation whose request body defines a non-empty content map must include
// the configured media type, application/json by default, as an exact key. A
// key such as application/json; charset=utf-8, application/vnd.api+json or
// application/*+json does not count. A request body given by $ref is judged by
// the object it points at, and operations without a request body, or with an
// empty content map, are not reported.
//
// # Why
//
// Language-model agents, code generators and most HTTP tooling default to JSON.
// A request body offered only as XML, form data or a vendor type cannot be
// called by a client that has only the plain JSON media type available.
//
// # Where it comes from
//
// ai-api-lint (request body should support application/json).
//
// # Options
//
//   - mediaType (string, default "application/json"): the media type that has
//     to be offered.
//
// # Bad
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        content:
//	          application/xml:
//	            schema: {type: string}
//
// # Good
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        content:
//	          application/json:
//	            schema: {type: string}
//	          application/xml:
//	            schema: {type: string}
//
// # Known false positives and limitations
//
// File upload operations that only accept multipart/form-data or
// application/octet-stream are reported, since they cannot offer JSON; suppress
// the rule for them. An empty mediaType option disables the rule.
func RequestBodyJSONMediaType() Rule {
	return New(Spec{
		Name:        "request-body-json-media-type",
		Description: "Request bodies must offer the application/json media type.",
		HowToFix:    "Add an application/json entry to the request body content.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        requestBodyJSONMediaTypeOption,
				Description: "The media type every request body has to offer.",
				Default:     requestBodyJSONMediaTypeJSON,
			},
		},
		Check: PerOperation(checkRequestBodyJSONMediaType),
	})
}

// checkRequestBodyJSONMediaType reports request bodies that lack the required media type.
func checkRequestBodyJSONMediaType(call *Call, operation oas.Operation) []Finding {
	required := call.String(requestBodyJSONMediaTypeOption)
	body := oas.Get(operation.Node, "requestBody")
	content := oas.Get(oas.Resolve(call.Root, body), "content")

	return When(
		required != "" && oas.IsMap(content) && !oas.Empty(content) && !oas.Has(content, required),
		func() Finding {
			return Finding{
				Node:    body,
				Path:    oas.Join(operation.Path, "requestBody"),
				Message: "request body does not include the '" + required + "' media type",
			}
		},
	)
}
