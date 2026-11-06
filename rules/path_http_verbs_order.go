package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// pathHTTPVerbsOrderOption is the name of the option listing the verb order.
const pathHTTPVerbsOrderOption = "order"

// pathHTTPVerbsOrderDelete is the delete method keyword.
const pathHTTPVerbsOrderDelete = "delete"

// PathHTTPVerbsOrder returns the extra-path-http-verbs-order rule.
//
// # What it checks
//
// Within one path item the operations must appear in the configured order of
// HTTP methods. A fixed order makes documents predictable to read and keeps
// diffs small. Each operation that appears after an operation that should
// follow it is reported at that operation. Methods missing from the configured
// order are ignored.
//
// # Where it comes from
//
// Redocly CLI's path-http-verbs-order rule.
//
// # Options
//
//   - order: list of lower-case HTTP methods. Default: get, head, post, put,
//     patch, delete, options, trace. An empty list disables the rule.
//
// # Bad
//
//	paths:
//	  /pets:
//	    post:
//	      responses: {}
//	    get:
//	      responses: {}
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      responses: {}
//	    post:
//	      responses: {}
//
// # Known false positives
//
// Documents that group operations by importance or lifecycle rather than by
// verb are flagged wholesale; configure the order option or disable the rule.
func PathHTTPVerbsOrder() Rule {
	return New(Spec{
		Name:        "path-http-verbs-order",
		Description: "Operations within a path item must follow the configured HTTP method order.",
		HowToFix:    "Reorder the operations of the path item to follow the configured method order.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        pathHTTPVerbsOrderOption,
				Description: "HTTP methods in the order they must appear.",
				Default: []string{
					"get",
					"head",
					literalPost,
					literalPut,
					literalPatch,
					pathHTTPVerbsOrderDelete,
					"options",
					"trace",
				},
			},
		},
		Check: OnDocument(checkPathHTTPVerbsOrder),
	})
}

// checkPathHTTPVerbsOrder reports an operation that is declared after an
// operation of the same path that the configured order places later.
func checkPathHTTPVerbsOrder(call *Call) []Finding {
	order := pathHTTPVerbsOrderNormalise(call.Strings(pathHTTPVerbsOrderOption))
	operations := slices.Collect(oas.Operations(call.Root))

	return seq.FlatMapIndexed(operations, func(i int, operation oas.Operation) []Finding {
		rank := slices.Index(order, operation.Method)
		declaredEarlier := slices.ContainsFunc(operations[:i], func(earlier oas.Operation) bool {
			return earlier.Route == operation.Route && slices.Index(order, earlier.Method) > rank
		})

		return When(rank >= 0 && declaredEarlier, func() Finding {
			return Finding{
				Node:    operation.Node,
				Path:    operation.Path,
				Message: operation.Method + " must be declared before the operations that follow it in the configured order",
			}
		})
	})
}

// pathHTTPVerbsOrderNormalise lower-cases and trims the configured methods.
func pathHTTPVerbsOrderNormalise(order []string) []string {
	return seq.Map(slices.Values(order), func(method string) string {
		return strings.ToLower(strings.TrimSpace(method))
	})
}
