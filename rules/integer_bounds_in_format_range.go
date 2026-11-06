package rules

import (
	"slices"

	"math/big"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	integerBoundsInFormatRangeInt32  = "int32"
	integerBoundsInFormatRangeInt64  = "int64"
	integerBoundsInFormatRangeBase   = 10
	integerBoundsInFormatRangeBits32 = 31
	integerBoundsInFormatRangeBits64 = 63
)

// IntegerBoundsInFormatRange returns the extra-integer-bounds-in-format-range rule.
//
// # What it checks
//
// Every schema of type integer must declare a format of int32 or int64 and, by
// default, both minimum and maximum. When a minimum or maximum is present it
// must lie inside the range of the declared format: -2147483648 to 2147483647
// for int32 and -9223372036854775808 to 9223372036854775807 for int64.
// Findings are reported on the schema for missing keywords and on the keyword
// for out-of-range values.
//
// # Why
//
// Without a format, generators pick whatever integer width they like; without
// bounds, clients and servers disagree on valid input and fuzzers have no
// range to work with. A bound outside the format can never be satisfied.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (ibm-integer-attributes).
//
// # Options
//
//   - requireFormat (boolean, default true): require format int32 or int64.
//   - requireBounds (boolean, default true): require minimum and maximum.
//
// # Bad
//
//	type: integer
//	format: int32
//	minimum: 0
//	maximum: 5000000000
//
// # Good
//
//	type: integer
//	format: int32
//	minimum: 0
//	maximum: 100
//
// # Known false positives
//
// Integer schemas that are intentionally unbounded, such as opaque counters,
// are flagged; disable requireBounds to allow them. Schemas that only use
// exclusiveMinimum or exclusiveMaximum are treated as missing bounds.
func IntegerBoundsInFormatRange() Rule {
	return New(Spec{
		Name:        "integer-bounds-in-format-range",
		Description: "Integer schemas must define an int32 or int64 format and bounds inside its range.",
		HowToFix:    "Add format int32 or int64 and minimum and maximum values within the range of the format.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{Name: "requireFormat", Description: "Require format int32 or int64.", Default: true},
			{Name: "requireBounds", Description: "Require minimum and maximum.", Default: true},
		},
		Check: PerSchema(checkIntegerBoundsInFormatRange),
	})
}

// checkIntegerBoundsInFormatRange inspects all integer schemas.
func checkIntegerBoundsInFormatRange(call *Call, schema oas.Schema) []Finding {
	if !oas.HasType(schema.Node, keywordInteger) {
		return nil
	}

	format := oas.StringAt(schema.Node, "format")

	return slices.Concat(
		integerBoundsInFormatRangeFormat(call, schema, format),
		integerBoundsInFormatRangeBounds(call, schema, format),
	)
}

// integerBoundsInFormatRangeFormat checks the format keyword.
func integerBoundsInFormatRangeFormat(call *Call, schema oas.Schema, format string) []Finding {
	return When(
		call.Bool("requireFormat") && format != integerBoundsInFormatRangeInt32 &&
			format != integerBoundsInFormatRangeInt64,
		func() Finding {
			return Finding{
				Node:    schema.Node,
				Path:    schema.Path,
				Message: "integer schema must define format int32 or int64",
			}
		},
	)
}

// integerBoundsInFormatRangeBounds checks minimum and maximum.
func integerBoundsInFormatRangeBounds(call *Call, schema oas.Schema, format string) []Finding {
	return seq.FlatMapSlice([]string{keywordMinimum, keywordMaximum}, func(keyword string) []Finding {
		key, value := oas.Entry(schema.Node, keyword)

		if value == nil {
			return When(call.Bool("requireBounds"), func() Finding {
				return Finding{
					Node:    schema.Node,
					Path:    schema.Path,
					Message: "integer schema must define " + keyword,
				}
			})
		}

		return When(!integerBoundsInFormatRangeInside(value, format), func() Finding {
			return Finding{
				Node:    value,
				Path:    oas.Join(schema.Path, key.Value),
				Message: keyword + " " + value.Value + " is outside the range of format " + format,
			}
		})
	})
}

// integerBoundsInFormatRangeInside reports whether value fits the format.
func integerBoundsInFormatRangeInside(value *yaml.Node, format string) bool {
	number, ok := new(big.Int).SetString(oas.Text(value), integerBoundsInFormatRangeBase)
	if !ok {
		return true
	}

	low, high, known := integerBoundsInFormatRangeLimits(format)
	if !known {
		return true
	}

	return number.Cmp(low) >= 0 && number.Cmp(high) <= 0
}

// integerBoundsInFormatRangeLimits returns the lowest and highest value of the
// format, and false for a format without a known range.
func integerBoundsInFormatRangeLimits(format string) (*big.Int, *big.Int, bool) {
	switch format {
	case integerBoundsInFormatRangeInt32:
		return big.NewInt(
				-1 << integerBoundsInFormatRangeBits32,
			), big.NewInt(
				1<<integerBoundsInFormatRangeBits32 - 1,
			), true
	case integerBoundsInFormatRangeInt64:
		low, high := integerBoundsInFormatRangeInt64Range()

		return low, high, true
	default:
		return nil, nil, false
	}
}

// integerBoundsInFormatRangeInt64Range returns the limits of int64.
func integerBoundsInFormatRangeInt64Range() (*big.Int, *big.Int) {
	limit := new(big.Int).Lsh(big.NewInt(1), integerBoundsInFormatRangeBits64)

	return new(big.Int).Neg(limit), new(big.Int).Sub(limit, big.NewInt(1))
}
