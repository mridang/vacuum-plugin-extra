package rules

import (
	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// NullableRequiresType returns the extra-nullable-requires-type rule.
//
// # What it checks
//
// A schema that carries the nullable keyword must also declare a type next to
// it. Every schema of the document is inspected: component schemas, parameter,
// request body and response schemas and everything nested below them.
//
// # Why
//
// In OpenAPI 3.0 nullable only modifies the type of the same schema object. The
// specification states that nullable: true without a sibling type has no
// effect, so a schema such as an allOf wrapper with nullable: true silently
// stays non-nullable in tools that follow it, while other tools assume the
// opposite.
//
// # Where it comes from
//
// Redocly CLI and speakeasy (nullable requires a sibling type).
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Owner:
//	      nullable: true
//	      allOf:
//	        - $ref: "#/components/schemas/User"
//
// # Good
//
//	components:
//	  schemas:
//	    Owner:
//	      type: object
//	      nullable: true
//	      allOf:
//	        - $ref: "#/components/schemas/User"
//
// # Known false positives and limitations
//
// OpenAPI 3.1 removed nullable in favour of type arrays; the keyword is still
// flagged there when it appears without a type, which is a leftover to clean up
// as well. A type declared only inside an allOf member is not seen as a
// sibling type.
func NullableRequiresType() Rule {
	return New(Spec{
		Name:        "nullable-requires-type",
		Description: "A schema using nullable must declare a sibling type.",
		HowToFix:    "Add a type next to nullable, for example type: object.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkNullableRequiresType),
	})
}

// checkNullableRequiresType reports a schema with nullable and no type.
func checkNullableRequiresType(_ *Call, schema oas.Schema) []Finding {
	key, nullable := oas.Entry(schema.Node, "nullable")

	return When(nullable != nil && !oas.Has(schema.Node, "type"), func() Finding {
		return Finding{
			Node:    key,
			Path:    oas.Join(schema.Path, "nullable"),
			Message: "nullable requires a sibling type",
		}
	})
}
