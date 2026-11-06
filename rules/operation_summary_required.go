package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// OperationSummaryRequired returns the extra-operation-summary-required rule.
//
// # What it checks
//
// Every operation must have a summary that is not empty and not just
// whitespace. A missing summary is reported on the operation, an empty one on
// the summary itself.
//
// # Why
//
// The summary is the short label shown in documentation navigation, generated
// client method comments and tool listings. Without it, tools fall back to the
// path or operationId, which are rarely suitable for people.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (ibm-operation-summary), Redocly
// (operation-summary) and rate-my-openapi (operation summary), merged into one
// rule.
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
//	      responses:
//	        "200":
//	          description: ok
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      summary: List pets
//	      responses:
//	        "200":
//	          description: ok
//
// # Known false positives
//
// None known. Operations whose summary is supplied only through a vendor
// extension are still flagged.
func OperationSummaryRequired() Rule {
	return New(Spec{
		Name:        "operation-summary-required",
		Description: "Every operation must have a non-empty summary.",
		HowToFix:    "Add a short summary to the operation.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkOperationSummaryRequired),
	})
}

// checkOperationSummaryRequired reports missing and empty summaries.
func checkOperationSummaryRequired(_ *Call, operation oas.Operation) []Finding {
	key, summary := oas.Entry(operation.Node, "summary")

	switch {
	case summary == nil:
		return One(Finding{
			Node:    operation.Node,
			Path:    operation.Path,
			Message: "operation must have a summary",
		})
	case strings.TrimSpace(oas.Text(summary)) == "":
		return One(Finding{
			Node:    summary,
			Path:    oas.Join(operation.Path, key.Value),
			Message: "operation summary must not be empty",
		})
	}

	return nil
}
