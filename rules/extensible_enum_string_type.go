package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// ExtensibleEnumStringType returns the extra-extensible-enum-string-type rule.
//
// # What it checks
//
// Every schema or property that carries the x-extensible-enum extension must
// have type string. A schema without a type is reported too. In OpenAPI 3.1 a
// type list is accepted when it contains string and, optionally, null.
//
// An extensible enum lists the values known today while allowing clients to
// receive new ones, which only makes sense for strings.
//
// # Where it comes from
//
// Zally, the guideline that x-extensible-enum is only defined for string
// values.
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Colour:
//	      type: integer
//	      x-extensible-enum: [1, 2]
//
// # Good
//
//	components:
//	  schemas:
//	    Colour:
//	      type: string
//	      x-extensible-enum: [red, green]
//
// # Known false positives
//
// A schema that gets its type from an allOf member and only adds the extension
// itself is reported because the type is not declared on the same schema.
func ExtensibleEnumStringType() Rule {
	return New(Spec{
		Name:        "extensible-enum-string-type",
		Description: "Schemas with x-extensible-enum must have type string.",
		HowToFix:    "Declare type: string on the schema that carries x-extensible-enum.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkExtensibleEnumStringType),
	})
}

// checkExtensibleEnumStringType reports schemas that carry x-extensible-enum
// without being a string.
func checkExtensibleEnumStringType(_ *Call, schema oas.Schema) []Finding {
	key, _ := oas.Entry(schema.Node, "x-extensible-enum")

	return When(key != nil && !extensibleEnumStringTypeIsString(schema.Node), func() Finding {
		return Finding{
			Node:    schema.Node,
			Path:    schema.Path,
			Message: "schemas with x-extensible-enum must have type string",
		}
	})
}

// extensibleEnumStringTypeIsString reports whether string is the only
// non-null type of the schema.
func extensibleEnumStringTypeIsString(schema *yaml.Node) bool {
	return oas.HasType(schema, "string") && !slices.ContainsFunc(oas.Types(schema), func(declared string) bool {
		return declared != literalString && declared != keywordNull
	})
}
