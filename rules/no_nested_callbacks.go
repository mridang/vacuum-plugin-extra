package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// NoNestedCallbacks returns the extra-no-nested-callbacks rule.
//
// # What it checks
//
// A callback must not contain callbacks of its own. The operations of the path
// items inside a callback (and inside components/callbacks) are searched for a
// callbacks object, and each one found is reported, at any depth.
//
// Nested callbacks make a webhook flow hard to follow, are poorly supported by
// documentation renderers and code generators, and usually indicate that a
// workflow should be modelled as separate operations.
//
// # Where it comes from
//
// Spectral, where the OpenAPI ruleset discourages callbacks inside callbacks.
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /subscribe:
//	    post:
//	      callbacks:
//	        onEvent:
//	          "{$request.body#/url}":
//	            post:
//	              callbacks:
//	                onAck:
//	                  "{$request.body#/ack}":
//	                    post:
//	                      responses: {}
//
// # Good
//
//	paths:
//	  /subscribe:
//	    post:
//	      callbacks:
//	        onEvent:
//	          "{$request.body#/url}":
//	            post:
//	              responses: {}
//
// # Known false positives
//
// Callbacks and path items that are reached through $ref are inspected only
// where they are defined (components/callbacks), not at the place that refers
// to them.
func NoNestedCallbacks() Rule {
	return New(Spec{
		Name:        "no-nested-callbacks",
		Description: "A callback must not contain its own callbacks.",
		HowToFix:    "Remove the nested callbacks, or model the workflow with separate operations.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkNoNestedCallbacks),
	})
}

func checkNoNestedCallbacks(call *Call) []Finding {
	return Concat(
		PerOperation(func(_ *Call, operation oas.Operation) []Finding {
			return noNestedCallbacksInOperation(operation.Node, operation.Path, false)
		}),
		OnDocument(func(call *Call) []Finding {
			return seq.FlatMapPairs(
				oas.Pairs(oas.Dig(call.Root, "components", "callbacks")),
				func(name, callback *yaml.Node) []Finding {
					return noNestedCallbacksInCallback(callback, oas.JoinAll("$.components.callbacks", name.Value))
				},
			)
		}),
	)(call)
}

func noNestedCallbacksInOperation(operation *yaml.Node, path string, nested bool) []Finding {
	callbacks, callbacksNode := oas.Entry(operation, "callbacks")
	if callbacks == nil {
		return nil
	}

	callbacksPath := oas.Join(path, "callbacks")

	return slices.Concat(
		When(nested, func() Finding {
			return Finding{
				Node:    callbacksNode,
				Path:    callbacksPath,
				Message: "callbacks must not contain nested callbacks",
			}
		}),
		seq.FlatMapPairs(oas.Pairs(callbacksNode), func(name, callback *yaml.Node) []Finding {
			return noNestedCallbacksInCallback(callback, oas.Join(callbacksPath, name.Value))
		}),
	)
}

func noNestedCallbacksInCallback(callback *yaml.Node, path string) []Finding {
	return seq.FlatMapPairs(oas.Pairs(callback), func(expression, item *yaml.Node) []Finding {
		return seq.FlatMapPairs(oas.Pairs(item), func(method, operation *yaml.Node) []Finding {
			if !oas.IsMethod(method.Value) {
				return nil
			}

			return noNestedCallbacksInOperation(operation, oas.JoinAll(path, expression.Value, method.Value), true)
		})
	})
}
