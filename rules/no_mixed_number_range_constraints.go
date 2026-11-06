package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// NoMixedNumberRangeConstraints returns the extra-no-mixed-number-range-constraints rule.
//
// # What it checks
//
// A port of Redocly CLI's no-mixed-number-range-constraints. A schema is
// reported when maximum and exclusiveMaximum are both numbers, and when minimum
// and exclusiveMinimum are both numbers. The finding is placed on the
// exclusive keyword and reads "Schema should not have both `maximum` and
// `exclusiveMaximum`. Use one or the other." (or the minimum equivalent).
//
// The test is the one of the original: the values must be numbers
// (typeof number). In OpenAPI 3.0 exclusiveMaximum and exclusiveMinimum are
// booleans that modify maximum and minimum; that spelling is therefore never
// reported, and neither is a bound given as a string. The original applies the
// check to every OpenAPI version, so a 3.0 document that uses the numeric
// 3.1 form next to the inclusive bound is reported too.
//
// # Why
//
// Numeric exclusive bounds come from JSON Schema 2020-12 (OpenAPI 3.1). Having
// both an inclusive and an exclusive bound for the same side is redundant at
// best, and tools disagree about which one wins when they differ.
//
// # Where it comes from
//
// Redocly CLI, rule no-mixed-number-range-constraints
// (packages/core/src/rules/common/no-mixed-number-range-constraints.ts).
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Age:
//	      type: integer
//	      maximum: 10
//	      exclusiveMaximum: 10
//
// # Good
//
//	components:
//	  schemas:
//	    Age:
//	      type: integer
//	      exclusiveMaximum: 10
//
// # Known false positives and limitations
//
// Bound values are not compared, exactly as in the original: two bounds that
// agree are reported as well as bounds that differ.
func NoMixedNumberRangeConstraints() Rule {
	return New(Spec{
		Name:        "no-mixed-number-range-constraints",
		Description: "Schemas must not define both an inclusive and a numeric exclusive bound on the same side.",
		HowToFix:    "Keep either the inclusive or the exclusive bound for the side that is reported.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkNoMixedNumberRangeConstraints),
	})
}

// noMixedNumberRangeConstraintsPair names an inclusive and exclusive keyword.
type noMixedNumberRangeConstraintsPair struct {
	inclusive string
	exclusive string
}

// checkNoMixedNumberRangeConstraints reports schemas with both bounds on a side.
func checkNoMixedNumberRangeConstraints(_ *Call, schema oas.Schema) []Finding {
	pairs := []noMixedNumberRangeConstraintsPair{
		{inclusive: keywordMaximum, exclusive: keywordExclusiveMaximum},
		{inclusive: keywordMinimum, exclusive: keywordExclusiveMinimum},
	}

	return seq.FlatMapSlice(pairs, func(pair noMixedNumberRangeConstraintsPair) []Finding {
		return When(noMixedNumberRangeConstraintsMixed(schema.Node, pair), func() Finding {
			return Finding{
				Node:    oas.Get(schema.Node, pair.exclusive),
				Path:    oas.Join(schema.Path, pair.exclusive),
				Message: "Schema should not have both `" + pair.inclusive + "` and `" + pair.exclusive + "`. Use one or the other.",
			}
		})
	})
}

// noMixedNumberRangeConstraintsMixed reports an inclusive bound next to an
// exclusive bound when both are numbers.
func noMixedNumberRangeConstraintsMixed(schema *yaml.Node, pair noMixedNumberRangeConstraintsPair) bool {
	return noMixedNumberRangeConstraintsIsNumber(oas.Get(schema, pair.exclusive)) &&
		noMixedNumberRangeConstraintsIsNumber(oas.Get(schema, pair.inclusive))
}

// noMixedNumberRangeConstraintsIsNumber reports whether the node is a YAML
// number, not a boolean or a string.
func noMixedNumberRangeConstraintsIsNumber(node *yaml.Node) bool {
	_, ok := oas.Number(node)

	return ok && !slices.Contains(
		[]string{noMixedNumberRangeConstraintsStrTag, noMixedNumberRangeConstraintsBoolTag},
		node.Tag,
	)
}

const (
	noMixedNumberRangeConstraintsStrTag  = "!!str"
	noMixedNumberRangeConstraintsBoolTag = "!!bool"
)
