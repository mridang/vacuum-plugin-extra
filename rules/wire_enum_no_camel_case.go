package rules

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// WireEnumNoCamelCase returns the extra-wire-enum-no-camel-case rule.
//
// # What it checks
//
// String values of a schema's enum must not contain a lower-case letter that is
// immediately followed by an upper-case letter, which is the signature of
// camelCase and PascalCase spellings (for example inProgress or created_atUTC).
// SCREAMING_SNAKE_CASE, snake_case and kebab-case values are accepted.
//
// # Why
//
// Enum values are part of the wire format and are fixed forever once clients
// depend on them. Code generators derive constant names from them, and
// camelCase values convert badly to the idiomatic constant style of other
// languages, so a separator-based spelling is the safer choice.
//
// # Where it comes from
//
// Redocly CLI (wire-enum-no-camel-case, which tests enum values against the
// expression [a-z][A-Z]).
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Status:
//	      type: string
//	      enum: [inProgress, done]
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
// Values that are not camelCase but contain a lower-to-upper transition, such
// as product names (iPhone), are reported. Non-string enum values are ignored.
func WireEnumNoCamelCase() Rule {
	return New(Spec{
		Name:        "wire-enum-no-camel-case",
		Description: "Enum values must not be written in camelCase.",
		HowToFix:    "Spell the enum value in snake_case or SCREAMING_SNAKE_CASE.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerSchema(checkWireEnumNoCamelCase),
	})
}

// checkWireEnumNoCamelCase reports camelCase enum values.
func checkWireEnumNoCamelCase(_ *Call, schema oas.Schema) []Finding {
	return seq.FlatMapPairs(oas.Items(oas.Get(schema.Node, "enum")), func(i int, value *yaml.Node) []Finding {
		return When(
			oas.IsScalar(value) && value.Tag != "!!int" && value.Tag != "!!float" &&
				wireEnumNoCamelCaseMatches(value.Value),
			func() Finding {
				return Finding{
					Node:    value,
					Path:    oas.Index(oas.Join(schema.Path, "enum"), i),
					Message: "enum value " + value.Value + " must not be camelCase",
				}
			},
		)
	})
}

// wireEnumNoCamelCaseMatches reports a lower-case letter followed by an upper-case one.
func wireEnumNoCamelCaseMatches(value string) bool {
	runes := []rune(value)
	for i := range len(runes) - 1 {
		if runes[i] >= 'a' && runes[i] <= 'z' && runes[i+1] >= 'A' && runes[i+1] <= 'Z' {
			return true
		}
	}

	return false
}
