package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// SummaryNoTrailingPeriod returns the extra-summary-no-trailing-period rule.
//
// # What it checks
//
// The summary of every operation must not end with a period. Summaries are
// short labels shown in navigation menus, tables of contents and generated
// client method documentation; a full stop makes them look like sentences and
// is inconsistent when the labels are listed.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (summary-no-trailing-period) and the Oxide
// OpenAPI lint (operation summaries must not end with a period). The two
// near-duplicate rows of the source list are merged into this one rule.
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      summary: List all pets.
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      summary: List all pets
//
// # Known false positives
//
// A summary that ends in an abbreviation such as "etc." is flagged even though
// the period belongs to the abbreviation. Reword the summary or suppress the
// rule for that operation. Ellipses ("...") are flagged too.
func SummaryNoTrailingPeriod() Rule {
	return New(Spec{
		Name:        "summary-no-trailing-period",
		Description: "Operation summaries must not end with a period.",
		HowToFix:    "Remove the trailing period from the operation summary.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkSummaryNoTrailingPeriod),
	})
}

func checkSummaryNoTrailingPeriod(_ *Call, operation oas.Operation) []Finding {
	key, summary := oas.Entry(operation.Node, "summary")

	return When(
		oas.IsScalar(summary) && strings.HasSuffix(strings.TrimSpace(summary.Value), "."),
		func() Finding {
			return Finding{
				Node:    summary,
				Path:    oas.Join(operation.Path, key.Value),
				Message: "operation summary must not end with a period",
			}
		},
	)
}
