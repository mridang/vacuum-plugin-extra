package rules

import (
	"fmt"
	"slices"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// apiMaxOperationsDefault is the default limit on the number of operations.
const apiMaxOperationsDefault = 40

// APIMaxOperations returns the extra-api-max-operations rule.
//
// # What it checks
//
// The total number of operations in the document, counted as the entries under
// paths that use one of the eight HTTP method keys (get, put, post, delete,
// options, head, patch and trace), must not exceed a limit. When the limit is
// exceeded a single finding is reported on the paths object.
//
// A very large API surface is hard to document, review, version and consume;
// it is also a poor fit for tool-calling clients that must load every
// operation. A document that grows past the limit is a hint that it should be
// split into several smaller, focused APIs.
//
// # Where it comes from
//
// The ai-api-lint rule api-max-operations, which flags documents with more than
// 40 operations.
//
// # Options
//
//   - maxOperations (int, default 40): the largest number of operations that is
//     still accepted. A value of zero or less disables the rule.
//
// # Bad
//
// With maxOperations set to 1:
//
//	paths:
//	  /pets:
//	    get:
//	      responses: {}
//	    post:
//	      responses: {}
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      responses: {}
//
// # Known false positives
//
// Operations that are only reachable through a $ref to an external path item
// are not counted, and path items reached through a local $ref are counted once
// per referencing path. The limit is global, so a large but cohesive API has to
// raise the option or disable the rule.
func APIMaxOperations() Rule {
	return New(Spec{
		Name:        "api-max-operations",
		Description: "The document must not define more operations than the configured limit.",
		HowToFix:    "Split the API into several smaller documents or raise the maxOperations option.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        "maxOperations",
				Description: "The largest number of operations that is accepted.",
				Default:     apiMaxOperationsDefault,
			},
		},
		Check: OnDocument(checkAPIMaxOperations),
	})
}

func checkAPIMaxOperations(call *Call) []Finding {
	limit := call.Int("maxOperations")
	if limit <= 0 {
		return nil
	}

	total := len(slices.Collect(oas.Operations(call.Root)))

	return When(total > limit, func() Finding {
		return Finding{
			Node: oas.Get(call.Root, "paths"),
			Path: "$.paths",
			Message: fmt.Sprintf(
				"the document defines %d operations, which is more than the limit of %d",
				total,
				limit,
			),
		}
	})
}
