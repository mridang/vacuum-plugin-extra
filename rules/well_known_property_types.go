package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// wellKnownPropertyTypesDateTime is the default timestamp format.
const wellKnownPropertyTypesDateTime = "date-time"

// WellKnownPropertyTypes returns the extra-well-known-property-types rule.
//
// # What it checks
//
// Properties that carry the same meaning everywhere must have the same type.
// Properties named id must be strings. Properties named created or modified
// must be strings with the format date-time. Which names are checked and which
// types they get can be set with options. A property whose type is missing, or
// whose type or format differs, is reported. Type lists of OpenAPI 3.1 are
// accepted when, ignoring null, they contain only the expected type.
//
// # Why
//
// Consumers write generic code for identifiers and timestamps. When one resource
// uses an integer id and another a string, or timestamps as numbers in some
// places and date-time strings in others, that code needs special cases and
// generated clients differ between resources.
//
// # Where it comes from
//
// Zally (CommonFieldTypesRule, Zalando guideline on common field names).
//
// # Options
//
//   - idProperties: property names that must be of idType. Default: id.
//   - idType: the type of the identifier properties. Default: string.
//   - timestampProperties: property names that must be timestamps. Default:
//     created, modified.
//   - timestampType: the type of the timestamp properties. Default: string.
//   - timestampFormat: the format of the timestamp properties. Default:
//     date-time.
//
// # Bad
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      properties:
//	        id: {type: integer}
//	        created: {type: string}
//
// # Good
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      properties:
//	        id: {type: string}
//	        created: {type: string, format: date-time}
//
// # Known false positives / limitations
//
// A property called id in a schema where an integer key is intended is reported
// even when that is deliberate. Names are matched exactly and case-sensitively.
// A property that is only a $ref is judged by the type of its target.
func WellKnownPropertyTypes() Rule {
	return New(Spec{
		Name:        "well-known-property-types",
		Description: "Properties named id, created and modified must have the common types.",
		HowToFix:    "Give id the type string, and created and modified the type string with format date-time.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{Name: "idProperties", Description: "Property names that are identifiers.", Default: []string{"id"}},
			{Name: "idType", Description: "The type of identifier properties.", Default: literalString},
			{
				Name:        "timestampProperties",
				Description: "Property names that are timestamps.",
				Default:     []string{"created", "modified"},
			},
			{Name: "timestampType", Description: "The type of timestamp properties.", Default: literalString},
			{
				Name:        "timestampFormat",
				Description: "The format of timestamp properties.",
				Default:     wellKnownPropertyTypesDateTime,
			},
		},
		Check: PerSchema(checkWellKnownPropertyTypes),
	})
}

func checkWellKnownPropertyTypes(call *Call, schema oas.Schema) []Finding {
	return seq.FlatMapPairs(oas.Pairs(oas.Get(schema.Node, "properties")), func(name, property *yaml.Node) []Finding {
		resolved := oas.Resolve(call.Root, property)
		path := oas.JoinAll(schema.Path, "properties", name.Value)

		return slices.Concat(
			wellKnownPropertyTypesCheck(
				slices.Contains(call.Strings("idProperties"), name.Value),
				name.Value, resolved, path, call.String("idType"), "",
			),
			wellKnownPropertyTypesCheck(
				slices.Contains(call.Strings("timestampProperties"), name.Value),
				name.Value, resolved, path, call.String("timestampType"), call.String("timestampFormat"),
			),
		)
	})
}

// wellKnownPropertyTypesCheck compares one property with the expected type and
// format when applies is true. An empty expectation is not checked.
func wellKnownPropertyTypesCheck(
	applies bool,
	name string,
	property *yaml.Node,
	path, wantType, wantFormat string,
) []Finding {
	if !applies {
		return nil
	}

	return slices.Concat(
		When(wantType != "" && !wellKnownPropertyTypesIs(property, wantType), func() Finding {
			return Finding{
				Node:    property,
				Path:    path,
				Message: "property '" + name + "' must be of type " + wantType,
			}
		}),
		When(wantFormat != "" && oas.StringAt(property, "format") != wantFormat, func() Finding {
			return Finding{
				Node:    property,
				Path:    path,
				Message: "property '" + name + "' must have the format " + wantFormat,
			}
		}),
	)
}

// wellKnownPropertyTypesIs reports whether the schema type is the wanted one,
// ignoring null.
func wellKnownPropertyTypesIs(schema *yaml.Node, want string) bool {
	types := seq.Filter(slices.Values(oas.Types(schema)), func(name string) bool {
		return name != keywordNull
	})

	return len(types) == 1 && types[0] == want
}
