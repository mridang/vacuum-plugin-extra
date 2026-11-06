package rules

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// AvoidInlineObjectSchemas returns the extra-avoid-inline-object-schemas rule.
//
// # What it checks
//
// Object schemas must be defined once under components/schemas and referenced
// with $ref instead of being written inline. A schema is reported when it is
// not itself a $ref, it does not live directly under components/schemas, and it
// is an object schema: it defines properties or it composes other schemas with
// allOf, anyOf or oneOf. The schema positions that are inspected are request
// and response bodies, parameters and headers, and every nested position
// (properties, items, additionalProperties and the composition keywords).
// Primitive schemas such as a string or an array of strings are never reported.
//
// Named models give code generators a type name to use, keep the description
// of a structure in one place and make the document smaller. Anonymous inline
// objects force generators to invent names and make it impossible to reuse the
// structure.
//
// # Where it comes from
//
// IBM openapi-validator, rule ibm-avoid-inline-schemas.
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
//	              properties:
//	                name:
//	                  type: string
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
//	              $ref: "#/components/schemas/Pet"
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      properties:
//	        name:
//	          type: string
//
// # Known false positives
//
// Every inline object is reported, including objects nested inside another
// inline object, so one inline structure can produce several findings.
// Schemas that are intentionally anonymous, such as a one-off error payload,
// have to be named or the rule suppressed for that location. An object schema
// without properties or composition (a free-form object) is not reported.
func AvoidInlineObjectSchemas() Rule {
	return New(Spec{
		Name:        "avoid-inline-object-schemas",
		Description: "Object schemas must be defined in components/schemas and referenced instead of being inlined.",
		HowToFix:    "Move the inline object schema to components/schemas and reference it with $ref.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerSchema(checkAvoidInlineObjectSchemas),
	})
}

func checkAvoidInlineObjectSchemas(_ *Call, schema oas.Schema) []Finding {
	return When(schema.Component == "" && avoidInlineObjectSchemasIsInlineObject(schema.Node), func() Finding {
		return Finding{
			Node:    schema.Node,
			Path:    schema.Path,
			Message: "object schemas must be defined in components/schemas and referenced with $ref",
		}
	})
}

func avoidInlineObjectSchemasIsInlineObject(schema *yaml.Node) bool {
	if oas.Ref(schema) != "" {
		return false
	}

	if !oas.Empty(oas.Get(schema, "properties")) {
		return true
	}

	for _, keyword := range []string{avoidInlineObjectSchemasAllOf, avoidInlineObjectSchemasAnyOf, literalOneof} {
		if !oas.Empty(oas.Get(schema, keyword)) {
			return true
		}
	}

	return false
}

// Keyword names used by this rule.
const (
	avoidInlineObjectSchemasAllOf = "allOf"
	avoidInlineObjectSchemasAnyOf = "anyOf"
)
