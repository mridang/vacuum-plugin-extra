package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

const (
	patternAnchoredStart = "requireStart"
	patternAnchoredEnd   = "requireEnd"
)

// PatternAnchored returns the extra-pattern-anchored rule.
//
// # What it checks
//
// Every pattern keyword of a string schema must start with ^ and end with an
// unescaped $. Schemas are found in parameters, request bodies, responses and
// components, nested properties, items and the members of allOf, anyOf and
// oneOf included. A schema without a type is checked as well (it is typically a
// composition member); a schema whose type is declared and does not include
// string is skipped.
//
// # Why
//
// JSON Schema patterns are not anchored: they match anywhere in the value.
// A pattern such as [a-z]+ accepts "123abc456", which is almost never intended,
// and unanchored patterns make validators and generated clients disagree with
// the author. Anchoring states exactly which values are accepted.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-pattern-properties and string-attributes patterns
// guidance; the anchoring requirement).
//
// # Options
//
//   - requireStart (bool, default true): the pattern must start with ^.
//   - requireEnd (bool, default true): the pattern must end with an unescaped $.
//
// # Bad
//
//	components:
//	  schemas:
//	    Code:
//	      type: string
//	      pattern: "[A-Z]{3}"
//
// # Good
//
//	components:
//	  schemas:
//	    Code:
//	      type: string
//	      pattern: "^[A-Z]{3}$"
//
// # Known false positives and limitations
//
// Patterns that are deliberately unanchored for substring matching are
// flagged; disable requireStart or requireEnd, or suppress the rule. Only
// the first and last character are examined: an alternation such as ^a|b$ is
// accepted although it is not anchored as a whole. The pattern is never
// compiled.
func PatternAnchored() Rule {
	return New(Spec{
		Name:        "pattern-anchored",
		Description: "String schema patterns must be anchored with ^ and $.",
		HowToFix:    "Start the pattern with ^ and end it with $.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{Name: patternAnchoredStart, Description: "Require the pattern to start with ^.", Default: true},
			{Name: patternAnchoredEnd, Description: "Require the pattern to end with $.", Default: true},
		},
		Check: PerSchema(checkPatternAnchored),
	})
}

// checkPatternAnchored reports unanchored patterns of string schemas.
func checkPatternAnchored(call *Call, schema oas.Schema) []Finding {
	key, pattern := oas.Entry(schema.Node, "pattern")
	if !oas.IsScalar(pattern) || !patternAnchoredIsString(schema.Node) {
		return nil
	}

	message := patternAnchoredProblem(pattern.Value, call.Bool(patternAnchoredStart), call.Bool(patternAnchoredEnd))

	return When(message != "", func() Finding {
		return Finding{Node: pattern, Path: oas.Join(schema.Path, key.Value), Message: message}
	})
}

// patternAnchoredIsString reports whether the schema is a string or has no declared type.
func patternAnchoredIsString(schema *yaml.Node) bool {
	return len(oas.Types(schema)) == 0 || oas.HasType(schema, "string")
}

// patternAnchoredProblem describes what is wrong with the pattern, or returns an empty string.
func patternAnchoredProblem(pattern string, needStart, needEnd bool) string {
	startOK := !needStart || strings.HasPrefix(pattern, "^")
	endOK := !needEnd || patternAnchoredEndsWithDollar(pattern)

	switch {
	case !startOK && !endOK:
		return "pattern must start with ^ and end with $"
	case !startOK:
		return "pattern must start with ^"
	case !endOK:
		return "pattern must end with $"
	default:
		return ""
	}
}

// patternAnchoredEndsWithDollar reports whether the pattern ends in a dollar that is not escaped.
func patternAnchoredEndsWithDollar(pattern string) bool {
	body, found := strings.CutSuffix(pattern, "$")
	if !found {
		return false
	}

	backslashes := len(body) - len(strings.TrimRight(body, `\`))

	return backslashes%2 == 0
}
