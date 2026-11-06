package rules

import (
	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// ReadonlyAndWriteonlyExclusive returns the
// extra-readonly-and-writeonly-exclusive rule.
//
// # What it checks
//
// A schema must not set both readOnly and writeOnly to true. The two keywords
// are mutually exclusive: a read-only property is only sent by the server and a
// write-only property is only sent by the client, so a property that is both
// can never appear in any message.
//
// # Where it comes from
//
// KaiZen OpenAPI Parser, which reports a property that is both readOnly and
// writeOnly.
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    User:
//	      type: object
//	      properties:
//	        password:
//	          type: string
//	          readOnly: true
//	          writeOnly: true
//
// # Good
//
//	components:
//	  schemas:
//	    User:
//	      type: object
//	      properties:
//	        password:
//	          type: string
//	          writeOnly: true
//
// # Known false positives
//
// None known. Only the literal value true is considered; a keyword set to false
// is never reported.
func ReadonlyAndWriteonlyExclusive() Rule {
	return New(Spec{
		Name:        "readonly-and-writeonly-exclusive",
		Description: "A schema must not be both readOnly and writeOnly.",
		HowToFix:    "Remove either readOnly or writeOnly from the schema.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkReadonlyAndWriteonlyExclusive),
	})
}

func checkReadonlyAndWriteonlyExclusive(_ *Call, schema oas.Schema) []Finding {
	both := oas.IsTrue(oas.Get(schema.Node, "readOnly")) && oas.IsTrue(oas.Get(schema.Node, "writeOnly"))

	return When(both, func() Finding {
		return Finding{
			Node:    schema.Node,
			Path:    schema.Path,
			Message: "a schema must not be both readOnly and writeOnly",
		}
	})
}
