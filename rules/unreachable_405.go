package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const unreachable405Methods = "methods"

// Unreachable405 returns the extra-unreachable-405 rule.
//
// # What it checks
//
// An operation must not declare a 405 (Method Not Allowed) response when every
// HTTP method is already declared for its path. The set of methods is
// configurable and by default is get, put, post, delete, options, head, patch
// and trace, the methods OpenAPI knows. Every 405 response of such a path is
// reported.
//
// # Why
//
// A server answers 405 when a request uses a method the resource does not
// implement. If the path declares every method, there is no method left to be
// rejected, so the 405 can never occur and the documented response is
// misleading noise in generated clients and tests.
//
// # Where it comes from
//
// specmatic (unreachable 405 response).
//
// # Options
//
//   - methods (list of strings, default get, put, post, delete, options, head,
//     patch, trace): the methods that must all be declared for the 405 to count
//     as unreachable. Narrow the list to the methods your server really
//     dispatches on.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "405":
//	          description: not allowed
//	    put: {responses: {"200": {description: ok}}}
//	    post: {responses: {"200": {description: ok}}}
//	    delete: {responses: {"200": {description: ok}}}
//	    options: {responses: {"200": {description: ok}}}
//	    head: {responses: {"200": {description: ok}}}
//	    patch: {responses: {"200": {description: ok}}}
//	    trace: {responses: {"200": {description: ok}}}
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "405":
//	          description: not allowed
//
// # Known false positives and limitations
//
// Only the methods declared in the document are known; a server may still answer
// 405 for a method the document declares but a gateway blocks. Path items given
// by $ref are judged by the item they point at. An empty methods list disables
// the rule.
func Unreachable405() Rule {
	return New(Spec{
		Name:        "unreachable-405",
		Description: "A 405 response must not be declared when all HTTP methods are already declared for the path.",
		HowToFix:    "Remove the 405 response, or remove the methods the server does not implement.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        unreachable405Methods,
				Description: "The methods that must all be declared for a 405 to be unreachable.",
				Default:     unreachable405DefaultMethods(),
			},
		},
		Check: OnDocument(checkUnreachable405),
	})
}

// checkUnreachable405 reports 405 responses of paths that declare every method.
func checkUnreachable405(call *Call) []Finding {
	methods := call.Strings(unreachable405Methods)
	if len(methods) == 0 {
		return nil
	}

	return seq.FlatMap(oas.PathItems(call.Root), func(item oas.PathItem) []Finding {
		if !unreachable405Complete(item, methods) {
			return nil
		}

		return seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
			if operation.Route != item.Route {
				return nil
			}

			key, response := oas.Entry(oas.Get(operation.Node, "responses"), "405")

			return When(response != nil, func() Finding {
				return Finding{
					Node:    key,
					Path:    oas.JoinAll(operation.Path, "responses", "405"),
					Message: "405 response is unreachable because every HTTP method is already declared for " + item.Route,
				}
			})
		})
	})
}

// unreachable405Complete reports whether the path item declares every listed method.
func unreachable405Complete(item oas.PathItem, methods []string) bool {
	declared := seq.Map(
		slices.Values(seq.Filter(slices.Values(oas.Keys(item.Node)), oas.IsMethod)),
		strings.ToLower,
	)

	return !slices.ContainsFunc(methods, func(method string) bool {
		return !slices.Contains(declared, Fold(method))
	})
}

// unreachable405DefaultMethods returns every HTTP method known to OpenAPI.
func unreachable405DefaultMethods() []any {
	return seq.Map(slices.Values(oas.Methods()), func(method string) any {
		return method
	})
}
