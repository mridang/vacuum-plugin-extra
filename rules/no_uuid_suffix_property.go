package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// noUUIDSuffixPropertyWord is the last word of a property name that is reported.
const noUUIDSuffixPropertyWord = "uuid"

// NoUUIDSuffixProperty returns the extra-no-uuid-suffix-property rule.
//
// # What it checks
//
// A property whose name ends in the word uuid, written in the configured
// casing (_uuid in the default snake_case, Uuid or UUID in camelCase), and whose schema is an inline plain
// string with format uuid (no pattern, enum, minLength or maxLength, and not a
// $ref) is flagged: the name should end in id (_id, Id or ID) instead.
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
//   - casing (string or list, default "snake_case"): kebab-case, snake_case,
//     camelCase, PascalCase, SCREAMING_SNAKE_CASE, SCREAMING-KEBAB-CASE, dot.case
//     or lowercase. It decides how the uuid word is attached to a name. A name
//     is reported when it ends in the word in any listed convention, an empty
//     list disables the rule and an unknown value is reported.
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
		HowToFix:    "Rename the property to end in id, such as project_id.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options:     []Option{casingOption("Convention of the property name", casing.Snake)},
		Check:       WithCasing(checkNoUUIDSuffixProperty),
	})
}

// checkNoUUIDSuffixProperty reports plain uuid strings named with a uuid suffix.
func checkNoUUIDSuffixProperty(call *Call, allowed casing.Set) []Finding {
	return PerSchema(func(_ *Call, schema oas.Schema) []Finding {
		return seq.FlatMapPairs(
			oas.Pairs(oas.Get(schema.Node, "properties")),
			func(name, property *yaml.Node) []Finding {
				renamed, found := noUUIDSuffixPropertyRename(name.Value, allowed)

				return When(found && noUUIDSuffixPropertyPlainUUID(property), func() Finding {
					return Finding{
						Node:    name,
						Path:    oas.JoinAll(schema.Path, "properties", name.Value),
						Message: "property " + name.Value + " is a uuid; rename it to " + renamed,
					}
				})
			},
		)
	})(call)
}

// noUUIDSuffixPropertyRename replaces the uuid suffix of a name with the id
// suffix of the same convention, and reports whether the name had one.
func noUUIDSuffixPropertyRename(name string, allowed casing.Set) (string, bool) {
	for _, convention := range allowed {
		suffixes := convention.Suffixes(noUUIDSuffixPropertyWord)
		ids := convention.Suffixes("id")

		for i, suffix := range suffixes {
			if base, found := strings.CutSuffix(name, suffix); found {
				return base + ids[i], true
			}
		}
	}

	return "", false
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
