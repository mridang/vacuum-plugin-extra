package rules

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

const (
	descriptionUsageHintWordsOption     = "hintWords"
	descriptionUsageHintMinLengthOption = "minLength"
	descriptionUsageHintDefaultLength   = 30
)

// DescriptionUsageHintWording returns the extra-description-usage-hint-wording rule.
//
// # What it checks
//
// An operation description (the summary is not considered) that is at least
// minLength characters long must contain at least one usage hint word, such as
// "returns", "allows" or "use". The comparison is a case-insensitive substring
// test. Operations without a description, or with one shorter than minLength,
// are not checked here; their absence is a different rule's concern.
//
// # Why
//
// Descriptions are what people and language-model agents read to decide when to
// call an operation. A description that never says what the operation returns,
// allows or is used for makes that decision harder.
//
// # Where it comes from
//
// ai-api-lint (AI013, description lacks usage hint words).
//
// # Options
//
//   - hintWords (list of strings, default: use, call, returns, when, used to,
//     allows, enables, provides, retrieves, creates, updates, deletes): the
//     accepted hint words. An empty list disables the rule.
//   - minLength (integer, default 30): descriptions shorter than this are skipped.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      description: The collection of every pet in the store.
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      description: Returns the collection of every pet in the store.
//
// # Known false positives and limitations
//
// Matching is by substring, so "use" is found inside words such as "because"
// and "house". Descriptions in languages other than English will not contain the
// default words; configure hintWords for them.
func DescriptionUsageHintWording() Rule {
	return New(Spec{
		Name:        "description-usage-hint-wording",
		Description: "Operation descriptions should contain a usage hint word such as returns, allows or use.",
		HowToFix:    "Reword the description to say what the operation returns, allows or is used for.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        descriptionUsageHintWordsOption,
				Description: "Words of which at least one must appear in the description.",
				Default: []any{
					"use", "call", "returns", "when", "used to", "allows",
					"enables", "provides", "retrieves", "creates", "updates", "deletes",
				},
			},
			{
				Name:        descriptionUsageHintMinLengthOption,
				Description: "Minimum description length that is checked.",
				Default:     descriptionUsageHintDefaultLength,
			},
		},
		Check: PerOperation(checkDescriptionUsageHintWording),
	})
}

// checkDescriptionUsageHintWording reports a long description without a hint word.
func checkDescriptionUsageHintWording(call *Call, operation oas.Operation) []Finding {
	words := call.Strings(descriptionUsageHintWordsOption)
	if len(words) == 0 {
		return nil
	}

	key, node := oas.Entry(operation.Node, "description")
	if !oas.IsScalar(node) {
		return nil
	}

	text := strings.TrimSpace(node.Value)

	return When(
		text != "" &&
			utf8.RuneCountInString(text) >= call.Int(descriptionUsageHintMinLengthOption) &&
			!descriptionUsageHintContains(text, words),
		func() Finding {
			return Finding{
				Node:    node,
				Path:    oas.Join(operation.Path, key.Value),
				Message: "operation description lacks usage hint words (for example 'returns', 'allows', 'use')",
			}
		},
	)
}

// descriptionUsageHintContains reports whether the text contains any of the words.
func descriptionUsageHintContains(text string, words []string) bool {
	lower := strings.ToLower(text)

	return slices.ContainsFunc(words, func(word string) bool {
		return word != "" && strings.Contains(lower, strings.ToLower(word))
	})
}
