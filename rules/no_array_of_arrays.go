package rules

import (
	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// NoArrayOfArrays returns the extra-no-array-of-arrays rule.
//
// # What it checks
//
// An array schema must not use an array schema as its items. Nested arrays
// cannot be extended with further members, are awkward in most generated
// clients and make the payload shape hard to document; wrapping the inner array
// in an object gives it a name and room to grow. The items schema is followed
// through a local $ref before it is inspected.
//
// # Where it comes from
//
// IBM's openapi-validator (ibm-no-array-of-arrays).
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Matrix:
//	      type: array
//	      items:
//	        type: array
//	        items:
//	          type: number
//
// # Good
//
//	components:
//	  schemas:
//	    Matrix:
//	      type: array
//	      items:
//	        type: object
//	        properties:
//	          values:
//	            type: array
//	            items:
//	              type: number
//
// # Known false positives
//
// Geometry formats such as GeoJSON coordinate lists are inherently arrays of
// arrays and are flagged; suppress the rule for those schemas.
func NoArrayOfArrays() Rule {
	return New(Spec{
		Name:        "no-array-of-arrays",
		Description: "Array schemas must not use another array schema as their items.",
		HowToFix:    "Wrap the inner array in an object schema.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerSchema(checkNoArrayOfArrays),
	})
}

func checkNoArrayOfArrays(call *Call, schema oas.Schema) []Finding {
	return When(
		oas.HasType(schema.Node, "array") &&
			oas.HasType(oas.Resolve(call.Root, oas.Get(schema.Node, "items")), "array"),
		func() Finding {
			return Finding{
				Node:    oas.Get(schema.Node, "items"),
				Path:    oas.Join(schema.Path, "items"),
				Message: "an array schema must not have array items",
			}
		},
	)
}
