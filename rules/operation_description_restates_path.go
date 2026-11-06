package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// operationDescriptionRestatesPathDescription is the description keyword.
const operationDescriptionRestatesPathDescription = "description"

// OperationDescriptionRestatesPath returns the extra-operation-description-restates-path rule.
//
// # What it checks
//
// The description and the summary of an operation must say more than the
// request line. They are reported when, lower-cased and trimmed, they equal the
// path alone, the method followed by a space and the path ("get /pets"), or the
// method directly followed by the path ("get/pets").
//
// # Why
//
// A description that repeats the method and path tells readers nothing they do
// not already see in the document, and is a typical placeholder left behind by
// generators. Agents and developers rely on descriptions to decide what an
// operation does.
//
// # Where it comes from
//
// ai-api-lint (AI012, description repeats the HTTP method and path).
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
//	      summary: GET /pets
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      summary: List all pets
//
// # Known false positives / limitations
//
// Only exact restatements are found; "Gets /pets" or a description that merely
// contains the path is accepted.
func OperationDescriptionRestatesPath() Rule {
	return New(Spec{
		Name:        "operation-description-restates-path",
		Description: "Operation descriptions and summaries must not merely restate the method and path.",
		HowToFix:    "Describe what the operation does instead of repeating its method and path.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkOperationDescriptionRestatesPath),
	})
}

func checkOperationDescriptionRestatesPath(_ *Call, operation oas.Operation) []Finding {
	return seq.FlatMapSlice(
		[]string{operationDescriptionRestatesPathDescription, "summary"},
		func(keyword string) []Finding {
			key, value := oas.Entry(operation.Node, keyword)

			return When(
				oas.IsScalar(value) && operationDescriptionRestatesPathMatches(operation, value.Value),
				func() Finding {
					return Finding{
						Node:    value,
						Path:    oas.Join(operation.Path, key.Value),
						Message: "operation " + keyword + " only restates the method and path",
					}
				},
			)
		},
	)
}

// operationDescriptionRestatesPathMatches reports whether text is the path or
// the method and path.
func operationDescriptionRestatesPathMatches(operation oas.Operation, text string) bool {
	folded := Fold(text)
	route := strings.ToLower(operation.Route)

	return folded == route ||
		folded == operation.Method+" "+route ||
		folded == operation.Method+route
}
