package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// requestBodyRequiredListJSON is the default media type.
const requestBodyRequiredListJSON = "application/json"

// RequestBodyRequiredList returns the extra-request-body-required-list rule.
//
// # What it checks
//
// For every operation, the schema of the JSON request body must list its
// mandatory properties. A schema that declares properties but has no required
// list, or an empty one, is reported. The schema is read from the media types
// named by the mediaTypes option, and a $ref schema is followed to its target.
//
// # Why
//
// Without a required list every property is optional. Clients, code generators
// and AI agents then cannot tell which fields they must send, and servers
// receive incomplete payloads that only fail at runtime.
//
// # Where it comes from
//
// ai-api-lint (AI032, request body schema should specify required fields).
//
// # Options
//
//   - mediaTypes: the request body media types to inspect. Default:
//     application/json.
//
// # Bad
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        content:
//	          application/json:
//	            schema:
//	              type: object
//	              properties:
//	                name: {type: string}
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
//	              required: [name]
//	              properties:
//	                name: {type: string}
//
// # Known false positives / limitations
//
// Request bodies in which every property is genuinely optional, such as PATCH
// payloads, are reported. Only the top-level schema is examined; properties
// introduced through allOf are not counted.
func RequestBodyRequiredList() Rule {
	return New(Spec{
		Name:        "request-body-required-list",
		Description: "A JSON request body schema with properties must list its required properties.",
		HowToFix:    "Add a required list with the mandatory properties to the request body schema.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        "mediaTypes",
			Description: "The request body media types to inspect.",
			Default:     []string{requestBodyRequiredListJSON},
		}},
		Check: PerOperation(checkRequestBodyRequiredList),
	})
}

func checkRequestBodyRequiredList(call *Call, operation oas.Operation) []Finding {
	wanted := seq.Map(slices.Values(call.Strings("mediaTypes")), func(mediaType string) string {
		return strings.ToLower(strings.TrimSpace(mediaType))
	})
	body := oas.Resolve(call.Root, oas.Get(operation.Node, "requestBody"))
	contentPath := oas.JoinAll(operation.Path, "requestBody", "content")

	return seq.FlatMapPairs(oas.Pairs(oas.Get(body, "content")), func(name, media *yaml.Node) []Finding {
		schema := oas.Resolve(call.Root, oas.Get(media, "schema"))

		return When(
			slices.Contains(wanted, strings.ToLower(name.Value)) &&
				!oas.Empty(oas.Get(schema, "properties")) && oas.Empty(oas.Get(schema, "required")),
			func() Finding {
				return Finding{
					Node:    oas.Get(media, "schema"),
					Path:    oas.JoinAll(contentPath, name.Value, "schema"),
					Message: "request body schema has properties but no required list",
				}
			},
		)
	})
}
