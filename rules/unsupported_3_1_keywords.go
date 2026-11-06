package rules

import (
	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// Unsupported31Keywords returns the extra-unsupported-3-1-keywords rule.
//
// # What it checks
//
// An OpenAPI 3.1 document must not use top-level keywords that tooling around
// the document does not support. By default these are jsonSchemaDialect and
// webhooks.
//
// # Why
//
// Several widely used SDK generators and gateways accept 3.1 documents but
// ignore these parts, so the information silently disappears from generated
// clients and documentation. Flagging the keywords tells authors that they will
// not take effect.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-no-unsupported-keywords).
//
// # Options
//
//   - keywords: the top-level keywords that are reported. Default:
//     jsonSchemaDialect, webhooks.
//
// # Bad
//
//	openapi: 3.1.0
//	info: {title: Test, version: "1.0"}
//	webhooks:
//	  newPet:
//	    post: {}
//
// # Good
//
//	openapi: 3.1.0
//	info: {title: Test, version: "1.0"}
//	paths: {}
//
// # Known false positives / limitations
//
// The keywords are valid OpenAPI 3.1; the rule is only useful when the toolchain
// is known not to support them. Nested use of the names (for example a schema
// property called webhooks) is not reported.
func Unsupported31Keywords() Rule {
	return New(Spec{
		Name:        "unsupported-3-1-keywords",
		Description: "OpenAPI 3.1 documents must not use the top-level keywords jsonSchemaDialect or webhooks.",
		HowToFix:    "Remove the unsupported keyword from the document, or move its content elsewhere.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Formats:     []string{formatOAS31},
		Options: []Option{{
			Name:        "keywords",
			Description: "The top-level keywords that must not be used.",
			Default:     []string{"jsonSchemaDialect", "webhooks"},
		}},
		Check: OnDocument(checkUnsupported31Keywords),
	})
}

func checkUnsupported31Keywords(call *Call) []Finding {
	return seq.FlatMapSlice(call.Strings("keywords"), func(keyword string) []Finding {
		key, _ := oas.Entry(call.Root, keyword)

		return When(key != nil, func() Finding {
			return Finding{
				Node:    key,
				Path:    oas.Join("$", keyword),
				Message: "the OpenAPI 3.1 keyword " + keyword + " is not supported",
			}
		})
	})
}
