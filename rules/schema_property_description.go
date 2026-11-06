package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// SchemaPropertyDescription returns the extra-schema-property-description rule.
//
// # What it checks
//
// Every property of every schema must have a description that is not empty and
// not just whitespace. A property written as a $ref satisfies the rule when its
// own node or the referenced schema has a description. The finding is reported
// on the property.
//
// # Why
//
// Property names alone rarely explain units, formats, nullability or intent.
// Generated reference documentation and SDK comments are only as good as the
// descriptions they are built from.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (ibm-schema-description / property description
// requirement).
//
// # Options
//
// None.
//
// # Bad
//
//	Pet:
//	  type: object
//	  properties:
//	    name:
//	      type: string
//
// # Good
//
//	Pet:
//	  type: object
//	  properties:
//	    name:
//	      type: string
//	      description: The display name of the pet.
//
// # Known false positives
//
// Properties that are composed with allOf and take their description from a
// branch are reported, because only the property itself and its direct $ref
// target are inspected.
func SchemaPropertyDescription() Rule {
	return New(Spec{
		Name:        "schema-property-description",
		Description: "Every schema property must have a non-empty description.",
		HowToFix:    "Add a description to the property.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerSchema(checkSchemaPropertyDescription),
	})
}

// checkSchemaPropertyDescription reports properties without a description.
func checkSchemaPropertyDescription(call *Call, schema oas.Schema) []Finding {
	return seq.FlatMapPairs(oas.Pairs(oas.Get(schema.Node, "properties")), func(name, property *yaml.Node) []Finding {
		return When(
			!schemaPropertyDescriptionPresent(oas.Text(oas.Get(property, "description"))) &&
				!schemaPropertyDescriptionPresent(oas.StringAt(oas.Resolve(call.Root, property), "description")),
			func() Finding {
				return Finding{
					Node:    property,
					Path:    oas.JoinAll(schema.Path, "properties", name.Value),
					Message: "property '" + name.Value + "' must have a description",
				}
			},
		)
	})
}

// schemaPropertyDescriptionPresent reports whether text is a real description.
func schemaPropertyDescriptionPresent(text string) bool {
	return strings.TrimSpace(text) != ""
}
