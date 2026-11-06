package rules

import (
	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// arrayRequiresItemsAllowPrefix is the option that accepts prefixItems.
const arrayRequiresItemsAllowPrefix = "allowPrefixItems"

// ArrayRequiresItems returns the extra-array-requires-items rule.
//
// # What it checks
//
// Every schema that declares type: array (or, in OpenAPI 3.1, a type list that
// contains "array") must define an items keyword. In OpenAPI 3.1 a schema that
// defines prefixItems is accepted as well unless the option is switched off.
//
// # Why
//
// OpenAPI 3.0 makes items mandatory for arrays. Without it, generators cannot
// pick an element type and fall back to untyped lists, and consumers cannot
// validate the elements.
//
// # Where it comes from
//
// The scalar "array-items-required" rule and the Spectral
// "array-requires-items" rule. The two rows of the source list are merged.
//
// # Options
//
//   - allowPrefixItems (bool, default true): treat prefixItems as a valid
//     replacement for items.
//
// # Bad
//
//	components:
//	  schemas:
//	    Tags:
//	      type: array
//
// # Good
//
//	components:
//	  schemas:
//	    Tags:
//	      type: array
//	      items:
//	        type: string
//
// # Known false positives / limitations
//
// A schema that is an array only through allOf composition (type: array in a
// referenced parent, items in the child) is judged per schema, so a child that
// repeats type: array without items is flagged even when the parent has them.
func ArrayRequiresItems() Rule {
	return New(Spec{
		Name:        "array-requires-items",
		Description: "Schemas of type array must define items.",
		HowToFix:    "Add an items schema to the array schema.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        arrayRequiresItemsAllowPrefix,
				Description: "Accept prefixItems as a replacement for items.",
				Default:     true,
			},
		},
		Check: PerSchema(checkArrayRequiresItems),
	})
}

// checkArrayRequiresItems reports array schemas without items.

func checkArrayRequiresItems(call *Call, schema oas.Schema) []Finding {
	allowPrefix := call.Bool(arrayRequiresItemsAllowPrefix)
	prefixed := allowPrefix && oas.Has(schema.Node, "prefixItems")
	missing := oas.HasType(schema.Node, "array") && !oas.Has(schema.Node, "items") && !prefixed

	return When(missing, func() Finding {
		return Finding{
			Node:    schema.Node,
			Path:    schema.Path,
			Message: "schema of type array must define items",
		}
	})
}
