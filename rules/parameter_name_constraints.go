package rules

import (
	"regexp"
	"strconv"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	// parameterNameConstraintsPattern is the option with the charset expression.
	parameterNameConstraintsPattern = "pattern"
	// parameterNameConstraintsMaxLength is the option with the maximum length.
	parameterNameConstraintsMaxLength = "maxLength"
	// parameterNameConstraintsMinLength is the option with the minimum length.
	parameterNameConstraintsMinLength = "minLength"
	// parameterNameConstraintsMaxLengthDefault is the default maximum length.
	parameterNameConstraintsMaxLengthDefault = 50
	// parameterNameConstraintsMinLengthDefault is the default minimum length.
	parameterNameConstraintsMinLengthDefault = 1
)

// ParameterNameConstraints returns the extra-parameter-name-constraints rule.
//
// # What it checks
//
// The name of every parameter (operation, path item and components.parameters)
// must contain at least minLength characters, at most maxLength characters and,
// when pattern is set, match that regular expression.
//
// # Why
//
// Empty parameter names make a parameter unusable, extremely long names are
// almost always mistakes, and a charset restriction (for example the RFC 6570
// variable name characters) keeps names valid in URI templates and in the
// identifiers of generated clients.
//
// # Where it comes from
//
// Speccy and oas-kit (parameter-name-charset, shipped disabled with an empty
// pattern), and rate-my-openapi (parameter-name-max-length, limited to 50
// characters, and parameter-name-nonempty). The three rows are merged.
//
// # Options
//
//   - pattern (string, default empty): expression every name must match. An
//     empty or invalid expression disables the charset check.
//   - maxLength (int, default 50): longest allowed name. Zero disables it.
//   - minLength (int, default 1): shortest allowed name. Zero disables it.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: ""
//	          in: query
//	          schema:
//	            type: string
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: limit
//	          in: query
//	          schema:
//	            type: integer
//
// # Known false positives / limitations
//
// A parameter without a name key is not reported here; a missing name is a
// structural error for the OpenAPI validation rules. Lengths count characters,
// not bytes. Expressions use the Go regexp syntax (RE2).
func ParameterNameConstraints() Rule {
	return New(Spec{
		Name:        "parameter-name-constraints",
		Description: "Parameter names must be non-empty, short and match the configured pattern.",
		HowToFix:    "Rename the parameter so that it satisfies the length and pattern constraints.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        parameterNameConstraintsPattern,
				Description: "Regular expression every name must match; empty disables the check.",
				Default:     "",
			},
			{
				Name:        parameterNameConstraintsMaxLength,
				Description: "Maximum name length; zero disables the check.",
				Default:     parameterNameConstraintsMaxLengthDefault,
			},
			{
				Name:        parameterNameConstraintsMinLength,
				Description: "Minimum name length; zero disables the check.",
				Default:     parameterNameConstraintsMinLengthDefault,
			},
		},
		Check: OnDocument(checkParameterNameConstraints),
	})
}

// parameterNameConstraintsLimits holds the resolved options.
type parameterNameConstraintsLimits struct {
	pattern   *regexp.Regexp
	maxLength int
	minLength int
}

// checkParameterNameConstraints reports parameter names that break a constraint.
func checkParameterNameConstraints(call *Call) []Finding {
	limits := parameterNameConstraintsLimits{
		pattern:   parameterNameConstraintsCompile(call.String(parameterNameConstraintsPattern)),
		maxLength: call.Int(parameterNameConstraintsMaxLength),
		minLength: call.Int(parameterNameConstraintsMinLength),
	}

	return seq.FlatMapSlice(documentParameters(call.Root), func(parameter oas.Parameter) []Finding {
		key, name := oas.Entry(parameter.Node, "name")
		if !oas.IsScalar(name) {
			return nil
		}

		message := limits.violation(name.Value)

		return When(message != "", func() Finding {
			return Finding{
				Node:    name,
				Path:    oas.Join(parameter.Path, key.Value),
				Message: message,
			}
		})
	})
}

// parameterNameConstraintsCompile compiles the pattern, or returns nil when it is empty or invalid.
func parameterNameConstraintsCompile(source string) *regexp.Regexp {
	if source == "" {
		return nil
	}

	pattern, _ := regexp.Compile(source)

	return pattern
}

// violation returns the first broken constraint of name, or the empty string.
func (l parameterNameConstraintsLimits) violation(name string) string {
	length := utf8.RuneCountInString(name)

	switch {
	case l.minLength > 0 && length < l.minLength:
		return "parameter name must be at least " + strconv.Itoa(l.minLength) + " characters long"
	case l.maxLength > 0 && length > l.maxLength:
		return "parameter name " + name + " must be at most " + strconv.Itoa(l.maxLength) + " characters long"
	case l.pattern != nil && !l.pattern.MatchString(name):
		return "parameter name " + name + " must match " + l.pattern.String()
	default:
		return ""
	}
}
