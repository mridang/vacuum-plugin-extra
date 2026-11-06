package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// RequiredParametersBeforeOptional returns the extra-required-parameters-before-optional rule.
//
// # What it checks
//
// In the parameters list of an operation, every required parameter must be
// listed before every optional one. Each required parameter that follows an
// optional parameter is reported on its own entry. $ref entries are resolved to
// decide whether they are required. Only the parameters declared on the
// operation itself are compared, not those inherited from the path item.
//
// # Why
//
// Code generators that turn parameters into positional function arguments
// cannot place a required argument after an optional one in most languages, so
// they reorder the parameters and break the signature whenever the order
// changes. A stable order of required first avoids that.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (ibm-parameter-order).
//
// # Options
//
// None.
//
// # Bad
//
//	parameters:
//	  - {name: limit, in: query, schema: {type: integer}}
//	  - {name: id, in: path, required: true, schema: {type: string}}
//
// # Good
//
//	parameters:
//	  - {name: id, in: path, required: true, schema: {type: string}}
//	  - {name: limit, in: query, schema: {type: integer}}
//
// # Known false positives
//
// None known. Path parameters are required by definition but are only treated
// as required when they say so with required: true.
func RequiredParametersBeforeOptional() Rule {
	return New(Spec{
		Name:        "required-parameters-before-optional",
		Description: "Required parameters must be listed before optional parameters.",
		HowToFix:    "Move the required parameters to the start of the parameters list.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkRequiredParametersBeforeOptional),
	})
}

// checkRequiredParametersBeforeOptional reports required parameters that follow optional ones.
func checkRequiredParametersBeforeOptional(call *Call, operation oas.Operation) []Finding {
	parameters := oas.OwnParameters(call.Root, operation)

	return seq.FlatMapIndexed(parameters, func(i int, parameter oas.Parameter) []Finding {
		optionalBefore := slices.ContainsFunc(parameters[:i], func(earlier oas.Parameter) bool {
			return !earlier.Required()
		})

		return When(parameter.Required() && optionalBefore, func() Finding {
			return Finding{
				Path:    parameter.Path,
				Node:    parameter.Entry,
				Message: "required parameter '" + parameter.Name() + "' must be listed before optional parameters",
			}
		})
	})
}
