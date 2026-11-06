package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// enumValueCaseStrTag is the YAML tag of string scalars.
const enumValueCaseStrTag = "!!str"

// EnumValueCase returns the extra-enum-value-case rule.
//
// # What it checks
//
// Every string value of an enum must be written in snake_case or in
// SCREAMING_SNAKE_CASE. Null and non-string values are ignored. A value is
// snake_case when it consists of lower-case letters and digits separated by
// single underscores, and SCREAMING_SNAKE_CASE when it consists of upper-case
// letters and digits separated by single underscores. Mixed case, hyphens,
// spaces and leading, trailing or doubled underscores are reported.
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
//   - allowSnake: accept snake_case values. Default: true.
//   - allowScreamingSnake: accept SCREAMING_SNAKE_CASE values. Default: true.
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
		Description: "String enum values must be snake_case or SCREAMING_SNAKE_CASE.",
		HowToFix:    "Rename the enum value to snake_case or SCREAMING_SNAKE_CASE.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        "allowSnake",
				Description: "Accept snake_case enum values.",
				Default:     true,
			},
			{
				Name:        "allowScreamingSnake",
				Description: "Accept SCREAMING_SNAKE_CASE enum values.",
				Default:     true,
			},
		},
		Check: PerSchema(checkEnumValueCase),
	})
}

func checkEnumValueCase(call *Call, schema oas.Schema) []Finding {
	snake := call.Bool("allowSnake")
	screaming := call.Bool("allowScreamingSnake")
	enumPath := oas.Join(schema.Path, "enum")

	return seq.FlatMapPairs(oas.Items(oas.Get(schema.Node, "enum")), func(i int, value *yaml.Node) []Finding {
		invalid := enumValueCaseIsString(value) && !enumValueCaseAccepted(value.Value, snake, screaming)

		return When(invalid, func() Finding {
			return Finding{
				Node:    value,
				Path:    oas.Index(enumPath, i),
				Message: "enum value '" + value.Value + "' must be " + enumValueCaseWanted(snake, screaming),
			}
		})
	})
}

// enumValueCaseIsString reports whether the node is a string scalar.
func enumValueCaseIsString(node *yaml.Node) bool {
	return oas.IsScalar(node) && !oas.IsNull(node) && (node.Tag == "" || node.Tag == enumValueCaseStrTag)
}

// enumValueCaseWanted describes the accepted casings for a message.
func enumValueCaseWanted(snake, screaming bool) string {
	switch {
	case snake && screaming:
		return "snake_case or SCREAMING_SNAKE_CASE"
	case screaming:
		return "SCREAMING_SNAKE_CASE"
	default:
		return "snake_case"
	}
}

// enumValueCaseAccepted reports whether the value has an accepted casing.
func enumValueCaseAccepted(value string, snake, screaming bool) bool {
	return (snake && enumValueCaseIsSnake(value, false)) || (screaming && enumValueCaseIsSnake(value, true))
}

// enumValueCaseIsSnake reports whether value is made of non-empty words of
// letters and digits joined by single underscores, all in the requested case.
func enumValueCaseIsSnake(value string, upper bool) bool {
	return value != "" && !slices.ContainsFunc(strings.Split(value, "_"), func(word string) bool {
		return word == "" || strings.ContainsFunc(word, func(r rune) bool {
			digit := r >= '0' && r <= '9'
			letter := (!upper && r >= 'a' && r <= 'z') || (upper && r >= 'A' && r <= 'Z')

			return !digit && !letter
		})
	})
}
