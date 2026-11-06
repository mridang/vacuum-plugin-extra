package rules

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	operationIDMaxPrefixesMaxOption = "max"
	operationIDMaxPrefixesDefault   = 5
)

// operationIDMaxPrefixesPattern matches the lowercase prefix of a camelCase id.
const operationIDMaxPrefixesPattern = `^([a-z]+)[A-Z]`

// OperationIDMaxPrefixes returns the extra-operation-id-max-prefixes rule.
//
// # What it checks
//
// The lowercase prefix before the first uppercase letter of every operationId
// (getPet gives get, listPets gives list) is collected across the whole
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
		Options: []Option{{
			Name:        operationIDMaxPrefixesMaxOption,
			Description: "Largest number of distinct operationId prefixes.",
			Default:     operationIDMaxPrefixesDefault,
		}},
		Check: OnDocument(checkOperationIDMaxPrefixes),
	})
}

// checkOperationIDMaxPrefixes compares the number of prefixes with the maximum.
func checkOperationIDMaxPrefixes(call *Call) []Finding {
	pattern := regexp.MustCompile(operationIDMaxPrefixesPattern)

	prefixes := slices.Compact(slices.Sorted(slices.Values(
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []string {
			match := pattern.FindStringSubmatch(oas.StringAt(operation.Node, "operationId"))
			if match == nil {
				return nil
			}

			return []string{match[1]}
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
