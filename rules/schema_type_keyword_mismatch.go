package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

const schemaTypeKeywordMismatchRequireType = "requireExplicitType"

// SchemaTypeKeywordMismatch returns the extra-schema-type-keyword-mismatch rule.
//
// # What it checks
//
// A schema must not carry keywords that belong to a type it does not have:
//
//   - a schema with a declared type other than array must not carry items;
//   - a schema with a declared type other than object must not carry properties;
//   - with requireExplicitType (the default), a schema that carries properties
//     but declares no type must declare type: object, and one that carries
//     items but declares no type must declare type: array.
//
// OpenAPI 3.1 type arrays are honoured: ["object", "null"] allows properties.
//
// # Why
//
// JSON Schema ignores keywords that do not apply to the instance type, so an
// items keyword on a string schema silently does nothing. These are almost
// always copy and paste mistakes, and tools disagree on how to treat them.
// Leaving the type out of a schema with properties makes generators fall back to
// an untyped value instead of an object.
//
// # Where it comes from
//
// Merges Redocly CLI (schema-type-keyword-mismatch) with oas-kit
// (schema-with-properties-must-be-type-object).
//
// # Options
//
//   - requireExplicitType (bool, default true): also require type: object next
//     to properties and type: array next to items. With false only an
//     explicit, contradicting type is reported.
//
// # Bad
//
//	components:
//	  schemas:
//	    Tag:
//	      type: string
//	      items:
//	        type: string
//	    Pet:
//	      properties:
//	        name: {type: string}
//
// # Good
//
//	components:
//	  schemas:
//	    Tag:
//	      type: array
//	      items:
//	        type: string
//	    Pet:
//	      type: object
//	      properties:
//	        name: {type: string}
//
// # Known false positives and limitations
//
// Schemas that are valid JSON Schema without a type (for instance allOf
// wrappers that add properties) are reported while requireExplicitType is on.
// The same applies to a type given only in an allOf member.
func SchemaTypeKeywordMismatch() Rule {
	return New(Spec{
		Name:        "schema-type-keyword-mismatch",
		Description: "A schema must not carry items or properties that contradict its type, and properties require type object.",
		HowToFix:    "Declare type array next to items and type object next to properties, or remove the keyword.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        schemaTypeKeywordMismatchRequireType,
				Description: "Require type object next to properties and type array next to items.",
				Default:     true,
			},
		},
		Check: PerSchema(checkSchemaTypeKeywordMismatch),
	})
}

// checkSchemaTypeKeywordMismatch compares the type of every schema with its keywords.
func checkSchemaTypeKeywordMismatch(call *Call, schema oas.Schema) []Finding {
	explicit := call.Bool(schemaTypeKeywordMismatchRequireType)

	return slices.Concat(
		schemaTypeKeywordMismatchKeyword(schema, "items", "array", explicit),
		schemaTypeKeywordMismatchKeyword(schema, "properties", "object", explicit),
	)
}

// schemaTypeKeywordMismatchKeyword checks one keyword against the type it belongs to.
func schemaTypeKeywordMismatchKeyword(schema oas.Schema, keyword string, kind string, explicit bool) []Finding {
	key, value := oas.Entry(schema.Node, keyword)
	message := schemaTypeKeywordMismatchMessage(schema.Node, keyword, kind, explicit)

	return When(value != nil && message != "", func() Finding {
		return Finding{Node: key, Path: oas.Join(schema.Path, keyword), Message: message}
	})
}

// schemaTypeKeywordMismatchMessage describes the mismatch, or returns an empty string.
func schemaTypeKeywordMismatchMessage(schema *yaml.Node, keyword string, kind string, explicit bool) string {
	types := oas.Types(schema)

	switch {
	case len(types) == 0 && explicit:
		return "schema with " + keyword + " must declare type: " + kind
	case len(types) > 0 && !oas.HasType(schema, kind):
		return "schema of type " + types[0] + " must not carry " + keyword
	default:
		return ""
	}
}
