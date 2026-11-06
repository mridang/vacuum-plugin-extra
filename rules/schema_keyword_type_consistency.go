package rules

import (
	"fmt"
	"maps"
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// schemaKeywordTypeConsistencyRange is a pair of bounds that must be ordered.
type schemaKeywordTypeConsistencyRange struct {
	low    string
	high   string
	strict bool
}

// SchemaKeywordTypeConsistency returns the extra-schema-keyword-type-consistency
// rule.
//
// # What it checks
//
// The validation keywords of a schema must be consistent with its type and with
// each other. Two kinds of problems are reported:
//
//   - A lower bound greater than its upper bound: minimum and maximum,
//     minLength and maxLength, minItems and maxItems, minProperties and
//     maxProperties, minContains and maxContains. The numeric exclusive forms
//     (OpenAPI 3.1) are compared strictly: exclusiveMinimum must be below
//     maximum and exclusiveMaximum, and minimum must be below
//     exclusiveMaximum. In the OpenAPI 3.0 boolean form, exclusiveMinimum or
//     exclusiveMaximum set to true is reported when minimum equals maximum,
//     because the range is then empty.
//   - A keyword that does not apply to the declared type: minimum, maximum,
//     exclusiveMinimum, exclusiveMaximum and multipleOf need integer or number;
//     minLength, maxLength and pattern need string; minItems, maxItems,
//     uniqueItems, items, prefixItems, contains, minContains, maxContains and
//     unevaluatedItems need array; properties, additionalProperties,
//     patternProperties, propertyNames, unevaluatedProperties, minProperties,
//     maxProperties and required (as a list) need object.
//
// Schemas without a declared type are not checked for the second kind of
// problem. A type list in OpenAPI 3.1 allows a keyword when any of its entries
// does.
//
// Such schemas are almost always mistakes, and the keywords are silently
// ignored by validators, so the intended constraint is never enforced.
//
// # Where it comes from
//
// IBM openapi-validator, rule ibm-schema-type-format and the related schema
// consistency checks, which relate numeric, string, array and object keywords
// to the schema type.
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
//	      type: string
//	      minimum: 0
//
// # Good
//
//	components:
//	  schemas:
//	    Age:
//	      type: integer
//	      minimum: 0
//
// # Known false positives
//
// Only keywords declared directly on the schema are inspected, so a keyword
// that is valid through an allOf member is not seen, and the type of an
// untyped schema is not inferred from its allOf members.
func SchemaKeywordTypeConsistency() Rule {
	return New(Spec{
		Name:        "schema-keyword-type-consistency",
		Description: "Schema keywords must match the declared type and lower bounds must not exceed upper bounds.",
		HowToFix:    "Change the type, remove the keyword or fix the bounds.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkSchemaKeywordTypeConsistency),
	})
}

func schemaKeywordTypeConsistencyRanges() []schemaKeywordTypeConsistencyRange {
	return []schemaKeywordTypeConsistencyRange{
		{low: keywordMinimum, high: keywordMaximum},
		{low: keywordExclusiveMinimum, high: keywordMaximum, strict: true},
		{low: keywordMinimum, high: keywordExclusiveMaximum, strict: true},
		{low: keywordExclusiveMinimum, high: keywordExclusiveMaximum, strict: true},
		{low: keywordMinLength, high: keywordMaxLength},
		{low: keywordMinItems, high: keywordMaxItems},
		{low: keywordMinProperties, high: keywordMaxProperties},
		{low: schemaKeywordTypeConsistencyMinContains, high: schemaKeywordTypeConsistencyMaxContains},
	}
}

func schemaKeywordTypeConsistencyKeywords() map[string][]string {
	return map[string][]string{
		keywordNumber: {
			keywordMinimum,
			keywordMaximum,
			keywordExclusiveMinimum,
			keywordExclusiveMaximum,
			"multipleOf",
		},
		literalString: {keywordMinLength, keywordMaxLength, literalPattern},
		"array": {
			keywordMinItems,
			keywordMaxItems,
			"uniqueItems",
			keywordItems,
			schemaKeywordTypeConsistencyPrefixItems,
			schemaKeywordTypeConsistencyContains,
			schemaKeywordTypeConsistencyMinContains,
			schemaKeywordTypeConsistencyMaxContains,
			schemaKeywordTypeConsistencyUnevaluatedItems,
		},
		literalObject: {
			literalProperties,
			schemaKeywordTypeConsistencyAdditional,
			literalPatternproperties,
			schemaKeywordTypeConsistencyPropertyNames,
			keywordUnevaluatedProperties,
			keywordMinProperties,
			keywordMaxProperties,
			"required",
		},
	}
}

func checkSchemaKeywordTypeConsistency(_ *Call, schema oas.Schema) []Finding {
	return slices.Concat(
		schemaKeywordTypeConsistencyBounds(schema),
		schemaKeywordTypeConsistencyApplies(schema),
	)
}

func schemaKeywordTypeConsistencyBounds(schema oas.Schema) []Finding {
	return slices.Concat(
		seq.FlatMapSlice(
			schemaKeywordTypeConsistencyRanges(),
			func(bounds schemaKeywordTypeConsistencyRange) []Finding {
				low, lowOK := oas.Number(oas.Get(schema.Node, bounds.low))
				high, highOK := oas.Number(oas.Get(schema.Node, bounds.high))
				message := schemaKeywordTypeConsistencyOrder(bounds, low, high)

				return When(lowOK && highOK && message != "", func() Finding {
					return Finding{
						Node:    oas.Get(schema.Node, bounds.high),
						Path:    oas.Join(schema.Path, bounds.high),
						Message: message,
					}
				})
			},
		),
		schemaKeywordTypeConsistencyBooleanBounds(schema),
	)
}

// schemaKeywordTypeConsistencyOrder describes how a pair of bounds is out of
// order, or returns the empty string when the bounds are ordered.
func schemaKeywordTypeConsistencyOrder(bounds schemaKeywordTypeConsistencyRange, low, high float64) string {
	switch {
	case bounds.strict && low >= high:
		return fmt.Sprintf("%s must be less than %s", bounds.low, bounds.high)
	case !bounds.strict && low > high:
		return fmt.Sprintf("%s must not be greater than %s", bounds.low, bounds.high)
	default:
		return ""
	}
}

// schemaKeywordTypeConsistencyBooleanBounds handles the OpenAPI 3.0 form in
// which exclusiveMinimum and exclusiveMaximum are booleans that make the
// minimum or maximum bound exclusive.
func schemaKeywordTypeConsistencyBooleanBounds(schema oas.Schema) []Finding {
	low, lowOK := oas.Number(oas.Get(schema.Node, keywordMinimum))
	high, highOK := oas.Number(oas.Get(schema.Node, keywordMaximum))
	if !lowOK || !highOK || low != high {
		return nil
	}

	return seq.FlatMapSlice([]string{keywordExclusiveMinimum, keywordExclusiveMaximum}, func(flag string) []Finding {
		node := oas.Get(schema.Node, flag)
		set := oas.IsScalar(node) && node.Tag == schemaKeywordTypeConsistencyBoolTag && oas.IsTrue(node)

		return When(set, func() Finding {
			return Finding{
				Node:    node,
				Path:    oas.Join(schema.Path, flag),
				Message: fmt.Sprintf("%s makes minimum and maximum an empty range", flag),
			}
		})
	})
}

func schemaKeywordTypeConsistencyApplies(schema oas.Schema) []Finding {
	declared := oas.Types(schema.Node)
	if len(declared) == 0 {
		return nil
	}

	return seq.FlatMapPairs(
		maps.All(schemaKeywordTypeConsistencyKeywords()),
		func(family string, keywords []string) []Finding {
			if schemaKeywordTypeConsistencyHasFamily(declared, family) {
				return nil
			}

			return seq.FlatMapSlice(keywords, func(keyword string) []Finding {
				node := oas.Get(schema.Node, keyword)

				return When(node != nil && schemaKeywordTypeConsistencyApplicable(keyword, node), func() Finding {
					return Finding{
						Node:    node,
						Path:    oas.Join(schema.Path, keyword),
						Message: fmt.Sprintf("%s does not apply to type %v", keyword, declared),
					}
				})
			})
		},
	)
}

func schemaKeywordTypeConsistencyHasFamily(declared []string, family string) bool {
	if family == keywordNumber {
		return slices.Contains(declared, "number") || slices.Contains(declared, "integer")
	}

	return slices.Contains(declared, family)
}

func schemaKeywordTypeConsistencyApplicable(keyword string, node *yaml.Node) bool {
	return keyword != "required" || oas.IsSeq(node)
}

// Keyword names used by this rule.
const (
	schemaKeywordTypeConsistencyAdditional       = "additionalProperties"
	schemaKeywordTypeConsistencyArray            = "array"
	schemaKeywordTypeConsistencyPrefixItems      = "prefixItems"
	schemaKeywordTypeConsistencyUnevaluatedItems = "unevaluatedItems"
	schemaKeywordTypeConsistencyPropertyNames    = "propertyNames"
	schemaKeywordTypeConsistencyContains         = "contains"
	schemaKeywordTypeConsistencyBoolTag          = "!!bool"
	schemaKeywordTypeConsistencyMinContains      = "minContains"
	schemaKeywordTypeConsistencyMaxContains      = "maxContains"
)
