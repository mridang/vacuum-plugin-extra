package rules

import (
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

const multiplePathParamsMaxOption = "max"

// MultiplePathParams returns the extra-multiple-path-params rule.
//
// # What it checks
//
// A path template must not contain more than the allowed number of path
// parameters (one by default). The parameters are counted by the number of {name}
// expressions in the template, so /orgs/{org}/repos/{repo} has two. One finding
// is reported per offending path, on the path item.
//
// # Why
//
// A path with several identifiers usually means that a nested resource is
// addressed through its parents even though its own identifier is globally
// unique. Such paths are long, couple resources to their parents and are hard
// to evolve; a root path such as /repos/{repo} is often enough.
//
// # Where it comes from
//
// Zally (avoid multiple path parameters, suggesting a nested resource could be
// a root path).
//
// # Options
//
//   - max (integer, default 1): the largest number of path parameters a path
//     may contain.
//
// # Bad
//
//	paths:
//	  /orgs/{org}/repos/{repo}:
//	    get: {}
//
// # Good
//
//	paths:
//	  /repos/{repo}:
//	    get: {}
//
// # Known false positives
//
// Resources whose identifiers are only unique within their parent, such as
// /orders/{order}/lines/{line}, legitimately need several parameters. Raise max
// for such APIs.
func MultiplePathParams() Rule {
	return New(Spec{
		Name:        "multiple-path-params",
		Description: "Paths should not contain more than one path parameter.",
		HowToFix:    "Address the nested resource through its own root path, or raise the max option.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        multiplePathParamsMaxOption,
			Description: "Largest number of path parameters allowed in one path.",
			Default:     1,
		}},
		Check: PerPathItem(checkMultiplePathParams),
	})
}

// checkMultiplePathParams reports paths with too many parameters.
func checkMultiplePathParams(call *Call, item oas.PathItem) []Finding {
	limit := call.Int(multiplePathParamsMaxOption)
	count := strings.Count(item.Route, "{")

	return When(count > limit, func() Finding {
		return Finding{
			Node: item.KeyNode,
			Path: item.Path,
			Message: "path has " + strconv.Itoa(
				count,
			) + " path parameters but at most " + strconv.Itoa(
				limit,
			) + " are allowed",
		}
	})
}
