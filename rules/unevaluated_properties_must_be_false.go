package rules

import (
	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// UnevaluatedPropertiesMustBeFalse returns the extra-unevaluated-properties-must-be-false rule.
//
// # What it checks
//
// In OpenAPI 3.1 documents, a schema that sets unevaluatedProperties must set
// it to the boolean false. Schemas without the keyword are not affected, and
// the rule does not run on OpenAPI 3.0 documents. A true value or a schema
// value is reported on the keyword.
//
// # Why
//
// unevaluatedProperties: false closes an object after composition (allOf,
// oneOf, if/then). Any other value either reopens the object to arbitrary extra
// properties or hides a second set of rules that clients cannot reliably
// follow; policy-wise the choice is between closed and silent.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (ibm-unevaluated-properties).
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      unevaluatedProperties: true
//
// # Good
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      unevaluatedProperties: false
//
// # Known false positives
//
// Schemas that deliberately accept typed extras with a schema value, such as
// unevaluatedProperties: {type: string}, are flagged and need a suppression.
func UnevaluatedPropertiesMustBeFalse() Rule {
	return New(Spec{
		Name:        "unevaluated-properties-must-be-false",
		Description: "unevaluatedProperties, when set, must be false.",
		HowToFix:    "Set unevaluatedProperties to false or remove it.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Formats:     []string{formatOAS31},
		Check:       PerSchema(checkUnevaluatedPropertiesMustBeFalse),
	})
}

// checkUnevaluatedPropertiesMustBeFalse reports unevaluatedProperties values other than false.
func checkUnevaluatedPropertiesMustBeFalse(call *Call, schema oas.Schema) []Finding {
	if !oas.IsOAS31(call.Root) {
		return nil
	}

	key, value := oas.Entry(schema.Node, "unevaluatedProperties")

	return When(value != nil && (!oas.IsScalar(value) || value.Value != "false"), func() Finding {
		return Finding{
			Node:    value,
			Path:    oas.Join(schema.Path, key.Value),
			Message: "unevaluatedProperties must be false when it is set",
		}
	})
}
