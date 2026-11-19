package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
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
// getUser or listOrders. The casing option selects another convention, so with
// snake_case the verb must be followed by an underscore, as in get_user. The default verbs are get, list, create, update,
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
//   - casing (string or list, default "camelCase"): kebab-case, snake_case,
//     camelCase, PascalCase, SCREAMING_SNAKE_CASE, SCREAMING-KEBAB-CASE, dot.case
//     or lowercase. It decides what must follow the verb: an upper-case letter in
//     camelCase and PascalCase (where the verb is capitalised too), the
//     separator in the others and any character in lowercase. An identifier
//     passes when it follows any listed convention, an empty list disables the
//     rule and an unknown value is reported.
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
// An operationId in snake_case or kebab-case (list_pets) never matches under the
// default casing, because the verb must be followed by an upper-case letter. Set
// the casing option to the convention of the API. A bare verb such as "list" is also reported.
func OperationIDVerbPrefix() Rule {
	return New(Spec{
		Name:        "operation-id-verb-prefix",
		Description: "An operationId must start with a common verb followed by an upper-case letter.",
		HowToFix:    "Rename the operationId so that it starts with a verb such as get, list, create, update or delete.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{casingOption("Convention of the operationId", casing.Camel), {
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
		Check: WithCasing(checkOperationIDVerbPrefix),
	})
}

func checkOperationIDVerbPrefix(call *Call, allowed casing.Set) []Finding {
	if len(allowed) == 0 {
		return nil
	}

	return PerOperation(func(call *Call, operation oas.Operation) []Finding {
		return operationIDVerbPrefixOperation(call, operation, allowed)
	})(call)
}

// operationIDVerbPrefixOperation checks the operationId of one operation.
func operationIDVerbPrefixOperation(call *Call, operation oas.Operation, allowed casing.Set) []Finding {
	verbs := call.Strings("prefixes")
	if len(verbs) == 0 {
		return nil
	}

	key, value := oas.Entry(operation.Node, "operationId")

	return When(
		oas.IsScalar(value) && value.Value != "" && !operationIDVerbPrefixMatches(value.Value, verbs, allowed),
		func() Finding {
			return Finding{
				Node: value,
				Path: oas.Join(operation.Path, key.Value),
				Message: "operationId '" + value.Value + "' must start with a verb such as " + strings.Join(
					verbs[:min(len(verbs), operationIDVerbPrefixExamples)],
					", ",
				) + operationIDVerbPrefixTail(allowed),
			}
		},
	)
}

// operationIDVerbPrefixTail finishes the message with what must follow the verb.
func operationIDVerbPrefixTail(allowed casing.Set) string {
	if slices.Equal(allowed, casing.Set{casing.Camel}) {
		return " followed by an upper-case letter"
	}

	return " written in " + allowed.String()
}

// operationIDVerbPrefixExamples is the number of verbs quoted in a message.
const operationIDVerbPrefixExamples = 5

// operationIDVerbPrefixMatches reports whether the identifier starts with one
// of the verbs as the first word of a convention of the set.
func operationIDVerbPrefixMatches(identifier string, verbs []string, allowed casing.Set) bool {
	return slices.ContainsFunc(verbs, func(verb string) bool {
		return slices.ContainsFunc(allowed, func(convention casing.Casing) bool {
			return convention.HasWordPrefix(identifier, verb)
		})
	})
}
