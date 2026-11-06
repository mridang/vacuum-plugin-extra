package rules

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// Verbs that appear more than once in this rule.
const (
	operationIDVerbPrefixCreate = "create"
	operationIDVerbPrefixDelete = "delete"
)

// OperationIDVerbPrefix returns the extra-operation-id-verb-prefix rule.
//
// # What it checks
//
// When an operation has an operationId, it must start with one of a list of
// verbs, written in lower case and followed by an upper-case letter, as in
// getUser or listOrders. The default verbs are get, list, create, update,
// delete, remove, search, find, fetch, add, set, check, verify, send, upload,
// download, export, batch and bulk. Operations without an operationId are
// ignored; use the operation-operationId rule to require one.
//
// # Why
//
// Generated SDK methods, MCP tools and agent function names are derived from
// the operationId. A leading verb makes the name read as an action, and a fixed
// vocabulary keeps the names predictable across the API.
//
// # Where it comes from
//
// ai-api-lint (AI002, operationId should start with a common verb).
//
// # Options
//
//   - prefixes: the accepted verbs. Default: get, list, create, update, delete,
//     remove, search, find, fetch, add, set, check, verify, send, upload,
//     download, export, batch, bulk.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      operationId: pets
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      operationId: listPets
//
// # Known false positives / limitations
//
// An operationId in snake_case or kebab-case (list_pets) never matches, because
// the verb must be followed by an upper-case letter. Rename the identifiers or
// disable the rule for such APIs. A bare verb such as "list" is also reported.
func OperationIDVerbPrefix() Rule {
	return New(Spec{
		Name:        "operation-id-verb-prefix",
		Description: "An operationId must start with a common verb followed by an upper-case letter.",
		HowToFix:    "Rename the operationId so that it starts with a verb such as get, list, create, update or delete.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        "prefixes",
			Description: "The verbs an operationId may start with.",
			Default: []string{
				"get",
				keywordList,
				operationIDVerbPrefixCreate,
				"update",
				operationIDVerbPrefixDelete,
				"remove",
				"search",
				"find",
				"fetch",
				"add",
				"set",
				"check",
				"verify",
				"send",
				"upload",
				"download",
				"export",
				"batch",
				"bulk",
			},
		}},
		Check: PerOperation(checkOperationIDVerbPrefix),
	})
}

func checkOperationIDVerbPrefix(call *Call, operation oas.Operation) []Finding {
	verbs := call.Strings("prefixes")
	if len(verbs) == 0 {
		return nil
	}

	key, value := oas.Entry(operation.Node, "operationId")

	return When(
		oas.IsScalar(value) && value.Value != "" && !operationIDVerbPrefixMatches(value.Value, verbs),
		func() Finding {
			return Finding{
				Node: value,
				Path: oas.Join(operation.Path, key.Value),
				Message: "operationId '" + value.Value + "' must start with a verb such as " + strings.Join(
					verbs[:min(len(verbs), operationIDVerbPrefixExamples)],
					", ",
				) + " followed by an upper-case letter",
			}
		},
	)
}

// operationIDVerbPrefixExamples is the number of verbs quoted in a message.
const operationIDVerbPrefixExamples = 5

// operationIDVerbPrefixMatches reports whether the identifier starts with one
// of the verbs followed by an upper-case letter.
func operationIDVerbPrefixMatches(identifier string, verbs []string) bool {
	return slices.ContainsFunc(verbs, func(verb string) bool {
		rest, found := strings.CutPrefix(identifier, verb)
		if !found || rest == "" {
			return false
		}

		first, _ := utf8.DecodeRuneInString(rest)

		return unicode.IsUpper(first)
	})
}
