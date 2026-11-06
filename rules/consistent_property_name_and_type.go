package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// ConsistentPropertyNameAndType returns the extra-consistent-property-name-and-type rule.
//
// # What it checks
//
// Properties that have the same name in different schemas must be declared
// with the same type. The first declaration of a name, in document order,
// fixes the expected type; every later property of that name with another type
// is reported. The type of a property is the sorted set of its declared
// types after local $ref resolution (so "integer" and ["integer", "null"] are
// different types). Properties without a type, and properties whose $ref
// cannot be resolved, are ignored.
//
// # Why
//
// A field called "id" that is a string in one schema and an integer in another
// forces clients to special-case the field and usually points to an accidental
// modelling mistake.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (ibm-property-consistent-name-and-type). It is off
// by default there and is a warning here.
//
// # Options
//
//   - ignore (list of strings, default empty): property names that are exempt.
//
// # Bad
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      properties:
//	        id: {type: string}
//	    Owner:
//	      type: object
//	      properties:
//	        id: {type: integer}
//
// # Good
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      properties:
//	        id: {type: string}
//	    Owner:
//	      type: object
//	      properties:
//	        id: {type: string}
//
// # Known false positives
//
// Names that are deliberately overloaded, such as "value" or "data", are
// flagged; list them in the ignore option. A schema that is reachable twice is
// only inspected once.
func ConsistentPropertyNameAndType() Rule {
	return New(Spec{
		Name:        "consistent-property-name-and-type",
		Description: "Properties with the same name in different schemas must have the same type.",
		HowToFix:    "Align the type of the property with its earlier declaration, or rename the property.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        "ignore",
			Description: "Property names that are exempt from the check.",
		}},
		Check: OnDocument(checkConsistentPropertyNameAndType),
	})
}

// consistentPropertyNameAndTypeUse is one declaration of a named property.
type consistentPropertyNameAndTypeUse struct {
	name  string
	kind  string
	path  string
	value *yaml.Node
}

// checkConsistentPropertyNameAndType reports every property whose type differs
// from the type of the first property of the same name.
func checkConsistentPropertyNameAndType(call *Call) []Finding {
	ignored := call.Strings("ignore")

	uses := seq.FlatMap(oas.Schemas(call.Root), func(schema oas.Schema) []consistentPropertyNameAndTypeUse {
		return seq.FlatMapPairs(oas.Pairs(oas.Get(schema.Node, "properties")),
			func(name, value *yaml.Node) []consistentPropertyNameAndTypeUse {
				kind := consistentPropertyNameAndTypeKind(call.Root, value)

				if kind == "" || slices.Contains(ignored, name.Value) {
					return nil
				}

				return []consistentPropertyNameAndTypeUse{{
					name:  name.Value,
					kind:  kind,
					path:  oas.JoinAll(schema.Path, "properties", name.Value),
					value: value,
				}}
			})
	})
	first := seq.GroupBy(slices.Values(uses), func(use consistentPropertyNameAndTypeUse) string {
		return use.name
	})

	return seq.FlatMapSlice(uses, func(use consistentPropertyNameAndTypeUse) []Finding {
		earliest := first[use.name][0]

		return When(earliest.kind != use.kind, func() Finding {
			return Finding{
				Node: use.value,
				Path: use.path,
				Message: "property '" + use.name + "' has type " + use.kind + " but type " + earliest.kind +
					" at " + earliest.path,
			}
		})
	})
}

// consistentPropertyNameAndTypeKind returns the sorted, joined types of the
// resolved property schema.
func consistentPropertyNameAndTypeKind(root, property *yaml.Node) string {
	types := slices.Clone(oas.Types(oas.Resolve(root, property)))
	slices.Sort(types)

	return strings.Join(types, "|")
}
