package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// PathParamPlacement returns the extra-path-param-placement rule.
//
// # What it checks
//
// Two placement rules for path parameters in the keys of the paths object. A
// path must not start with a path parameter (/{id}/pets), and a segment that
// holds a path parameter must consist of the parameter only: text before or
// after the braces (/pets/id-{id}, /files/{name}.json) is reported.
//
// # Why
//
// A path that starts with a parameter cannot be told apart from any other first
// segment and makes routing and gateway rules ambiguous. A parameter mixed with
// literal text in one segment is hard to document, is not supported by every
// router and breaks tools that treat each segment as one value.
//
// # Where it comes from
//
// Zally path guidelines (the 129 kebab-case path segment rule and 143 resource
// identifier rule family), which do not restrict where a parameter sits. There
// is no upstream implementation; the check is the structural guard that the
// pattern (^/\{)|[^/]\{|\}[^/] expresses: a route must not match it. This rule
// reproduces that pattern on segments, so it reports exactly the routes that
// the pattern matches, and it names the problem.
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /{tenant}/pets: {}
//	  /pets/id-{id}: {}
//
// # Good
//
//	paths:
//	  /tenants/{tenant}/pets: {}
//	  /pets/{id}: {}
//
// # Known false positives / limitations
//
// APIs that deliberately use a file extension on a parameter (/reports/{id}.csv)
// are reported; suppress the rule for those paths. A path with a base path in
// the server URL is judged by its own text only.
func PathParamPlacement() Rule {
	return New(Spec{
		Name:        "path-param-placement",
		Description: "Paths must not start with a path parameter, and path parameters must fill a whole segment.",
		HowToFix:    "Start the path with a literal segment and keep each path parameter in a segment of its own.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerPathItem(checkPathParamPlacement),
	})
}

// checkPathParamPlacement reports a route whose parameters are misplaced.
func checkPathParamPlacement(_ *Call, item oas.PathItem) []Finding {
	return seq.Map(slices.Values(pathParamPlacementProblems(item.Route)), func(message string) Finding {
		return Finding{
			Node:    item.KeyNode,
			Path:    item.Path,
			Message: message,
		}
	})
}

// pathParamPlacementProblems returns a message for every placement problem of
// the route.
func pathParamPlacementProblems(route string) []string {
	segments := strings.Split(strings.TrimPrefix(route, "/"), "/")

	return slices.Concat(
		pathParamPlacementStart(route, segments),
		seq.FlatMapSlice(segments, func(segment string) []string {
			if !strings.Contains(segment, "{") || pathParamPlacementIsParameter(segment) {
				return nil
			}

			return []string{
				"segment '" + segment + "' of path '" + route + "' must not mix a path parameter with other text",
			}
		}),
	)
}

// pathParamPlacementStart returns the message for a route starting with a path parameter.
func pathParamPlacementStart(route string, segments []string) []string {
	if len(segments) == 0 || !pathParamPlacementIsParameter(segments[0]) {
		return nil
	}

	return []string{"path '" + route + "' must not start with a path parameter"}
}

// pathParamPlacementIsParameter reports whether the whole segment is a single
// {name} template.
func pathParamPlacementIsParameter(segment string) bool {
	rest, found := strings.CutPrefix(segment, "{")
	if !found {
		return false
	}

	inner, closed := strings.CutSuffix(rest, "}")

	return closed && inner != "" && !strings.ContainsAny(inner, "{}")
}
