package rules

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	operationIDMaxPrefixesMaxOption = "max"
	operationIDMaxPrefixesDefault   = 5
)

// operationIDMaxPrefixesPatterns maps each convention to the expression that
// captures the first word of an identifier written in it. The lowercase
// convention has no word boundary and so no prefix.
func operationIDMaxPrefixesPatterns() map[casing.Casing]*regexp.Regexp {
	return map[casing.Casing]*regexp.Regexp{
		casing.Camel:          regexp.MustCompile(`^([a-z]+)[A-Z]`),
		casing.Pascal:         regexp.MustCompile(`^([A-Z][a-z]+)[A-Z]`),
		casing.Snake:          regexp.MustCompile(`^([a-z]+)_`),
		casing.Kebab:          regexp.MustCompile(`^([a-z]+)-`),
		casing.Dot:            regexp.MustCompile(`^([a-z]+)\.`),
		casing.ScreamingSnake: regexp.MustCompile(`^([A-Z]+)_`),
		casing.ScreamingKebab: regexp.MustCompile(`^([A-Z]+)-`),
	}
}

// OperationIDMaxPrefixes returns the extra-operation-id-max-prefixes rule.
//
// # What it checks
//
// The first word of every operationId in the configured casing, camelCase by
// default (getPet gives get, listPets gives list), is collected across the whole
// document. When more distinct prefixes are used than the allowed maximum, one
// finding is reported on the document root, listing the prefixes. operationIds
// without an uppercase letter after a lowercase prefix, such as list_pets, are
// ignored.
//
// # Why
//
// Consistent verbs make an API predictable and are what tool-calling agents and
// SDK users rely on to guess method names. A document that mixes get, fetch,
// retrieve, read, load and obtain for the same idea is harder to use.
//
// # Where it comes from
//
// ai-api-lint (AI063, operationId verb prefixes should be consistent).
//
// # Options
//
//   - casing (string or list, default "camelCase"): kebab-case, snake_case,
//     camelCase, PascalCase, SCREAMING_SNAKE_CASE, SCREAMING-KEBAB-CASE, dot.case
//     or lowercase. It decides where the first word of an identifier ends, so
//     get_pet gives get under snake_case. An identifier is read in the first
//     listed convention that captures a prefix from it, an empty list collects
//     nothing and an unknown value is reported. lowercase has no word boundary
//     and so no prefix.
//   - max (integer, default 5): the largest number of distinct prefixes allowed.
//
// # Bad
//
//	# six or more distinct prefixes, for example:
//	operationId: getPet
//	operationId: fetchOwner
//	operationId: retrieveVet
//	operationId: loadClinic
//	operationId: readVisit
//	operationId: obtainBill
//
// # Good
//
//	operationId: getPet
//	operationId: listPets
//	operationId: createPet
//
// # Known false positives
//
// APIs that deliberately use many distinct verbs, such as action-style APIs,
// exceed the default; raise max for them.
func OperationIDMaxPrefixes() Rule {
	return New(Spec{
		Name:        "operation-id-max-prefixes",
		Description: "The document must not use more than a few distinct operationId verb prefixes.",
		HowToFix:    "Standardise the operationIds on a smaller set of verb prefixes, or raise the max option.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			casingOption("Convention of the operationId", casing.Camel),
			{
				Name:        operationIDMaxPrefixesMaxOption,
				Description: "Largest number of distinct operationId prefixes.",
				Default:     operationIDMaxPrefixesDefault,
			},
		},
		Check: WithCasing(checkOperationIDMaxPrefixes),
	})
}

// operationIDMaxPrefixesPrefix returns the first word of the identifier as the
// first convention of the set that captures one reads it, or the empty string.
func operationIDMaxPrefixesPrefix(
	patterns map[casing.Casing]*regexp.Regexp,
	allowed casing.Set,
	id string,
) string {
	for _, convention := range allowed {
		pattern, ok := patterns[convention]
		if !ok {
			continue
		}

		if match := pattern.FindStringSubmatch(id); match != nil {
			return match[1]
		}
	}

	return ""
}

// checkOperationIDMaxPrefixes compares the number of prefixes with the maximum.
func checkOperationIDMaxPrefixes(call *Call, allowed casing.Set) []Finding {
	patterns := operationIDMaxPrefixesPatterns()

	prefixes := slices.Compact(slices.Sorted(slices.Values(
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []string {
			prefix := operationIDMaxPrefixesPrefix(patterns, allowed, oas.StringAt(operation.Node, "operationId"))
			if prefix == "" {
				return nil
			}

			return []string{prefix}
		}),
	)))

	limit := call.Int(operationIDMaxPrefixesMaxOption)

	return When(len(prefixes) > limit, func() Finding {
		return Finding{
			Message: "found " + strconv.Itoa(len(prefixes)) + " distinct operationId prefixes (" +
				strings.Join(prefixes, ", ") + ") but at most " + strconv.Itoa(limit) + " are allowed",
		}
	})
}
