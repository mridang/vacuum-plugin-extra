package rules

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// DocumentHasOperations returns the extra-document-has-operations rule.
//
// # What it checks
//
// The document must contain at least one operation: a method object (get, put,
// post, delete, options, head, patch or trace) below an entry of paths, or below
// an entry of webhooks in OpenAPI 3.1. The finding is reported on the paths
// object, or on the document root when paths is missing.
//
// # Why
//
// A document without operations describes nothing a client can call. It is
// usually the result of a broken merge, a wrong file being linted or a
// generator that failed silently, and every other rule passes on it vacuously.
//
// # Where it comes from
//
// oas-kit (the walker counts operations and fails when there are none).
//
// # Options
//
// None.
//
// # Bad
//
//	openapi: 3.0.3
//	info: {title: Empty, version: "1"}
//	paths: {}
//
// # Good
//
//	openapi: 3.0.3
//	info: {title: Pets, version: "1"}
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//
// # Known false positives
//
// Documents that only publish components (shared schema libraries) have no
// operations by design. Disable the rule for such files.
func DocumentHasOperations() Rule {
	return New(Spec{
		Name:        "document-has-operations",
		Description: "The document must define at least one operation.",
		HowToFix:    "Add an operation to paths, or lint the correct file.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkDocumentHasOperations),
	})
}

// checkDocumentHasOperations reports a document that declares no operation.
func checkDocumentHasOperations(call *Call) []Finding {
	if documentHasOperationsAny(call.Root) {
		return nil
	}

	message := "the document does not define any operation"

	paths := oas.Get(call.Root, "paths")
	if paths == nil {
		return One(Finding{Message: message})
	}

	return One(Finding{Node: paths, Path: "$.paths", Message: message})
}

func documentHasOperationsAny(root *yaml.Node) bool {
	for range oas.Operations(root) {
		return true
	}

	for _, hook := range oas.Pairs(oas.Get(root, "webhooks")) {
		for method := range oas.Pairs(oas.Resolve(root, hook)) {
			if oas.IsMethod(method.Value) {
				return true
			}
		}
	}

	return false
}
