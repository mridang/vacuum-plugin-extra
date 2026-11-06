package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	// requestBodySchemaHasPropertiesAllOf is the allOf keyword.
	requestBodySchemaHasPropertiesAllOf = "allOf"
	// requestBodySchemaHasPropertiesAnyOf is the anyOf keyword.
	requestBodySchemaHasPropertiesAnyOf = "anyOf"
	// requestBodySchemaHasPropertiesOneOf is the oneOf keyword.
	requestBodySchemaHasPropertiesOneOf = "oneOf"
)

// RequestBodySchemaHasProperties returns the extra-request-body-schema-has-properties rule.
//
// # What it checks
//
// The schema of a JSON request body (media type application/json or any
// +json type) must describe its properties. The schema is $ref-resolved first.
// An object schema (or one without a type) that has no properties and none of
// allOf, anyOf, oneOf, additionalProperties or patternProperties is reported,
// because it describes nothing about the payload. Schemas of another type, such
// as arrays or strings, are not reported.
//
// # Why
//
// A body that is declared as a bare object tells consumers and code generators
// nothing: clients cannot construct it, validators accept anything, and
// language models that read the specification have no field names to use.
//
// # Where it comes from
//
// ai-api-lint (request-body-schema-has-properties).
//
// # Options
//
// None.
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
//	              properties:
//	                name:
//	                  type: string
//
// # Known false positives / limitations
//
// A deliberately free-form body is not reported when it declares
// additionalProperties; one that declares nothing at all is. Request bodies
// declared in components.requestBodies are checked where they are defined.
func RequestBodySchemaHasProperties() Rule {
	return New(Spec{
		Name:        "request-body-schema-has-properties",
		Description: "JSON request body schemas must define properties.",
		HowToFix:    "Describe the fields of the request body with properties.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkRequestBodySchemaHasProperties),
	})
}

// checkRequestBodySchemaHasProperties reports request bodies without properties.
func checkRequestBodySchemaHasProperties(call *Call) []Finding {
	return slices.Concat(
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
			body := oas.Get(operation.Node, "requestBody")
			if oas.Ref(body) != "" {
				return nil
			}

			return requestBodySchemaHasPropertiesBody(call.Root, body, oas.Join(operation.Path, "requestBody"))
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(call.Root, "components", "requestBodies")),
			func(key, body *yaml.Node) []Finding {
				return requestBodySchemaHasPropertiesBody(
					call.Root,
					body,
					oas.JoinAll("$.components.requestBodies", key.Value),
				)
			},
		),
	)
}

// requestBodySchemaHasPropertiesBody checks the JSON media types of a body.
func requestBodySchemaHasPropertiesBody(root, body *yaml.Node, path string) []Finding {
	return seq.FlatMapSlice(
		oas.MediaTypes(oas.Get(body, "content"), oas.Join(path, "content")),
		func(media oas.MediaType) []Finding {
			name := strings.ToLower(strings.TrimSpace(media.Name))
			if !strings.HasPrefix(name, "application/json") && !strings.Contains(name, "+json") {
				return nil
			}

			schema := oas.Resolve(root, oas.Get(media.Node, "schema"))

			return When(requestBodySchemaHasPropertiesMissing(schema), func() Finding {
				return Finding{
					Node:    oas.Get(media.Node, "schema"),
					Path:    oas.Join(media.Path, "schema"),
					Message: "request body schema must define properties",
				}
			})
		},
	)
}

// requestBodySchemaHasPropertiesMissing reports an object schema with no description of its fields.
func requestBodySchemaHasPropertiesMissing(schema *yaml.Node) bool {
	if !oas.IsMap(schema) {
		return false
	}

	types := oas.Types(schema)
	if len(types) > 0 && !oas.HasType(schema, "object") {
		return false
	}

	describing := []string{
		literalProperties,
		requestBodySchemaHasPropertiesAllOf,
		requestBodySchemaHasPropertiesAnyOf,
		requestBodySchemaHasPropertiesOneOf,
		"additionalProperties",
		literalPatternproperties,
	}

	return !slices.ContainsFunc(describing, func(key string) bool {
		return oas.Has(schema, key)
	})
}
