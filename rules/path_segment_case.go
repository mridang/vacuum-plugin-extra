package rules

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// pathSegmentCaseDefault is the case convention applied when none is given.
const pathSegmentCaseDefault = "kebab"

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
//   - case (string, default "kebab"): one of kebab (lower case words joined by
//     hyphens), snake (lower case words joined by underscores), camel
//     (lowerCamelCase), pascal (UpperCamelCase) and lower (lower case letters
//     and digits only). An unknown value falls back to kebab.
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
			{
				Name:        "case",
				Description: "The case convention of literal segments: kebab, snake, camel, pascal or lower.",
				Default:     pathSegmentCaseDefault,
			},
		},
		Check: PerPathItem(checkPathSegmentCase),
	})
}

func pathSegmentCasePattern(name string) string {
	switch name {
	case literalSnake:
		return `^[a-z0-9]+(_[a-z0-9]+)*$`
	case pathSegmentCaseCamel:
		return `^[a-z0-9][a-zA-Z0-9]*$`
	case literalPascal:
		return pathSegmentCasePascalPattern
	case "lower":
		return `^[a-z0-9]+$`
	default:
		return `^[a-z0-9]+(-[a-z0-9]+)*$`
	}
}

func checkPathSegmentCase(call *Call, item oas.PathItem) []Finding {
	convention := call.String("case")
	pattern := regexp.MustCompile(pathSegmentCasePattern(convention))

	return seq.FlatMapSlice(pathSegmentCaseProblems(item.Route, convention, pattern), func(problem string) []Finding {
		return One(Finding{Node: item.KeyNode, Path: item.Path, Message: problem})
	})
}

func pathSegmentCaseProblems(route, convention string, pattern *regexp.Regexp) []string {
	segments := strings.Split(strings.Trim(route, "/"), "/")

	return slices.Concat(
		pathSegmentCaseWhen(!strings.HasPrefix(route, "/"), "path "+route+" must start with a slash"),
		pathSegmentCaseWhen(route != "/" && strings.HasSuffix(route, "/"), "path "+route+" must not end with a slash"),
		pathSegmentCaseWhen(
			route != "/" && strings.Contains(route, "//"),
			"path "+route+" must not contain empty segments",
		),
		seq.FlatMapSlice(segments, func(segment string) []string {
			valid := segment == "" || strings.Contains(segment, "{") || pattern.MatchString(segment)

			return pathSegmentCaseWhen(!valid, fmt.Sprintf(
				"path segment %q of %s must be in %s case", segment, route, pathSegmentCaseName(convention),
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

func pathSegmentCaseName(convention string) string {
	switch convention {
	case "snake", pathSegmentCaseCamel, "pascal", "lower":
		return convention
	default:
		return pathSegmentCaseDefault
	}
}

// Keyword names used by this rule.
const (
	pathSegmentCaseCamel         = "camel"
	pathSegmentCasePascalPattern = `^[A-Z][a-zA-Z0-9]*$`
)
