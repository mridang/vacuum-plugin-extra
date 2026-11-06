package rules

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// RequiredStringMinLength returns the extra-required-string-min-length rule.
//
// # What it checks
//
// A property that is listed in the required array of its object schema and
// whose own schema is a string must declare minLength. Without a lower bound a
// required string may be empty, which satisfies the schema while carrying no
// information, and clients tend to send "" for values they do not have. The
// property schema is followed through a local $ref. Strings that are enums or
// constants, and 3.1 type lists that do not include string, are skipped.
//
// # Where it comes from
//
// Redocly CLI (rule requiring minLength for required string properties).
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
//	      required: [name]
//	      properties:
//	        name:
//	          type: string
//
// # Good
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      required: [name]
//	      properties:
//	        name:
//	          type: string
//	          minLength: 1
//
// # Known false positives
//
// A required string that is legitimately allowed to be empty is flagged; set
// minLength: 0 to state that explicitly. Required properties declared only in
// an allOf member of another schema are not matched with the properties of
// this schema.
func RequiredStringMinLength() Rule {
	return New(Spec{
		Name:        "required-string-min-length",
		Description: "Required string properties must declare minLength.",
		HowToFix:    "Add minLength to the required string property, for example minLength: 1.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerSchema(checkRequiredStringMinLength),
	})
}

// checkRequiredStringMinLength reports required string properties without a
// minLength.
func checkRequiredStringMinLength(call *Call, schema oas.Schema) []Finding {
	properties := oas.Get(schema.Node, "properties")

	return seq.FlatMapSlice(oas.Strings(oas.Get(schema.Node, "required")), func(name string) []Finding {
		property := oas.Get(properties, name)

		return When(requiredStringMinLengthNeeded(oas.Resolve(call.Root, property)), func() Finding {
			return Finding{
				Node:    property,
				Path:    oas.JoinAll(schema.Path, "properties", name),
				Message: "required string property " + name + " must declare minLength",
			}
		})
	})
}

// requiredStringMinLengthNeeded reports whether the property is a string that
// has no minLength and no fixed set of values.
func requiredStringMinLengthNeeded(property *yaml.Node) bool {
	return oas.HasType(property, "string") &&
		!oas.Has(property, "minLength") &&
		!oas.Has(property, "enum") &&
		!oas.Has(property, "const")
}
