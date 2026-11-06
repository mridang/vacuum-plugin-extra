package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// AdditionalPropertiesObjectOnly returns the extra-additional-properties-object-only rule.
//
// # What it checks
//
// A schema that declares a type other than object must not use the
// additionalProperties keyword. additionalProperties only constrains the extra
// members of an object, so on a string, number, integer, boolean or array
// schema it has no effect and almost always signals a misplaced keyword. A
// schema that declares no type at all is left alone, and so is a schema whose
// type list in OpenAPI 3.1 includes object.
//
// # Where it comes from
//
// Specmatic's OpenAPI validation, which refuses additionalProperties outside
// object schemas.
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Name:
//	      type: string
//	      additionalProperties: false
//
// # Good
//
//	components:
//	  schemas:
//	    Name:
//	      type: object
//	      additionalProperties:
//	        type: string
//
// # Known false positives
//
// A schema that omits type but is clearly an object is not examined, so the
// rule can miss misplaced keywords there. A schema that combines type with
// composition keywords is judged only by its own type.
func AdditionalPropertiesObjectOnly() Rule {
	return New(Spec{
		Name:        "additional-properties-object-only",
		Description: "additionalProperties is only valid inside object schemas.",
		HowToFix:    "Remove additionalProperties or change the schema type to object.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkAdditionalPropertiesObjectOnly),
	})
}

func checkAdditionalPropertiesObjectOnly(_ *Call, schema oas.Schema) []Finding {
	key, value := oas.Entry(schema.Node, "additionalProperties")
	types := oas.Types(schema.Node)

	return When(
		value != nil && len(types) > 0 && !slices.Contains(types, "object"),
		func() Finding {
			return Finding{
				Node: value,
				Path: oas.Join(schema.Path, key.Value),
				Message: "additionalProperties is not allowed on a schema of type " +
					strings.Join(types, ", "),
			}
		},
	)
}
