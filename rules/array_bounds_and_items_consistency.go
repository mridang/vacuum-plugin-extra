package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const arrayBoundsAndItemsConsistencyRequireBounds = "requireBounds"

// ArrayBoundsAndItemsConsistency returns the extra-array-bounds-and-items-consistency rule.
//
// # What it checks
//
// Every array schema (type array, in any schema reachable from the paths and
// the components) must define an items schema, should define numeric minItems
// and maxItems, and must not set minItems above maxItems. Attributes are also
// looked up inside allOf members of the schema, so a composed array is judged
// by the sum of its parts.
//
// # Why
//
// An array without items accepts anything and generates untyped client code. An
// array without bounds invites unbounded payloads, which is both a robustness
// and a denial-of-service risk, and contradicting bounds make the schema
// impossible to satisfy.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-array-attributes).
//
// # Options
//
//   - requireBounds (bool, default true): also require numeric minItems and
//     maxItems. With false only items and the minItems/maxItems ordering are
//     checked.
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
//	      minItems: 0
//	      maxItems: 100
//	      items:
//	        type: string
//
// # Known false positives and limitations
//
// A schema reached only through a $ref is judged where it is defined. Arrays
// whose type is declared only in an anyOf or oneOf member are not recognised as
// arrays. The IBM rule's additional complaints about enum on arrays and about
// array keywords on non-array schemas are covered by other rules.
func ArrayBoundsAndItemsConsistency() Rule {
	return New(Spec{
		Name:        "array-bounds-and-items-consistency",
		Description: "Array schemas must define items and should define consistent minItems and maxItems.",
		HowToFix:    "Add an items schema and numeric minItems and maxItems with minItems not above maxItems.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        arrayBoundsAndItemsConsistencyRequireBounds,
				Description: "Require numeric minItems and maxItems on every array schema.",
				Default:     true,
			},
		},
		Check: PerSchema(checkArrayBoundsAndItemsConsistency),
	})
}

// checkArrayBoundsAndItemsConsistency inspects one array schema.
func checkArrayBoundsAndItemsConsistency(call *Call, schema oas.Schema) []Finding {
	if !oas.HasType(schema.Node, "array") || oas.UnderNegation(schema.Path) {
		return nil
	}

	items := arrayBoundsAndItemsConsistencyLookup(call.Root, schema.Node, "items")
	minItems, hasMin := oas.Number(arrayBoundsAndItemsConsistencyLookup(call.Root, schema.Node, "minItems"))
	maxItems, hasMax := oas.Number(arrayBoundsAndItemsConsistencyLookup(call.Root, schema.Node, "maxItems"))
	requireBounds := call.Bool(arrayBoundsAndItemsConsistencyRequireBounds)

	messages := slices.Concat(
		arrayBoundsAndItemsConsistencyIf(!oas.IsMap(items), "array schemas must specify an 'items' schema"),
		arrayBoundsAndItemsConsistencyIf(requireBounds && !hasMin, "array schemas should define a numeric 'minItems'"),
		arrayBoundsAndItemsConsistencyIf(requireBounds && !hasMax, "array schemas should define a numeric 'maxItems'"),
		arrayBoundsAndItemsConsistencyIf(hasMin && hasMax && minItems > maxItems,
			"'minItems' cannot be greater than 'maxItems'"),
	)

	return seq.Map(slices.Values(messages), func(message string) Finding {
		return Finding{Node: schema.Node, Path: schema.Path, Message: message}
	})
}

// arrayBoundsAndItemsConsistencyIf returns the message when the condition holds.
func arrayBoundsAndItemsConsistencyIf(condition bool, message string) []string {
	if !condition {
		return nil
	}

	return []string{message}
}

// arrayBoundsAndItemsConsistencyLookup finds a keyword on a schema or in its allOf members.
func arrayBoundsAndItemsConsistencyLookup(root, schema *yaml.Node, key string) *yaml.Node {
	if value := oas.Get(schema, key); value != nil {
		return value
	}

	for _, member := range oas.Items(oas.Get(schema, "allOf")) {
		resolved := oas.Resolve(root, member)
		if resolved == schema {
			continue
		}

		if value := oas.Get(resolved, key); value != nil {
			return value
		}
	}

	return nil
}
