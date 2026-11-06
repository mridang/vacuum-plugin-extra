package rules

import (
	"regexp"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

const operationDescriptionOpensWithSentencePattern = `^(?:#{1,6}(?:\s|$)|>|[-*+]\s|\d+[.)]\s|\||` + "```" + `|~~~|<[A-Za-z/!]|\t| {4})`

// OperationDescriptionOpensWithSentence returns the extra-operation-description-opens-with-sentence rule.
//
// # What it checks
//
// The description of an operation must open with a plain sentence: its first
// non-blank line must not start with Markdown or HTML structure. The flagged
// openings are a heading (#), a block quote (>), a list item (-, *, +, 1. or
// 1)), a table row (|), a code fence (``` or ~~~), an indented code block (a
// tab or four spaces) and an HTML tag.
//
// # Why
//
// Many tools use the beginning of the description as a short excerpt: reference
// documentation shows the first line in navigation and search results, and
// language-model tooling uses it as the operation blurb. An opening heading,
// list or code block yields an empty or garbled excerpt.
//
// # Where it comes from
//
// Redocly CLI (operation-description-opens-with-sentence).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      description: |
//	        ## Overview
//	        Lists the pets.
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      description: |
//	        Lists the pets.
//
//	        ## Details
//
// # Known false positives and limitations
//
// Only the start of the description is examined, with a line based test, so a
// description that begins with a sentence which happens to start with a number
// and a period ("2. Run it") is read as a list. Operations with no description
// are not reported.
func OperationDescriptionOpensWithSentence() Rule {
	return New(Spec{
		Name:        "operation-description-opens-with-sentence",
		Description: "Operation descriptions must open with a sentence, not a heading, quote, list, table, code fence or HTML.",
		HowToFix:    "Start the description with a plain sentence and move structured content after it.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkOperationDescriptionOpensWithSentence),
	})
}

// checkOperationDescriptionOpensWithSentence reports descriptions that open with markup.
func checkOperationDescriptionOpensWithSentence(_ *Call, operation oas.Operation) []Finding {
	key, description := oas.Entry(operation.Node, "description")

	return When(
		oas.IsScalar(description) &&
			regexp.MustCompile(operationDescriptionOpensWithSentencePattern).
				MatchString(operationDescriptionOpensWithSentenceFirstLine(description.Value)),
		func() Finding {
			return Finding{
				Node:    description,
				Path:    oas.Join(operation.Path, key.Value),
				Message: "operation description must open with a sentence, not a heading, quote, list, table, code fence or HTML",
			}
		},
	)
}

// operationDescriptionOpensWithSentenceFirstLine returns the first line that has content, keeping its indentation.
func operationDescriptionOpensWithSentenceFirstLine(text string) string {
	for line := range strings.SplitSeq(text, "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimRight(line, "\r")
		}
	}

	return ""
}
