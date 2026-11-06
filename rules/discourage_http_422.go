package rules

import (
	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// DiscourageHTTP422 returns the extra-discourage-http-422 rule.
//
// # What it checks
//
// An operation must not define a 422 (Unprocessable Content) response. Use 400
// (Bad Request) for invalid request payloads instead.
//
// # Why
//
// 422 comes from WebDAV and is not part of the commonly understood set of
// status codes. Using 400 for every client side validation failure keeps error
// handling in clients simple and consistent.
//
// # Where it comes from
//
// The Baloise Spectral ruleset (must-use-additional-standard-http-status-codes,
// the 422 warning). IBM's response-status-code-semantics rule contains the same
// check; it is also part of extra-response-status-code-semantics.
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets:
//	    post:
//	      responses:
//	        "422":
//	          description: invalid
//
// # Good
//
//	paths:
//	  /pets:
//	    post:
//	      responses:
//	        "400":
//	          description: invalid
//
// # Known false positives / limitations
//
// Teams that deliberately separate syntax errors (400) from semantic validation
// errors (422) will see every such response reported; disable the rule for them.
func DiscourageHTTP422() Rule {
	return New(Spec{
		Name:        "discourage-http-422",
		Description: "Operations should use status code 400 instead of 422.",
		HowToFix:    "Replace the 422 response with a 400 response.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkDiscourageHTTP422),
	})
}

func checkDiscourageHTTP422(_ *Call, operation oas.Operation) []Finding {
	key, _ := oas.Entry(oas.Get(operation.Node, "responses"), "422")

	return When(key != nil, func() Finding {
		return Finding{
			Node:    key,
			Path:    oas.JoinAll(operation.Path, "responses", "422"),
			Message: "use status code 400 instead of 422 for invalid request payloads",
		}
	})
}
