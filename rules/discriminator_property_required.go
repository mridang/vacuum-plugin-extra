package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// DiscriminatorPropertyRequired returns the
// extra-discriminator-property-required rule.
//
// # What it checks
//
// When a schema declares a discriminator, the property named by the
// discriminator must be listed in the required array of that schema. The
// required list of any allOf member of the schema is accepted as well, because
// a base type is often composed that way. A discriminator given as a plain
// string (Swagger 2.0 style) is handled in the same way.
//
// The OpenAPI specification states that the discriminator property must be
// required. Without it a payload may omit the property and no consumer can
// decide which subtype applies.
//
// # Where it comes from
//
// KaiZen OpenAPI Parser, which validates that the discriminator property is
// part of the schema's required list.
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
//	      discriminator:
//	        propertyName: kind
//	      properties:
//	        kind:
//	          type: string
//
// # Good
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      required: [kind]
//	      discriminator:
//	        propertyName: kind
//	      properties:
//	        kind:
//	          type: string
//
// # Known false positives
//
// A schema that only uses oneOf or anyOf with a discriminator and defines the
// property in its members, not in itself, is reported. The required list of a
// member reached through $ref inside allOf is resolved, but required lists
// behind external references are not seen.
func DiscriminatorPropertyRequired() Rule {
	return New(Spec{
		Name:        "discriminator-property-required",
		Description: "The discriminator property must be listed in the required list of the schema.",
		HowToFix:    "Add the discriminator property to the required list of the schema.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkDiscriminatorPropertyRequired),
	})
}

func checkDiscriminatorPropertyRequired(call *Call, schema oas.Schema) []Finding {
	discriminator := oas.Get(schema.Node, "discriminator")
	property := discriminatorPropertyRequiredName(discriminator)
	if property == "" || discriminatorPropertyRequiredListed(call.Root, schema.Node, property) {
		return nil
	}

	return One(Finding{
		Node:    discriminator,
		Path:    discriminatorPropertyRequiredPath(schema, discriminator),
		Message: "discriminator property " + property + " must be listed in the required list of the schema",
	})
}

func discriminatorPropertyRequiredPath(schema oas.Schema, discriminator *yaml.Node) string {
	if oas.IsMap(discriminator) {
		return oas.JoinAll(schema.Path, "discriminator", "propertyName")
	}

	return oas.Join(schema.Path, "discriminator")
}

func discriminatorPropertyRequiredName(discriminator *yaml.Node) string {
	if oas.IsScalar(discriminator) {
		return discriminator.Value
	}

	return oas.StringAt(discriminator, "propertyName")
}

func discriminatorPropertyRequiredListed(root, schema *yaml.Node, property string) bool {
	return discriminatorPropertyRequiredListedWithin(root, schema, property, 0)
}

// discriminatorPropertyRequiredMaxDepth bounds the walk through nested compositions.
const discriminatorPropertyRequiredMaxDepth = 8

// discriminatorPropertyRequiredListedWithin reports whether every value of the
// schema must carry the property: it is required by the schema or one of its
// allOf members, or every oneOf or anyOf member requires it in the same way.
func discriminatorPropertyRequiredListedWithin(root, schema *yaml.Node, property string, depth int) bool {
	if depth > discriminatorPropertyRequiredMaxDepth {
		return false
	}

	if slices.Contains(oas.Strings(oas.Get(schema, "required")), property) {
		return true
	}

	for _, member := range oas.Items(oas.Get(schema, "allOf")) {
		if discriminatorPropertyRequiredListedWithin(root, oas.Resolve(root, member), property, depth+1) {
			return true
		}
	}

	for _, keyword := range []string{"oneOf", "anyOf"} {
		if discriminatorPropertyRequiredAllMembers(root, oas.Get(schema, keyword), property, depth) {
			return true
		}
	}

	return false
}

// discriminatorPropertyRequiredAllMembers reports whether a non-empty oneOf or
// anyOf list has only members that require the property.
func discriminatorPropertyRequiredAllMembers(root, list *yaml.Node, property string, depth int) bool {
	count := 0

	for _, member := range oas.Items(list) {
		count++

		if !discriminatorPropertyRequiredListedWithin(root, oas.Resolve(root, member), property, depth+1) {
			return false
		}
	}

	return count > 0
}
