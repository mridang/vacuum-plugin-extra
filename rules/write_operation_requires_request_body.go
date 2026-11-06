package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const writeOperationRequiresRequestBodyMethodsOption = "methods"

// WriteOperationRequiresRequestBody returns the extra-write-operation-requires-request-body rule.
//
// # What it checks
//
// Operations with a write method must declare a requestBody. Only the absence of
// the requestBody key is reported; an empty requestBody object is accepted, as
// it is present. The finding is reported on the operation.
//
// # Why
//
// A POST, PUT or PATCH without a request body describes a write that carries no
// data, which is almost always an omission. Clients and generated SDKs cannot
// tell what to send.
//
// # Where it comes from
//
// ai-api-lint (operations with post, put or patch must declare a request body).
//
// # Options
//
//   - methods (list of strings, default post, put, patch): the HTTP methods that
//     need a request body.
//
// # Bad
//
//	paths:
//	  /pets:
//	    post:
//	      responses:
//	        "201":
//	          description: created
//
// # Good
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        content:
//	          application/json:
//	            schema: {type: object}
//	      responses:
//	        "201":
//	          description: created
//
// # Known false positives
//
// Action endpoints that trigger work without input, such as
// POST /orders/{id}/cancel, are flagged. Remove post from the methods option, or
// suppress the rule for such operations.
func WriteOperationRequiresRequestBody() Rule {
	return New(Spec{
		Name:        "write-operation-requires-request-body",
		Description: "POST, PUT and PATCH operations must declare a request body.",
		HowToFix:    "Add a requestBody to the operation.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{{
			Name:        writeOperationRequiresRequestBodyMethodsOption,
			Description: "HTTP methods that need a request body.",
			Default:     []string{literalPost, literalPut, literalPatch},
		}},
		Check: PerOperation(checkWriteOperationRequiresRequestBody),
	})
}

// checkWriteOperationRequiresRequestBody reports write operations without a body.
func checkWriteOperationRequiresRequestBody(call *Call, operation oas.Operation) []Finding {
	methods := seq.Map(slices.Values(call.Strings(writeOperationRequiresRequestBodyMethodsOption)), Fold)
	missing := slices.Contains(methods, operation.Method) && !oas.Has(operation.Node, "requestBody")

	return When(missing, func() Finding {
		return Finding{
			Node:    operation.Node,
			Path:    operation.Path,
			Message: operation.Method + " operation must declare a requestBody",
		}
	})
}
