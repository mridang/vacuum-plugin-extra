package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const bannedWordsInDocsWordsOption = "words"

// BannedWordsInDocs returns the extra-banned-words-in-docs rule.
//
// # What it checks
//
// Every description, summary and title string in the document is searched for
// the configured words. Matching is case-insensitive and on word boundaries, so
// the word "simple" matches "Simple" and "a simple call" but not "simplest" or
// "simplex". One finding is reported for each banned word found in each string.
//
// # Why
//
// Words such as "simply", "just", "obviously" or "easy" are subjective, patronise
// readers and age badly. Teams keep a short list of terms that must not appear
// in published documentation, for style or for legal reasons.
//
// # Where it comes from
//
// openapilint (banned words in documentation fields).
//
// # Options
//
//   - words (list of strings, no default): the banned words. Each entry is matched
//     literally. When the list is empty the rule does nothing.
//
// # Bad
//
//	# words: [simply]
//	paths:
//	  /pets:
//	    get:
//	      summary: Simply list pets
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      summary: List pets
//
// # Known false positives
//
// Example payloads that contain a key named description, summary or title are
// also scanned, so banned words inside example data are reported. A banned word
// that contains non-word characters, such as "c++", only matches where word
// boundaries exist around its first and last character.
func BannedWordsInDocs() Rule {
	return New(Spec{
		Name:        "banned-words-in-docs",
		Description: "Descriptions, summaries and titles must not contain the configured banned words.",
		HowToFix:    "Reword the text so that it no longer uses the banned word.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        bannedWordsInDocsWordsOption,
			Description: "List of words that must not appear in documentation strings. Empty disables the rule.",
		}},
		Check: OnDocument(checkBannedWordsInDocs),
	})
}

// bannedWordsInDocsKeys returns the keys whose scalar values are documentation.
func bannedWordsInDocsKeys() []string {
	return strings.Fields("description summary title")
}

// checkBannedWordsInDocs reports every documentation string that contains a
// banned word. It reports nothing when no words are configured.
func checkBannedWordsInDocs(call *Call) []Finding {
	patterns := bannedWordsInDocsPatterns(call.Strings(bannedWordsInDocsWordsOption))

	return seq.FlatMap(oas.Walk(call.Root, "$"), func(node oas.Node) []Finding {
		if !slices.Contains(bannedWordsInDocsKeys(), node.Key) || !oas.IsScalar(node.Node) {
			return nil
		}

		return seq.FlatMapSlice(patterns, func(banned bannedWordsInDocsPattern) []Finding {
			return When(banned.pattern.MatchString(node.Node.Value), func() Finding {
				return Finding{
					Node:    node.Node,
					Path:    node.Path,
					Message: "documentation must not use the banned word '" + banned.word + "'",
				}
			})
		})
	})
}

// bannedWordsInDocsPattern pairs a banned word with its matching expression.
type bannedWordsInDocsPattern struct {
	word    string
	pattern *regexp.Regexp
}

// bannedWordsInDocsPatterns compiles one whole-word expression per non-blank
// word.
func bannedWordsInDocsPatterns(words []string) []bannedWordsInDocsPattern {
	trimmed := seq.Map(slices.Values(words), strings.TrimSpace)
	nonBlank := seq.Filter(slices.Values(trimmed), func(word string) bool {
		return word != ""
	})

	return seq.Map(slices.Values(nonBlank), func(word string) bannedWordsInDocsPattern {
		return bannedWordsInDocsPattern{
			word:    word,
			pattern: regexp.MustCompile(`(?i)(?:^|\W)` + regexp.QuoteMeta(word) + `(?:\W|$)`),
		}
	})
}
