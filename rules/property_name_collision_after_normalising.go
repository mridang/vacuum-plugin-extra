package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// PropertyNameCollisionAfterNormalising returns the extra-property-name-collision-after-normalising rule.
//
// # What it checks
//
// Within one schema no two property names may become identical once the case
// convention is ignored. Names are normalised by lower-casing them and
// removing underscores, hyphens, dots and spaces, so thingType, thing_type,
// thing-type and ThingType all collide. Such pairs are distinct in JSON but
// collapse to one member in generated code or in clients that convert case,
// silently dropping data. Every property after the first of a colliding group
// is reported at its own key.
//
// # Where it comes from
//
// IBM's openapi-validator (ibm-property-case-collision).
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Thing:
//	      type: object
//	      properties:
//	        thingType: {type: string}
//	        thing_type: {type: string}
//
// # Good
//
//	components:
//	  schemas:
//	    Thing:
//	      type: object
//	      properties:
//	        thingType: {type: string}
//	        thingKind: {type: string}
//
// # Known false positives
//
// Names that differ only in case or separators on purpose (for example a
// pair of legacy and current spellings kept for compatibility) are flagged.
// Only the properties declared directly in one schema are compared, not those
// merged from allOf members.
func PropertyNameCollisionAfterNormalising() Rule {
	return New(Spec{
		Name:        "property-name-collision-after-normalising",
		Description: "Property names within a schema must be unique after normalising their case convention.",
		HowToFix:    "Rename one of the properties so that the names differ by more than case or separators.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkPropertyNameCollisionAfterNormalising),
	})
}

// propertyNameCollisionAfterNormalisingKey returns the name without case and
// separators.
func propertyNameCollisionAfterNormalisingKey(name string) string {
	return strings.ToLower(strings.Map(func(r rune) rune {
		if strings.ContainsRune("_-. ", r) {
			return -1
		}

		return r
	}, name))
}

// checkPropertyNameCollisionAfterNormalising reports every property whose
// normalised name equals that of an earlier property of the same schema.
func checkPropertyNameCollisionAfterNormalising(_ *Call, schema oas.Schema) []Finding {
	base := oas.Join(schema.Path, "properties")
	names := seq.FlatMapPairs(oas.Pairs(oas.Get(schema.Node, "properties")), func(name, _ *yaml.Node) []*yaml.Node {
		return []*yaml.Node{name}
	})

	return seq.FlatMapIndexed(names, func(i int, name *yaml.Node) []Finding {
		key := propertyNameCollisionAfterNormalisingKey(name.Value)
		first := names[slices.IndexFunc(names, func(other *yaml.Node) bool {
			return propertyNameCollisionAfterNormalisingKey(other.Value) == key
		})]

		return When(first != names[i], func() Finding {
			return Finding{
				Node:    name,
				Path:    oas.Join(base, name.Value),
				Message: "property " + name.Value + " collides with " + first.Value + " after normalising case",
			}
		})
	})
}
