package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const noNullablePropertiesMergePatch = "application/merge-patch+json"

// NoNullableProperties returns the extra-no-nullable-properties rule.
//
// # What it checks
//
// A property must not be nullable. A property is nullable when its schema, after
// local $ref resolution, sets nullable: true (OpenAPI 3.0) or lists null in its
// type (OpenAPI 3.1). Properties are exempt inside request bodies of media type
// application/merge-patch+json, where null means "remove this field".
//
// # Why
//
// Nullable properties force every client to handle three states (absent, null
// and a value). Omitting an optional property conveys the same information
// with fewer special cases and without the generator quirks that null causes.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (ibm-no-nullable-properties).
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
//	    nickname:
//	      type: string
//	      nullable: true
//
// # Good
//
//	Pet:
//	  type: object
//	  properties:
//	    nickname:
//	      type: string
//
// # Known false positives
//
// A component schema that is only used by a merge-patch request body is
// flagged, because the exemption applies to schemas written inline below the
// merge-patch media type. Properties that become nullable through oneOf with a
// null branch are not detected.
func NoNullableProperties() Rule {
	return New(Spec{
		Name:        "no-nullable-properties",
		Description: "Properties must not be nullable, except in merge-patch request bodies.",
		HowToFix:    "Remove nullable: true or the null type and make the property optional instead.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerSchema(checkNoNullableProperties),
	})
}

// checkNoNullableProperties reports nullable properties.
func checkNoNullableProperties(call *Call, schema oas.Schema) []Finding {
	if strings.Contains(schema.Path, noNullablePropertiesMergePatch) {
		return nil
	}

	return seq.FlatMapPairs(oas.Pairs(oas.Get(schema.Node, "properties")), func(name, property *yaml.Node) []Finding {
		return When(noNullablePropertiesNullable(oas.Resolve(call.Root, property)), func() Finding {
			return Finding{
				Node:    property,
				Path:    oas.JoinAll(schema.Path, "properties", name.Value),
				Message: "property '" + name.Value + "' must not be nullable",
			}
		})
	})
}

// noNullablePropertiesNullable reports whether a schema accepts null.
func noNullablePropertiesNullable(schema *yaml.Node) bool {
	return oas.IsTrue(oas.Get(schema, "nullable")) || oas.HasType(schema, "null")
}
