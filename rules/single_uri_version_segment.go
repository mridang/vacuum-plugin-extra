package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// singleURIVersionSegmentLimit is the number of version segments a path may hold.
const singleURIVersionSegmentLimit = 1

// SingleURIVersionSegment returns the extra-single-uri-version-segment rule.
//
// # What it checks
//
// A path key must not contain more than one version segment. A version segment
// is a whole path segment made of v (or V) and a number, optionally with minor
// parts, such as /v1, /v2 or /v1.2. A single version segment is allowed.
//
// # Why
//
// The API version belongs to the URI once. Two version segments, as in
// /v1/orders/v2/items, mean that the version of the whole API and the version
// of a sub-resource are mixed, which makes clients hard to evolve and routing
// ambiguous.
//
// # Where it comes from
//
// The Baloise Spectral ruleset (baloise-incubator/Spectral-ruleset,
// single-uri-version-segment).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /v1/orders/v2/items:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//
// # Good
//
//	paths:
//	  /v1/orders/items:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//
// # Known false positives / limitations
//
// A resource that is legitimately named like a version (for example a /v2
// product code) is counted. Server URLs are not inspected, so a version in a
// server URL plus one in the path is not reported.
func SingleURIVersionSegment() Rule {
	return New(Spec{
		Name:        "single-uri-version-segment",
		Description: "Paths must not contain more than one version segment.",
		HowToFix:    "Keep a single version segment in the path.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkSingleURIVersionSegment),
	})
}

// singleURIVersionSegmentExpression matches a version path segment.
func singleURIVersionSegmentExpression() *regexp.Regexp {
	return regexp.MustCompile(`^[vV][0-9]+(\.[0-9]+)*$`)
}

// checkSingleURIVersionSegment reports paths with several version segments.
func checkSingleURIVersionSegment(call *Call) []Finding {
	expression := singleURIVersionSegmentExpression()

	return PerPathItem(func(_ *Call, item oas.PathItem) []Finding {
		segments := seq.Filter(slices.Values(strings.Split(item.Route, "/")), expression.MatchString)

		return When(len(segments) > singleURIVersionSegmentLimit, func() Finding {
			return Finding{
				Node:    item.KeyNode,
				Path:    item.Path,
				Message: "path must not contain more than one version segment",
			}
		})
	})(call)
}
