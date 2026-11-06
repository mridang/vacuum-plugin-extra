package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// Operation2xxRequired returns the extra-operation-2xx-required rule.
//
// # What it checks
//
// Every operation must declare at least one successful response: a response
// key that starts with 2, such as 200, 204 or the range 2XX. Responses that
// only use 3xx, 4xx, 5xx or default do not satisfy the rule. The finding is
// reported on the responses object, or on the operation when it has none.
//
// # Why
//
// An operation with no documented success outcome tells clients nothing about
// what a good call returns, and SDK generators cannot choose a return type.
//
// # Where it comes from
//
// ai-api-lint (the operation must define a 2xx response).
//
// # Options
//
// None.
//
// # Bad
//
//	responses:
//	  "404":
//	    description: Not found
//
// # Good
//
//	responses:
//	  "200":
//	    description: ok
//
// # Known false positives
//
// Redirect-only endpoints, which answer 302 by design, and webhook callbacks
// documented with a default response are flagged. Operations below webhooks and
// callbacks are not inspected.
func Operation2xxRequired() Rule {
	return New(Spec{
		Name:        "operation-2xx-required",
		Description: "Every operation must declare a 2xx response.",
		HowToFix:    "Add a 2xx response such as 200, 201, 204 or 2XX to the operation.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerOperation(checkOperation2xxRequired),
	})
}

// checkOperation2xxRequired reports operations without a success response.
func checkOperation2xxRequired(call *Call, operation oas.Operation) []Finding {
	if operation2xxRequiredHasSuccess(call, operation) {
		return nil
	}

	responses := oas.Get(operation.Node, "responses")
	if responses == nil {
		return One(Finding{
			Node:    operation.Node,
			Path:    operation.Path,
			Message: "operation must declare a 2xx response",
		})
	}

	return One(Finding{
		Node:    responses,
		Path:    oas.Join(operation.Path, "responses"),
		Message: "operation must declare a 2xx response",
	})
}

func operation2xxRequiredHasSuccess(call *Call, operation oas.Operation) bool {
	return slices.ContainsFunc(oas.Responses(call.Root, operation), func(response oas.Response) bool {
		return strings.HasPrefix(response.Code, "2")
	})
}
