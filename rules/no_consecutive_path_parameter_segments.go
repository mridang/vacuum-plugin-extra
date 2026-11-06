package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// NoConsecutivePathParameterSegments returns the
// extra-no-consecutive-path-parameter-segments rule.
//
// # What it checks
//
// A path template must not contain two adjacent segments that each hold a path
// parameter, such as /pets/{petId}/{ownerId}. A segment holds a parameter when
// it contains a brace pair. Each reported path is flagged once.
//
// Two parameters in a row hide the resource that sits between them. Naming the
// collection (/pets/{petId}/owners/{ownerId}) makes the URL readable and keeps
// the hierarchy of resources explicit.
//
// # Where it comes from
//
// IBM openapi-validator, rule ibm-no-consecutive-path-parameter-segments.
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets/{petId}/{ownerId}:
//	    get:
//	      responses: {}
//
// # Good
//
//	paths:
//	  /pets/{petId}/owners/{ownerId}:
//	    get:
//	      responses: {}
//
// # Known false positives
//
// None known. A deliberately flat template such as /{org}/{repo} is reported
// and has to be suppressed per path.
func NoConsecutivePathParameterSegments() Rule {
	return New(Spec{
		Name:        "no-consecutive-path-parameter-segments",
		Description: "A path must not contain two consecutive path parameter segments.",
		HowToFix:    "Add a resource name segment between the two path parameters.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerPathItem(checkNoConsecutivePathParameterSegments),
	})
}

func checkNoConsecutivePathParameterSegments(_ *Call, item oas.PathItem) []Finding {
	segments := seq.Map(strings.SplitSeq(item.Route, "/"), func(segment string) bool {
		return strings.Contains(segment, "{") && strings.Contains(segment, "}")
	})

	return When(noConsecutivePathParameterSegmentsHasPair(segments), func() Finding {
		return Finding{
			Node:    item.KeyNode,
			Path:    item.Path,
			Message: "path " + item.Route + " has consecutive path parameter segments",
		}
	})
}

func noConsecutivePathParameterSegmentsHasPair(parameters []bool) bool {
	return len(parameters) > 1 &&
		(parameters[0] && parameters[1] || noConsecutivePathParameterSegmentsHasPair(parameters[1:]))
}
