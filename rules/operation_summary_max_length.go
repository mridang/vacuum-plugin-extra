package rules

import (
	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// operationSummaryMaxLengthDefault is the default longest accepted summary.
const operationSummaryMaxLengthDefault = 80

// OperationSummaryMaxLength returns the extra-operation-summary-max-length rule.
//
// # What it checks
//
// The summary of an operation must be at most a number of characters long
// (80 by default). Operations without a summary are ignored.
//
// Summaries are shown in navigation menus, tables and generated method
// documentation. A short summary stays readable there; long explanations
// belong in the description field.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-operation-summary-length, which limits the
// summary to 80 characters) and speccy (operation-summary-length).
//
// # Options
//
//   - maxLength (int, default 80): the longest accepted summary. A value of
//     zero or less disables the rule.
//
// # Bad
//
// With maxLength set to 10:
//
//	paths:
//	  /pets:
//	    get:
//	      summary: List every pet that is registered in the store
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      summary: List pets
//
// # Known false positives
//
// None known. The length is measured in characters, not bytes, after the YAML
// has been decoded, so folded block scalars are counted with their line
// breaks.
func OperationSummaryMaxLength() Rule {
	return New(Spec{
		Name:        "operation-summary-max-length",
		Description: "The operation summary must not be longer than the configured limit.",
		HowToFix:    "Shorten the summary and move the details to the description.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        keywordMaxLength,
				Description: "The longest accepted summary, in characters.",
				Default:     operationSummaryMaxLengthDefault,
			},
		},
		Check: PerOperation(checkOperationSummaryMaxLength),
	})
}

func checkOperationSummaryMaxLength(call *Call, operation oas.Operation) []Finding {
	return operationIDMaxLengthFindings(call, operation, "summary", "operation summary")
}
