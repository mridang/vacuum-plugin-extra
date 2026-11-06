package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// PathItemSummaryOrDescription returns the extra-path-item-summary-or-description rule.
//
// # What it checks
//
// Every path item must have a non-blank summary or description of its own. A
// path item given by $ref is judged by the item it points at, and paths whose
// key starts with x- are skipped.
//
// # Why
//
// The summary and description of a path item are inherited by all its
// operations in documentation tools. Describing the resource once, at path
// level, explains what a route is for independently of the individual verbs.
//
// # Where it comes from
//
// oas-kit (path-item-summary-or-description, shipped disabled by default there
// because many documents only describe operations). It is a candidate for
// organisations that want resource level documentation.
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
//	      responses:
//	        "200":
//	          description: ok
//
// # Good
//
//	paths:
//	  /pets:
//	    summary: The pets of the store
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//
// # Known false positives and limitations
//
// The summary and description of the operations are not considered, so a path
// whose operations are all documented is still reported. Path-level summary and
// description are only available in OpenAPI 3; in Swagger 2 documents the rule
// reports every path item.
func PathItemSummaryOrDescription() Rule {
	return New(Spec{
		Name:        "path-item-summary-or-description",
		Description: "Each path item must have a summary or a description.",
		HowToFix:    "Add a summary or a description to the path item.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerPathItem(checkPathItemSummaryOrDescription),
	})
}

// checkPathItemSummaryOrDescription reports path items with neither summary nor description.
func checkPathItemSummaryOrDescription(_ *Call, item oas.PathItem) []Finding {
	return When(
		strings.TrimSpace(oas.StringAt(item.Node, "summary")) == "" &&
			strings.TrimSpace(oas.StringAt(item.Node, "description")) == "",
		func() Finding {
			return Finding{
				Node:    item.KeyNode,
				Path:    item.Path,
				Message: "path item " + item.Route + " must have a summary or a description",
			}
		},
	)
}
