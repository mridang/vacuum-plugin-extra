package rules

import (
	"iter"
	"slices"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// counting wraps items so that the number of items consumed is added to the
// counter, which lets a combinator report how many nodes it visited.
func counting[T any](items iter.Seq[T], counter *int) iter.Seq[T] {
	return func(yield func(T) bool) {
		for item := range items {
			*counter++

			if !yield(item) {
				return
			}
		}
	}
}

// visiting runs check over items and records the visited count under kind.
func visiting[T any](call *Call, kind string, items iter.Seq[T], check func(T) []Finding) []Finding {
	count := 0
	findings := seq.FlatMap(counting(items, &count), check)
	call.visited(kind, count)

	return findings
}

// One returns a result holding a single finding. It is the value a per-item
// check returns when the item violates the rule.
func One(finding Finding) []Finding {
	return []Finding{finding}
}

// When returns a single finding when condition holds, and no finding otherwise.
// The finding is built lazily so that it is only computed for violations.
func When(condition bool, finding func() Finding) []Finding {
	if !condition {
		return nil
	}

	return One(finding())
}

// OnDocument adapts a check over the whole document. It exists so that every
// rule reads the same way: a Spec's Check is always built by a combinator.
func OnDocument(check func(call *Call) []Finding) Check {
	return check
}

// PerOperation runs check for every operation of the document and returns all
// findings.
func PerOperation(check func(call *Call, operation oas.Operation) []Finding) Check {
	return func(call *Call) []Finding {
		return visiting(call, "operation", oas.Operations(call.Root), func(operation oas.Operation) []Finding {
			return check(call, operation)
		})
	}
}

// PerPathItem runs check for every path item of the document and returns all
// findings.
func PerPathItem(check func(call *Call, item oas.PathItem) []Finding) Check {
	return func(call *Call) []Finding {
		return visiting(call, "path item", oas.PathItems(call.Root), func(item oas.PathItem) []Finding {
			return check(call, item)
		})
	}
}

// PerSchema runs check for every schema of the document and returns all
// findings.
func PerSchema(check func(call *Call, schema oas.Schema) []Finding) Check {
	return func(call *Call) []Finding {
		return visiting(call, "schema", oas.Schemas(call.Root), func(schema oas.Schema) []Finding {
			return check(call, schema)
		})
	}
}

// PerParameter runs check for every parameter of every operation, including the
// parameters inherited from the path item.
func PerParameter(check func(call *Call, operation oas.Operation, parameter oas.Parameter) []Finding) Check {
	return PerOperation(func(call *Call, operation oas.Operation) []Finding {
		return seq.FlatMapSlice(oas.Parameters(call.Root, operation), func(parameter oas.Parameter) []Finding {
			return check(call, operation, parameter)
		})
	})
}

// PerResponse runs check for every response of every operation.
func PerResponse(check func(call *Call, operation oas.Operation, response oas.Response) []Finding) Check {
	return PerOperation(func(call *Call, operation oas.Operation) []Finding {
		return seq.FlatMapSlice(oas.Responses(call.Root, operation), func(response oas.Response) []Finding {
			return check(call, operation, response)
		})
	})
}

// Concat combines several checks into one that returns all of their findings.
func Concat(checks ...Check) Check {
	return func(call *Call) []Finding {
		return slices.Concat(seq.Map(slices.Values(checks), func(check Check) []Finding {
			return check(call)
		})...)
	}
}
