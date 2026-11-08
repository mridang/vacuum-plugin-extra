package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// PathParameterCase returns the extra-path-parameter-case rule.
//
// # What it checks
//
// Every variable of a URI template in a path key, such as petId in
// /pets/{petId}, must follow the configured case style. The shared casing
// option selects the convention, snake_case by default.
//
// # Why
//
// Path variables appear in generated client signatures and in documentation.
// A single case style across the API makes URIs predictable and avoids mixed
// spellings such as /pets/{pet_id} next to /owners/{ownerId}.
//
// # Where it comes from
//
// apilint (path parameter case) and the Oxide OpenAPI lint (path parameters
// must be snake case). The two sources are merged into a configurable rule.
//
// # Options
//
//   - casing (string or list, default "snake_case"): kebab-case, snake_case,
//     camelCase, PascalCase, SCREAMING_SNAKE_CASE, SCREAMING-KEBAB-CASE, dot.case or
//     lowercase. A list accepts a variable that follows any of them, an empty
//     list disables the rule and an unknown value is reported.
//
// # Bad
//
//	paths:
//	  /pets/{petId}:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//
// # Good
//
//	paths:
//	  /pets/{pet_id}:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//
// # Known false positives / limitations
//
// Only the template variables in the path key are checked; the name field of
// the matching path parameter is not. Digits are allowed after the first
// letter.
func PathParameterCase() Rule {
	return New(Spec{
		Name:        "path-parameter-case",
		Description: "Path template variables must use the configured case style.",
		HowToFix:    "Rename the path variable to the configured case style and update the parameter definition.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			casingOption("Case style of the path variables", casing.Snake),
		},
		Check: WithCasing(checkPathParameterCase),
	})
}

// checkPathParameterCase reports template variables in the wrong case.
func checkPathParameterCase(call *Call, allowed casing.Set) []Finding {
	return PerPathItem(func(_ *Call, item oas.PathItem) []Finding {
		return seq.FlatMapSlice(pathParameterCaseVariables(item.Route), func(variable string) []Finding {
			return When(!allowed.Matches(variable), func() Finding {
				return Finding{
					Node:    item.KeyNode,
					Path:    item.Path,
					Message: "path parameter " + variable + " must be " + allowed.String(),
				}
			})
		})
	})(call)
}

// pathParameterCaseVariables returns the names inside {} in a path template.
func pathParameterCaseVariables(route string) []string {
	_, afterOpen, found := strings.Cut(route, "{")
	if !found {
		return nil
	}

	name, afterClose, closed := strings.Cut(afterOpen, "}")
	if !closed {
		return nil
	}

	return slices.Concat([]string{name}, pathParameterCaseVariables(afterClose))
}
