package rules

import (
	"strconv"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	stringLengthLimitsMinFloor  = "minLengthFloor"
	stringLengthLimitsCeiling   = "lengthCeiling"
	stringLengthLimitsDefaultMN = 1
	stringLengthLimitsFourMiB   = 4 * 1024 * 1024
)

// StringLengthLimits returns the extra-string-length-limits rule.
//
// # What it checks
//
// The minLength and maxLength keywords of every schema must be integers inside
// a sensible range:
//
//   - minLength must be an integer of at least minLengthFloor (default 1, a
//     positive number).
//   - minLength and maxLength must not exceed lengthCeiling (default 4194304,
//     4 MiB).
//
// Setting either option to zero or less switches that part of the check off.
//
// # Why
//
// A minLength of zero (or a negative one) says nothing, since every string is
// at least that long, and usually comes from a generator that spells out
// defaults. Lengths beyond a few mebibytes are not meaningful for a string field
// that travels inside a JSON document, and tooling that allocates test data from
// the bound may run out of memory.
//
// # Where it comes from
//
// specmatic, merging min-length-positive (minLength must be a positive integer)
// and string-max-length-ceiling (the 4 MB recommended maximum for string
// lengths).
//
// # Options
//
//   - minLengthFloor (integer, default 1): the smallest allowed minLength. Set
//     to 0 to accept minLength: 0 and only reject negative values and
//     non-integers.
//   - lengthCeiling (integer, default 4194304): the largest allowed minLength or
//     maxLength.
//
// # Bad
//
//	components:
//	  schemas:
//	    Name:
//	      type: string
//	      minLength: 0
//	      maxLength: 1073741824
//
// # Good
//
//	components:
//	  schemas:
//	    Name:
//	      type: string
//	      minLength: 1
//	      maxLength: 255
//
// # Known false positives and limitations
//
// With the default floor an explicit minLength: 0 is reported even though it is
// harmless; lower the floor to 0 for such documents. Lengths are only checked
// on schemas reachable from paths and components.
func StringLengthLimits() Rule {
	return New(Spec{
		Name:        "string-length-limits",
		Description: "minLength must be a positive integer and string lengths must not exceed 4 MiB.",
		HowToFix:    "Use an integer minLength of at least 1 and keep minLength and maxLength within the ceiling.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        stringLengthLimitsMinFloor,
				Description: "The smallest allowed minLength; zero or less disables the check.",
				Default:     stringLengthLimitsDefaultMN,
			},
			{
				Name:        stringLengthLimitsCeiling,
				Description: "The largest allowed minLength or maxLength; zero or less disables the check.",
				Default:     stringLengthLimitsFourMiB,
			},
		},
		Check: PerSchema(checkStringLengthLimits),
	})
}

// checkStringLengthLimits inspects the length keywords of every schema.
func checkStringLengthLimits(call *Call, schema oas.Schema) []Finding {
	floor := call.Int(stringLengthLimitsMinFloor)
	ceiling := call.Int(stringLengthLimitsCeiling)

	return seq.FlatMapSlice([]string{keywordMinLength, keywordMaxLength}, func(keyword string) []Finding {
		return stringLengthLimitsKeyword(schema, keyword, floor, ceiling)
	})
}

// stringLengthLimitsKeyword checks one length keyword of a schema.
func stringLengthLimitsKeyword(schema oas.Schema, keyword string, floor, ceiling int) []Finding {
	key, node := oas.Entry(schema.Node, keyword)
	if !oas.IsScalar(node) {
		return nil
	}

	message := stringLengthLimitsMessage(node, keyword, floor, ceiling)

	return When(message != "", func() Finding {
		return Finding{Node: key, Path: oas.Join(schema.Path, keyword), Message: message}
	})
}

// stringLengthLimitsMessage describes what is wrong with the keyword value, or returns an empty string.
func stringLengthLimitsMessage(node *yaml.Node, keyword string, floor, ceiling int) string {
	value, isInt := oas.Int(node)

	switch {
	case !isInt:
		return keyword + " must be an integer"
	case keyword == keywordMinLength && floor > 0 && value < floor:
		return "minLength must be at least " + strconv.Itoa(floor)
	case ceiling > 0 && value > ceiling:
		return keyword + " must not exceed " + strconv.Itoa(ceiling)
	default:
		return ""
	}
}
