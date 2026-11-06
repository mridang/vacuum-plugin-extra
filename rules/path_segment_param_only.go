package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// PathSegmentParamOnly returns the extra-path-segment-param-only rule.
//
// # What it checks
//
// A path segment that contains a parameter reference must consist of that
// reference and nothing else. /pets/{petId} is fine; /pets/pet_{petId},
// /files/{name}.json and /a/{x}{y} are reported, once per offending segment,
// on the path item. A segment with an unbalanced brace is reported too.
//
// # Why
//
// Mixing literals and parameters in one segment makes the template hard to
// match in routers, proxies and gateways and complicates parameter
// serialisation. Put the variable part in its own segment or into a query
// parameter.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (ibm-path-segment-param-only).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /files/{name}.json:
//	    get: {}
//
// # Good
//
//	paths:
//	  /files/{name}:
//	    get: {}
//
// # Known false positives
//
// File-extension style routes such as /reports/{id}.pdf are flagged although
// some APIs use them for content negotiation; prefer the Accept header.
func PathSegmentParamOnly() Rule {
	return New(Spec{
		Name:        "path-segment-param-only",
		Description: "A path segment with a parameter must contain only that parameter.",
		HowToFix:    "Move the literal text into its own path segment so the segment is only a parameter reference.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerPathItem(checkPathSegmentParamOnly),
	})
}

// checkPathSegmentParamOnly reports mixed segments of all paths.
func checkPathSegmentParamOnly(_ *Call, item oas.PathItem) []Finding {
	return seq.FlatMap(strings.SplitSeq(item.Route, "/"), func(segment string) []Finding {
		return When(pathSegmentParamOnlyMixed(segment), func() Finding {
			return Finding{
				Node:    item.KeyNode,
				Path:    item.Path,
				Message: "path segment '" + segment + "' must contain only a parameter reference",
			}
		})
	})
}

func pathSegmentParamOnlyMixed(segment string) bool {
	if !strings.ContainsAny(segment, "{}") {
		return false
	}

	inner, ok := strings.CutPrefix(segment, "{")
	if !ok {
		return true
	}

	rest, closed := strings.CutSuffix(inner, "}")

	return !closed || rest == "" || strings.ContainsAny(rest, "{}")
}
