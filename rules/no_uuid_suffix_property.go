package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// noUUIDSuffixPropertySuffix is the property name suffix that is reported.
const noUUIDSuffixPropertySuffix = "_uuid"

// NoUUIDSuffixProperty returns the extra-no-uuid-suffix-property rule.
//
// # What it checks
//
// A property whose name ends in _uuid and whose schema is an inline plain
// string with format uuid (no pattern, enum, minLength or maxLength, and not a
// $ref) is flagged: the name should end in _id instead.
//
// # Why
//
// Identifiers are opaque to clients. Encoding the representation (uuid) in the
// name leaks an implementation detail and makes it a breaking change to switch
// to another identifier format later, so project_uuid should be project_id.
//
// # Where it comes from
//
// The Oxide OpenAPI lint (uuid properties must not end with _uuid).
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Instance:
//	      type: object
//	      properties:
//	        project_uuid:
//	          type: string
//	          format: uuid
//
// # Good
//
//	components:
//	  schemas:
//	    Instance:
//	      type: object
//	      properties:
//	        project_id:
//	          type: string
//	          format: uuid
//
// # Known false positives / limitations
//
// Properties that reference a shared uuid schema with $ref, or constrain the
// string with a pattern or length, are intentionally not reported (matching
// the source linter), even though they carry the same naming problem.
func NoUUIDSuffixProperty() Rule {
	return New(Spec{
		Name:        "no-uuid-suffix-property",
		Description: "UUID properties must not have names ending in _uuid.",
		HowToFix:    "Rename the property to end in _id.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerSchema(checkNoUUIDSuffixProperty),
	})
}

// checkNoUUIDSuffixProperty reports plain uuid strings named with _uuid.
func checkNoUUIDSuffixProperty(_ *Call, schema oas.Schema) []Finding {
	return seq.FlatMapPairs(oas.Pairs(oas.Get(schema.Node, "properties")), func(name, property *yaml.Node) []Finding {
		violation := strings.HasSuffix(name.Value, noUUIDSuffixPropertySuffix) &&
			noUUIDSuffixPropertyPlainUUID(property)

		return When(violation, func() Finding {
			return Finding{
				Node: name,
				Path: oas.JoinAll(schema.Path, "properties", name.Value),
				Message: "property " + name.Value + " is a uuid; rename it to " + strings.TrimSuffix(
					name.Value,
					noUUIDSuffixPropertySuffix,
				) + "_id",
			}
		})
	})
}

// noUUIDSuffixPropertyPlainUUID reports an inline unconstrained uuid string.
func noUUIDSuffixPropertyPlainUUID(schema *yaml.Node) bool {
	return oas.IsMap(schema) && !oas.Has(schema, "$ref") && oas.HasType(schema, "string") &&
		oas.StringAt(schema, "format") == literalUUID &&
		!slices.ContainsFunc(
			[]string{literalPattern, keywordEnum, keywordMinLength, keywordMaxLength},
			func(key string) bool {
				return oas.Has(schema, key)
			},
		)
}
