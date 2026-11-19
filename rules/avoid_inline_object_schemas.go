package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// AvoidInlineObjectSchemas returns the extra-avoid-inline-object-schemas rule.
//
// # What it checks
//
// Object schemas must be defined once under components/schemas and referenced
// with $ref instead of being written inline. A schema is reported when it is
// not itself a $ref, it does not live directly under components/schemas, and it
// is an object schema: it defines properties or it composes other schemas with
// allOf, anyOf or oneOf. The schema positions that are inspected are request
// and response bodies, parameters and headers, and every nested position
// (properties, items, additionalProperties and the composition keywords).
// Primitive schemas such as a string or an array of strings are never reported,
// and neither is a schema below not, if, then or else, because those schemas
// constrain a value rather than describe one.
//
// Named models give code generators a type name to use, keep the description
// of a structure in one place and make the document smaller. Anonymous inline
// objects force generators to invent names and make it impossible to reuse the
// structure.
//
// # Where it comes from
//
// IBM openapi-validator, rule ibm-avoid-inline-schemas.
//
// # Options
//
//   - allowReferenceUnions (bool, default false): do not report a oneOf, anyOf
//     or allOf whose members are all open: a $ref, a null type, an enum, a
//     scalar type (string, number, integer or boolean) or a free-form object.
//     Such a union defines no object with properties itself, so it is often
//     acceptable inline. A member with properties or its own composition is
//     still an inline object and keeps the union reported.
//   - allowAllOfExtension (bool, default false): do not report an inline allOf
//     member when a sibling member of the same allOf is a $ref. That shape is
//     how a schema extends a named schema with extra properties. An allOf of
//     inline members only is still reported.
//
// # Bad
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        content:
//	          application/json:
//	            schema:
//	              type: object
//	              properties:
//	                name:
//	                  type: string
//
// # Good
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        content:
//	          application/json:
//	            schema:
//	              $ref: "#/components/schemas/Pet"
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      properties:
//	        name:
//	          type: string
//
// # Known false positives
//
// Every inline object is reported, including objects nested inside another
// inline object, so one inline structure can produce several findings.
// Schemas that are intentionally anonymous, such as a one-off error payload,
// have to be named or the rule suppressed for that location. An object schema
// without properties or composition (a free-form object) is not reported.
func AvoidInlineObjectSchemas() Rule {
	return New(Spec{
		Name:        "avoid-inline-object-schemas",
		Description: "Object schemas must be defined in components/schemas and referenced instead of being inlined.",
		HowToFix:    "Move the inline object schema to components/schemas and reference it with $ref.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        "allowReferenceUnions",
			Description: "Accept compositions whose members are all refs, nulls, enums, scalars or free-form objects.",
			Default:     false,
		}, {
			Name:        "allowAllOfExtension",
			Description: "Do not report an inline allOf member when a sibling member of the same allOf is a $ref.",
			Default:     false,
		}, {
			Name:        "allowNested",
			Description: "Do not report an inline object nested inside a named schema under components/schemas.",
			Default:     false,
		}},
		Check: func(call *Call) []Finding {
			extensions := avoidInlineObjectSchemasExtensions(call)

			return PerSchema(func(call *Call, schema oas.Schema) []Finding {
				return checkAvoidInlineObjectSchemas(call, schema, extensions)
			})(call)
		},
	})
}

// avoidInlineObjectSchemasExtensions returns the inline allOf members that
// extend a named schema: members of an allOf that also holds a $ref member. It
// is empty unless allowAllOfExtension is set.
func avoidInlineObjectSchemasExtensions(call *Call) map[*yaml.Node]bool {
	extensions := map[*yaml.Node]bool{}

	if !call.Bool("allowAllOfExtension") {
		return extensions
	}

	for schema := range oas.Schemas(call.Root) {
		members := oas.Get(schema.Node, avoidInlineObjectSchemasAllOf)
		if members == nil || !slices.ContainsFunc(members.Content, func(member *yaml.Node) bool {
			return oas.Ref(member) != ""
		}) {
			continue
		}

		for _, member := range members.Content {
			extensions[member] = true
		}
	}

	return extensions
}

