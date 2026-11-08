package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// PathSegmentCase returns the extra-path-segment-case rule.
//
// # What it checks
//
// The shape of every path template is checked:
//
//   - the path must start with a slash;
//   - the path must not end with a slash (the root path / is allowed);
//   - the path must not contain empty segments (a double slash);
//   - every literal segment must follow the configured case convention.
//
// A segment that contains a path parameter in braces is skipped, so
// /pets/{petId} only checks pets.
//
// Consistent URL casing makes an API predictable, and a trailing slash or
// empty segment makes two URLs address the same resource in the eyes of some
// servers and not of others.
//
// # Where it comes from
//
// openapilint and apilint, which check that paths start with a slash, have no
// trailing slash and no empty segments, and that segments use one casing.
//
// # Options
//
//   - casing (string or list, default "kebab-case"): kebab-case, snake_case,
//     camelCase, PascalCase, SCREAMING_SNAKE_CASE, SCREAMING-KEBAB-CASE, dot.case or
//     lowercase. A list accepts a segment that follows any of them, an empty
//     list skips the segment check (the shape checks remain) and an unknown
//     value is reported.
//
// # Bad
//
//	paths:
//	  /petOwners/:
//	    get:
//	      responses: {}
//
// # Good
//
//	paths:
//	  /pet-owners:
//	    get:
//	      responses: {}
//
// # Known false positives
//
// Segments with a file extension such as openapi.json contain a dot and do not
// match any case convention, so they are reported. Version segments such as v1
// and v2.1 are only accepted when they match the chosen convention.
func PathSegmentCase() Rule {
	return New(Spec{
		Name:        "path-segment-case",
		Description: "Paths must start with a slash, have no trailing slash or empty segments and use one casing.",
		HowToFix:    "Fix the shape of the path and rename the segments to the configured case.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			casingOption("Case convention of the literal path segments", casing.Kebab),
		},
		Check: WithCasing(checkPathSegmentCase),
	})
}

func checkPathSegmentCase(call *Call, allowed casing.Set) []Finding {
	return PerPathItem(func(_ *Call, item oas.PathItem) []Finding {
		return seq.FlatMapSlice(pathSegmentCaseProblems(item.Route, allowed), func(problem string) []Finding {
			return One(Finding{Node: item.KeyNode, Path: item.Path, Message: problem})
		})
	})(call)
}

func pathSegmentCaseProblems(route string, allowed casing.Set) []string {
	segments := strings.Split(strings.Trim(route, "/"), "/")

	return slices.Concat(
		pathSegmentCaseWhen(!strings.HasPrefix(route, "/"), "path "+route+" must start with a slash"),
		pathSegmentCaseWhen(route != "/" && strings.HasSuffix(route, "/"), "path "+route+" must not end with a slash"),
		pathSegmentCaseWhen(
			route != "/" && strings.Contains(route, "//"),
			"path "+route+" must not contain empty segments",
		),
		seq.FlatMapSlice(segments, func(segment string) []string {
			valid := segment == "" || strings.Contains(segment, "{") || allowed.Matches(segment)

			return pathSegmentCaseWhen(!valid, fmt.Sprintf(
				"path segment %q of %s must be %s", segment, route, allowed,
			))
		}),
	)
}

// pathSegmentCaseWhen returns the problem when condition holds, and nothing
// otherwise.
func pathSegmentCaseWhen(condition bool, problem string) []string {
	if !condition {
		return nil
	}

	return []string{problem}
}
