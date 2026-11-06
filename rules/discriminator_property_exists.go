package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// discriminatorPropertyExistsMaxDepth bounds the composition search so that a
// reference cycle cannot loop forever.
const discriminatorPropertyExistsMaxDepth = 16

// DiscriminatorPropertyExists returns the extra-discriminator-property-exists rule.
//
// # What it checks
//
// When a schema declares a discriminator, the property named by
// discriminator.propertyName must be defined by that schema. The property is
// found in the schema's own properties, in any allOf member, or in every
// member of a oneOf or anyOf list (so that each alternative carries it).
// Members are followed through local $ref.
//
// # Why
//
// The discriminator tells clients which property selects the concrete type. If
// the property does not exist, clients cannot read it and the polymorphic
// mapping is unusable.
//
// # Where it comes from
//
// The IBM API Handbook ruleset (discriminator-property-exists).
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
//	        name:
//	          type: string
//
// # Good
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
// # Known false positives / limitations
//
// A discriminator on a base schema whose property is only declared in the
// subtypes that extend it through allOf is flagged, because the base schema is
// judged on its own. External $ref targets are not followed.
func DiscriminatorPropertyExists() Rule {
	return New(Spec{
		Name:        "discriminator-property-exists",
		Description: "The property named by a discriminator must exist in the schema.",
		HowToFix:    "Define the discriminator property in the schema, or change propertyName to an existing property.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkDiscriminatorPropertyExists),
	})
}

// checkDiscriminatorPropertyExists reports discriminators naming a missing property.
func checkDiscriminatorPropertyExists(call *Call, schema oas.Schema) []Finding {
	key, discriminator := oas.Entry(schema.Node, "discriminator")
	name := oas.StringAt(discriminator, "propertyName")
	missing := oas.IsMap(discriminator) && !discriminatorPropertyExistsDefines(call.Root, schema.Node, name, 0)

	return When(missing, func() Finding {
		return Finding{
			Node:    oas.Get(discriminator, "propertyName"),
			Path:    oas.JoinAll(schema.Path, key.Value, "propertyName"),
			Message: "discriminator property " + name + " must be defined in the schema",
		}
	})
}

// discriminatorPropertyExistsDefines reports whether schema defines name.
func discriminatorPropertyExistsDefines(root, schema *yaml.Node, name string, depth int) bool {
	resolved := oas.Resolve(root, schema)
	if name == "" || !oas.IsMap(resolved) || depth > discriminatorPropertyExistsMaxDepth {
		return false
	}

	return oas.Has(oas.Get(resolved, "properties"), name) ||
		slices.ContainsFunc(
			discriminatorPropertyExistsMembers(oas.Get(resolved, "allOf")),
			func(member *yaml.Node) bool {
				return discriminatorPropertyExistsDefines(root, member, name, depth+1)
			},
		) ||
		discriminatorPropertyExistsInEvery(root, oas.Get(resolved, "oneOf"), name, depth) ||
		discriminatorPropertyExistsInEvery(root, oas.Get(resolved, "anyOf"), name, depth)
}

// discriminatorPropertyExistsInEvery reports whether every member of a non-empty
// list defines name.
func discriminatorPropertyExistsInEvery(root, list *yaml.Node, name string, depth int) bool {
	members := discriminatorPropertyExistsMembers(list)

	return len(members) > 0 && !slices.ContainsFunc(members, func(member *yaml.Node) bool {
		return !discriminatorPropertyExistsDefines(root, member, name, depth+1)
	})
}

// discriminatorPropertyExistsMembers returns the elements of a sequence node.
func discriminatorPropertyExistsMembers(list *yaml.Node) []*yaml.Node {
	return seq.FlatMapPairs(oas.Items(list), func(_ int, member *yaml.Node) []*yaml.Node {
		return []*yaml.Node{member}
	})
}
