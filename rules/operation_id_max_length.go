package rules

import (
	"fmt"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// operationIDMaxLengthDefault is the default longest accepted operationId.
const operationIDMaxLengthDefault = 64

// OperationIDMaxLength returns the extra-operation-id-max-length rule.
//
// # What it checks
//
// The operationId of every operation must not be longer than a limit, counted
// in characters. Operations without an operationId are ignored; use
// operation-operationId to require one.
//
// The operationId becomes a method or function name in generated clients, a
// tool name for agents and a file name in some tooling. Very long identifiers
// are truncated or rejected by many of them (several LLM tool-calling APIs cap
// names at 64 characters) and are unpleasant to read.
//
// # Where it comes from
//
// ai-api-lint and rate-my-openapi, which flag operationIds longer than 64
// characters.
//
// # Options
//
//   - maxLength (int, default 64): the longest accepted operationId. A value
//     of zero or less disables the rule.
//
// # Bad
//
// With maxLength set to 10:
//
//	paths:
//	  /pets:
//	    get:
//	      operationId: listAllThePetsInTheStore
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      operationId: listPets
//
// # Known false positives
//
// None known. The length is measured in characters, not bytes.
func OperationIDMaxLength() Rule {
	return New(Spec{
		Name:        "operation-id-max-length",
		Description: "The operationId must not be longer than the configured limit.",
		HowToFix:    "Shorten the operationId.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        keywordMaxLength,
				Description: "The longest accepted operationId, in characters.",
				Default:     operationIDMaxLengthDefault,
			},
		},
		Check: PerOperation(checkOperationIDMaxLength),
	})
}

func checkOperationIDMaxLength(call *Call, operation oas.Operation) []Finding {
	return operationIDMaxLengthFindings(call, operation, "operationId", "operationId")
}

// operationIDMaxLengthFindings reports the operation when its string field is
// longer than the maxLength option. The subject names the field in messages.
func operationIDMaxLengthFindings(call *Call, operation oas.Operation, field string, subject string) []Finding {
	limit := call.Int("maxLength")
	key, node := oas.Entry(operation.Node, field)
	length := utf8.RuneCountInString(oas.Text(node))

	return When(limit > 0 && oas.IsScalar(node) && length > limit, func() Finding {
		return Finding{
			Node:    node,
			Path:    oas.Join(operation.Path, key.Value),
			Message: fmt.Sprintf("%s is %d characters long, the limit is %d", subject, length, limit),
		}
	})
}
