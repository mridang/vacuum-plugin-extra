package rules

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// enumValueCaseStrTag is the YAML tag of string scalars.
const enumValueCaseStrTag = "!!str"

// EnumValueCase returns the extra-enum-value-case rule.
//
// # What it checks
//
// Every string value of an enum must follow one of the configured case
// conventions, snake_case or SCREAMING_SNAKE_CASE by default. Null and
// non-string values are ignored. A value is snake_case when it consists of
// lower-case letters and digits separated by single underscores, and
// SCREAMING_SNAKE_CASE when it consists of upper-case letters and digits
// separated by single underscores. Mixed case, hyphens, spaces and leading,
// trailing or doubled underscores are reported by those defaults.
//
// # Why
//
// Enum values become constants and variants in generated clients. A uniform
// casing maps cleanly onto the naming rules of every target language, while
// values such as "in-progress" or "InProgress" need per language conversion
// tables.
//
// # Where it comes from
//
// The Oxide OpenAPI lint (enum-value-case), which accepts snake_case and
// SCREAMING_SNAKE_CASE as produced by the heck crate.
//
// # Options
//
//   - casing (string or list, default [snake_case, SCREAMING_SNAKE_CASE]):
//     kebab-case, snake_case, camelCase, PascalCase, SCREAMING_SNAKE_CASE,
//     SCREAMING-KEBAB-CASE or lowercase. A value is accepted when it follows
//     any listed convention, an empty list disables the rule and an unknown
//     value is reported.
//
// # Bad
//
//	components:
//	  schemas:
//	    Status:
//	      type: string
//	      enum: [in-progress, Done]
//
// # Good
//
//	components:
//	  schemas:
//	    Status:
//	      type: string
//	      enum: [in_progress, done]
//
// # Known false positives / limitations
//
// Enums whose values are fixed by an external standard (media types, country
// codes, HTTP methods) are reported even though they cannot be renamed. Suppress
// the rule for those schemas.
func EnumValueCase() Rule {
	return New(Spec{
		Name:        "enum-value-case",
		Description: "String enum values must follow the configured case convention, snake_case or SCREAMING_SNAKE_CASE by default.",
		HowToFix:    "Rename the enum value to the configured case convention.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			casingOption("Case conventions of the enum values", casing.Snake, casing.ScreamingSnake),
		},
		Check: WithCasing(checkEnumValueCase),
	})
}

func checkEnumValueCase(call *Call, allowed casing.Set) []Finding {
	return PerSchema(func(_ *Call, schema oas.Schema) []Finding {
		enumPath := oas.Join(schema.Path, "enum")

		return seq.FlatMapPairs(oas.Items(oas.Get(schema.Node, "enum")), func(i int, value *yaml.Node) []Finding {
			invalid := enumValueCaseIsString(value) && !allowed.Matches(value.Value)

			return When(invalid, func() Finding {
				return Finding{
					Node:    value,
					Path:    oas.Index(enumPath, i),
					Message: "enum value '" + value.Value + "' must be " + allowed.String(),
				}
			})
		})
	})(call)
}

// enumValueCaseIsString reports whether the node is a string scalar.
func enumValueCaseIsString(node *yaml.Node) bool {
	return oas.IsScalar(node) && !oas.IsNull(node) && (node.Tag == "" || node.Tag == enumValueCaseStrTag)
}
