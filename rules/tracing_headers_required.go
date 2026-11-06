package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	// tracingHeadersRequiredStrict is the option that also demands the headers of operations without parameters.
	tracingHeadersRequiredStrict = "requireOnAllOperations"
	// tracingHeadersRequiredTraceID is the lower-case name of the B3 trace header.
	tracingHeadersRequiredTraceID = "x-b3-traceid"
	// tracingHeadersRequiredSpanID is the lower-case name of the B3 span header.
	tracingHeadersRequiredSpanID = "x-b3-spanid"
	// tracingHeadersRequiredTraceparent is the lower-case name of the W3C header.
	tracingHeadersRequiredTraceparent = "traceparent"
	// tracingHeadersRequiredB3Count is the number of B3 headers that must be present.
	tracingHeadersRequiredB3Count = 2
)

// TracingHeadersRequired returns the extra-tracing-headers-required rule.
//
// # What it checks
//
// Every path item must provide a distributed tracing context as header
// parameters. The check follows the original rule path item by path item:
//
//   - If the parameters declared on the path item itself are valid, the path
//     item passes, whatever its operations declare.
//   - Otherwise each operation that declares parameters is examined on its own
//     parameters (they are not merged with those of the path item), and the
//     path item is reported, once, when any of them is not valid.
//
// A list of parameters is valid when it either contains exactly the two B3
// headers X-B3-TraceId and X-B3-SpanId (names compared case-insensitively),
// all with in: header, or contains exactly one traceparent header (W3C Trace
// Context) with in: header. Parameters given as a $ref are resolved first.
//
// As in the original, an operation that declares no parameters at all is not
// examined, so a path item whose operations have no parameters passes. Set
// requireOnAllOperations to treat such an operation as invalid too.
//
// Requests that carry a trace context can be followed across services, which is
// what makes incidents debuggable in a system with many hops.
//
// # Where it comes from
//
// The baloise-incubator Spectral ruleset, rule must-use-tracing (MUST request
// must provide tracing, Zalando guideline 233 adapted), implemented by the
// validate-tracing function with given $.paths.* and error severity.
//
// # Options
//
//   - requireOnAllOperations (bool, default false): also report a path item
//     when one of its operations declares no parameters and the path item has no
//     valid tracing parameters of its own.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: limit
//	          in: query
//	          schema:
//	            type: integer
//	      responses: {}
//
// # Good
//
//	paths:
//	  /pets:
//	    parameters:
//	      - name: traceparent
//	        in: header
//	        schema:
//	          type: string
//	    get:
//	      responses: {}
//
// # Known false positives / limitations
//
// Parameters inherited from the path item are not combined with those of an
// operation, so B3 headers split over both levels are reported, exactly as in
// the original. APIs whose gateway adds the tracing headers and which therefore
// do not list them in the contract are reported; disable the rule there.
func TracingHeadersRequired() Rule {
	return New(Spec{
		Name:        "tracing-headers-required",
		Description: "Requests must provide the B3 or W3C tracing headers.",
		HowToFix:    "Add the X-B3-TraceId and X-B3-SpanId header parameters, or the traceparent header parameter.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        tracingHeadersRequiredStrict,
				Description: "Also require the headers for operations without parameters.",
				Default:     false,
			},
		},
		Check: PerPathItem(checkTracingHeadersRequired),
	})
}

// checkTracingHeadersRequired reports path items without valid tracing parameters.
func checkTracingHeadersRequired(call *Call, item oas.PathItem) []Finding {
	strict := call.Bool(tracingHeadersRequiredStrict)
	invalid := !tracingHeadersRequiredValid(call.Root, oas.Get(item.Node, "parameters")) &&
		tracingHeadersRequiredOperationInvalid(call.Root, item.Node, strict)

	return When(invalid, func() Finding {
		return Finding{
			Node:    item.KeyNode,
			Path:    item.Path,
			Message: "Header X-B3-Traceid, X-B3-Spanid or traceparent (w3c) missing",
		}
	})
}

// tracingHeadersRequiredOperationInvalid reports whether an operation of the
// path item has parameters that are not valid tracing parameters.
func tracingHeadersRequiredOperationInvalid(root, item *yaml.Node, strict bool) bool {
	lists := seq.FlatMapPairs(oas.Pairs(item), func(key, operation *yaml.Node) []*yaml.Node {
		list := oas.Get(operation, "parameters")
		if !oas.IsMethod(key.Value) || (list == nil && !strict) {
			return nil
		}

		return []*yaml.Node{list}
	})

	return slices.ContainsFunc(lists, func(list *yaml.Node) bool {
		return !tracingHeadersRequiredValid(root, list)
	})
}

// tracingHeadersRequiredValid reports whether the parameter list holds the B3
// pair or exactly one traceparent header.
func tracingHeadersRequiredValid(root, list *yaml.Node) bool {
	parameters := seq.FlatMapPairs(oas.Items(list), func(_ int, entry *yaml.Node) []*yaml.Node {
		return []*yaml.Node{oas.Resolve(root, entry)}
	})
	named := func(names ...string) []*yaml.Node {
		return seq.Filter(slices.Values(parameters), func(parameter *yaml.Node) bool {
			return slices.Contains(names, strings.ToLower(oas.StringAt(parameter, "name")))
		})
	}

	return tracingHeadersRequiredAllHeaders(
		named(tracingHeadersRequiredTraceID, tracingHeadersRequiredSpanID),
		tracingHeadersRequiredB3Count,
	) || tracingHeadersRequiredAllHeaders(named(tracingHeadersRequiredTraceparent), 1)
}

// tracingHeadersRequiredAllHeaders reports whether there are exactly want
// parameters and every one is a header.
func tracingHeadersRequiredAllHeaders(parameters []*yaml.Node, want int) bool {
	return len(parameters) == want && !slices.ContainsFunc(parameters, func(parameter *yaml.Node) bool {
		return oas.StringAt(parameter, "in") != keywordHeader
	})
}
