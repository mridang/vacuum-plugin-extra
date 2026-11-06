package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// operationIDReservedWordOption is the name of the option holding the words.
const operationIDReservedWordOption = "words"

// OperationIDReservedWord returns the extra-operation-id-reserved-word rule.
//
// # What it checks
//
// The operationId of an operation must not be a reserved word. By default the
// reserved words are the keywords of Python (the list reported by
// keyword.kwlist: False, None, True, and, as, assert, async, await, break,
// class, continue, def, del, elif, else, except, finally, for, from, global,
// if, import, in, is, lambda, nonlocal, not, or, pass, raise, return, try,
// while, with, yield). Code generators turn operation ids into function names,
// so an id such as import or class yields code that cannot be compiled or
// imported. The comparison is exact and case-sensitive, as the keywords are.
//
// # Where it comes from
//
// ai-api-lint, which rejects operation ids that are Python keywords.
//
// # Options
//
//   - words: list of reserved words. Default: the Python keywords above.
//     Supply a different list to protect other languages; an empty list
//     disables the rule.
//
// # Bad
//
//	paths:
//	  /imports:
//	    get:
//	      operationId: import
//
// # Good
//
//	paths:
//	  /imports:
//	    get:
//	      operationId: listImports
//
// # Known false positives
//
// Keywords of languages other than Python are not checked unless configured,
// and a word that is only reserved in some Python versions is always flagged.
func OperationIDReservedWord() Rule {
	return New(Spec{
		Name:        "operation-id-reserved-word",
		Description: "The operationId must not be a reserved word of the target language.",
		HowToFix:    "Rename the operationId so that it is not a reserved word.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{{
			Name:        operationIDReservedWordOption,
			Description: "Reserved words that an operationId must not equal.",
			Default:     operationIDReservedWordPython(),
		}},
		Check: PerOperation(checkOperationIDReservedWord),
	})
}

// operationIDReservedWordPython returns the reserved words of Python.
func operationIDReservedWordPython() []string {
	return []string{
		"False", "None", "True", "and", "as", "assert", "async", "await", "break", "class",
		"continue", "def", "del", "elif", "else", "except", "finally", "for", "from", "global",
		"if", "import", "in", "is", "lambda", "nonlocal", keywordNot, "or", "pass", "raise",
		"return", "try", "while", "with", "yield",
	}
}

// checkOperationIDReservedWord reports operations whose operationId equals a
// configured reserved word. It does nothing when no words are configured.
func checkOperationIDReservedWord(call *Call, operation oas.Operation) []Finding {
	key, id := oas.Entry(operation.Node, "operationId")

	return When(
		oas.IsScalar(id) && slices.Contains(call.Strings(operationIDReservedWordOption), id.Value),
		func() Finding {
			return Finding{
				Node:    id,
				Path:    oas.Join(operation.Path, key.Value),
				Message: "operationId " + id.Value + " is a reserved word",
			}
		},
	)
}
