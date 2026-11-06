package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// pathParameterCaseSnake is the default style.
const pathParameterCaseSnake = "snake"

// pathParameterCaseStyle is the option that selects the convention.
const pathParameterCaseStyle = "style"

// PathParameterCase returns the extra-path-parameter-case rule.
//
// # What it checks
//
// Every variable of a URI template in a path key, such as petId in
// /pets/{petId}, must follow the configured case style. The style option
// accepts snake (the default), camel, pascal, kebab and screaming-snake.
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
//   - style (string, default "snake"): snake, camel, pascal, kebab or
//     screaming-snake. An empty or unknown style disables the rule.
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
			{
				Name:        pathParameterCaseStyle,
				Description: "Case style: snake, camel, pascal, kebab or screaming-snake.",
				Default:     pathParameterCaseSnake,
			},
		},
		Check: OnDocument(checkPathParameterCase),
	})
}

// pathParameterCaseExpression returns the expression of a style, or nil.
func pathParameterCaseExpression(style string) *regexp.Regexp {
	expressions := map[string]string{
		literalSnake:      `^[a-z][a-z0-9]*(_[a-z0-9]+)*$`,
		"camel":           `^[a-z][a-zA-Z0-9]*$`,
		literalPascal:     `^[A-Z][a-zA-Z0-9]*$`,
		"kebab":           `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`,
		"screaming-snake": `^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$`,
	}

	source, ok := expressions[style]
	if !ok {
		return nil
	}

	return regexp.MustCompile(source)
}

// checkPathParameterCase reports template variables in the wrong case.
func checkPathParameterCase(call *Call) []Finding {
	style := call.String(pathParameterCaseStyle)
	expression := pathParameterCaseExpression(style)
	if expression == nil {
		return nil
	}

	return PerPathItem(func(_ *Call, item oas.PathItem) []Finding {
		return seq.FlatMapSlice(pathParameterCaseVariables(item.Route), func(variable string) []Finding {
			return When(!expression.MatchString(variable), func() Finding {
				return Finding{
					Node:    item.KeyNode,
					Path:    item.Path,
					Message: "path parameter " + variable + " must be " + style + " case",
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
