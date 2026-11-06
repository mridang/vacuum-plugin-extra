package rules

import (
	"slices"
	"strconv"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	// operationMaxParametersTotal is the option with the maximum parameter count.
	operationMaxParametersTotal = "maxParameters"
	// operationMaxParametersRequired is the option with the maximum required count.
	operationMaxParametersRequired = "maxRequiredParameters"
	// operationMaxParametersTotalDefault is the default maximum parameter count.
	operationMaxParametersTotalDefault = 15
	// operationMaxParametersRequiredDefault is the default maximum required count.
	operationMaxParametersRequiredDefault = 7
)

// OperationMaxParameters returns the extra-operation-max-parameters rule.
//
// # What it checks
//
// The number of parameters of an operation, counting the parameters inherited
// from its path item (an operation parameter replaces an inherited one with the
// same name and location), must not exceed maxParameters, and the number of
// those parameters marked required: true must not exceed maxRequiredParameters.
//
// # Why
//
// Operations with many parameters are hard to call, document and test, and are
// usually a sign that the endpoint tries to do too much or that a request body
// or a filter object would describe the input better. Many required inputs make
// the operation fragile for clients.
//
// # Where it comes from
//
// ai-api-lint (operation-max-parameters and operation-max-required-parameters).
// The two rows are merged.
//
// # Options
//
//   - maxParameters (int, default 15): the highest allowed parameter count.
//     Zero or a negative value disables this limit.
//   - maxRequiredParameters (int, default 7): the highest allowed number of
//     required parameters. Zero or a negative value disables this limit.
//
// # Bad
//
// An operation with sixteen query parameters, or eight required ones.
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
// Cookie and header parameters count like query parameters, although they are
// often cross-cutting and shared by every operation.
func OperationMaxParameters() Rule {
	return New(Spec{
		Name:        "operation-max-parameters",
		Description: "Operations must not have too many parameters or required parameters.",
		HowToFix:    "Reduce the number of parameters, for example by moving inputs into a request body.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        operationMaxParametersTotal,
				Description: "Maximum number of parameters; zero disables the limit.",
				Default:     operationMaxParametersTotalDefault,
			},
			{
				Name:        operationMaxParametersRequired,
				Description: "Maximum number of required parameters; zero disables the limit.",
				Default:     operationMaxParametersRequiredDefault,
			},
		},
		Check: OnDocument(checkOperationMaxParameters),
	})
}

// checkOperationMaxParameters reports operations that exceed a limit.
func checkOperationMaxParameters(call *Call) []Finding {
	maxTotal := call.Int(operationMaxParametersTotal)
	maxRequired := call.Int(operationMaxParametersRequired)

	return PerOperation(func(call *Call, operation oas.Operation) []Finding {
		parameters := oas.Parameters(call.Root, operation)
		required := len(seq.Filter(slices.Values(parameters), oas.Parameter.Required))

		return slices.Concat(
			When(maxTotal > 0 && len(parameters) > maxTotal, func() Finding {
				return Finding{
					Node: oas.Get(operation.Node, "parameters"),
					Path: operation.Path,
					Message: "operation has " + strconv.Itoa(
						len(parameters),
					) + " parameters, more than the maximum of " + strconv.Itoa(
						maxTotal,
					),
				}
			}),
			When(maxRequired > 0 && required > maxRequired, func() Finding {
				return Finding{
					Node: oas.Get(operation.Node, "parameters"),
					Path: operation.Path,
					Message: "operation has " + strconv.Itoa(
						required,
					) + " required parameters, more than the maximum of " + strconv.Itoa(
						maxRequired,
					),
				}
			}),
		)
	})(call)
}