func checkAvoidInlineObjectSchemas(call *Call, schema oas.Schema, extensions map[*yaml.Node]bool) []Finding {
	inline := !extensions[schema.Node] &&
		avoidInlineObjectSchemasIsInlineObject(schema.Node, call.Bool("allowReferenceUnions"))

	negated := oas.UnderNegation(schema.Path)

	nested := call.Bool("allowNested") && strings.HasPrefix(schema.Path, avoidInlineObjectSchemasComponents)

	return When(schema.Component == "" && inline && !negated && !nested, func() Finding {
		return Finding{
			Node:    schema.Node,
			Path:    schema.Path,
			Message: "object schemas must be defined in components/schemas and referenced with $ref",
		}
	})
}

func avoidInlineObjectSchemasIsInlineObject(schema *yaml.Node, allowUnions bool) bool {
	if oas.Ref(schema) != "" {
		return false
	}

	if !oas.Empty(oas.Get(schema, "properties")) {
		return true
	}

	for _, keyword := range []string{avoidInlineObjectSchemasAllOf, avoidInlineObjectSchemasAnyOf, literalOneof} {
		list := oas.Get(schema, keyword)
		if !oas.Empty(list) && (!allowUnions || avoidInlineObjectSchemasHasInlineMember(list)) {
			return true
		}
	}

	return false
}

// avoidInlineObjectSchemasHasInlineMember reports whether a composition list
// holds a member that is not open. A union whose members are all open defines
// no object with properties itself.
func avoidInlineObjectSchemasHasInlineMember(list *yaml.Node) bool {
	if list == nil {
		return false
	}

	return slices.ContainsFunc(list.Content, func(member *yaml.Node) bool {
		return !avoidInlineObjectSchemasIsOpenMember(member)
	})
}

// avoidInlineObjectSchemasIsOpenMember reports whether a union member defines no
// object with properties of its own: a $ref, a plain null type, an enum, a scalar
// type or a free-form object. A member with properties or a composition is not open.
func avoidInlineObjectSchemasIsOpenMember(member *yaml.Node) bool {
	isNull := len(oas.Keys(member)) == 1 && oas.StringAt(member, "type") == "null"

	if oas.Ref(member) != "" || isNull || !oas.Empty(oas.Get(member, "enum")) {
		return true
	}

	if avoidInlineObjectSchemasHasStructure(member) {
		return false
	}

	return avoidInlineObjectSchemasHasScalarOrFreeFormType(member)
}

// avoidInlineObjectSchemasHasStructure reports whether a schema defines
// properties or composes other schemas.
func avoidInlineObjectSchemasHasStructure(schema *yaml.Node) bool {
	if !oas.Empty(oas.Get(schema, "properties")) {
		return true
	}

	for _, keyword := range []string{avoidInlineObjectSchemasAllOf, avoidInlineObjectSchemasAnyOf, literalOneof} {
		if !oas.Empty(oas.Get(schema, keyword)) {
			return true
		}
	}

	return false
}

// avoidInlineObjectSchemasHasScalarOrFreeFormType reports whether a schema
// declares a scalar type or the object type. Callers check the structure first,
// so an object here has no properties and is a free-form object.
func avoidInlineObjectSchemasHasScalarOrFreeFormType(schema *yaml.Node) bool {
	for _, typeName := range []string{"string", "number", "integer", avoidInlineObjectSchemasBoolean, "object"} {
		if oas.HasType(schema, typeName) {
			return true
		}
	}

	return false
}

// Keyword names used by this rule.
const (
	avoidInlineObjectSchemasAllOf   = "allOf"
	avoidInlineObjectSchemasAnyOf   = "anyOf"
	avoidInlineObjectSchemasBoolean = "boolean"
	// avoidInlineObjectSchemasComponents prefixes the path of every schema
	// inside a named schema under components/schemas.
	avoidInlineObjectSchemasComponents = "$.components.schemas"
)
