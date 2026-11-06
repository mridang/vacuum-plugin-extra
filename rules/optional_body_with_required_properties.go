package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// optionalBodyWithRequiredPropertiesDepth bounds the allOf recursion.
const optionalBodyWithRequiredPropertiesDepth = 8

// OptionalBodyWithRequiredProperties returns the extra-optional-body-with-required-properties rule.
//
// # What it checks
//
// A request body that is optional (its required flag is absent or false) must
// not use a schema that lists required properties. Such a body can be omitted
// entirely, yet when it is sent it must contain the required members, which
// is rarely the intent and leaves clients unsure whether to send an empty
// object. The schema is followed through a local $ref, and the required lists
// of allOf members are taken into account. Request bodies in components are
// checked as well, and operation request bodies that are $ref references are
// resolved so that their required flag is read from the component.
//
// # Where it comes from
//
// IBM's openapi-validator, which warns when an optional request body requires
// properties.
//
// # Options
//
// None.
//
// # Bad
//
//	requestBody:
//	  content:
//	    application/json:
//	      schema:
//	        type: object
//	        required: [name]
//	        properties:
//	          name:
//	            type: string
//
// # Good
//
//	requestBody:
//	  required: true
//	  content:
//	    application/json:
//	      schema:
//	        type: object
//	        required: [name]
//	        properties:
//	          name:
//	            type: string
//
// # Known false positives
//
// A body that is deliberately optional but needs certain members when it is
// present (a partial update) is flagged. Required members reachable only
// through anyOf or oneOf are not considered.
func OptionalBodyWithRequiredProperties() Rule {
	return New(Spec{
		Name:        "optional-body-with-required-properties",
		Description: "An optional request body must not have a schema with required properties.",
		HowToFix:    "Mark the request body as required, or drop the required properties from the schema.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check: Concat(
			OnDocument(optionalBodyWithRequiredPropertiesComponents),
			PerOperation(optionalBodyWithRequiredPropertiesOperation),
		),
	})
}

// optionalBodyWithRequiredPropertiesComponents checks components.requestBodies.
func optionalBodyWithRequiredPropertiesComponents(call *Call) []Finding {
	return seq.FlatMapPairs(
		oas.Pairs(oas.Dig(call.Root, "components", "requestBodies")),
		func(key, body *yaml.Node) []Finding {
			return optionalBodyWithRequiredPropertiesBody(
				call.Root,
				body,
				oas.JoinAll("$.components.requestBodies", key.Value),
			)
		},
	)
}

// optionalBodyWithRequiredPropertiesOperation checks an inline request body.
func optionalBodyWithRequiredPropertiesOperation(call *Call, operation oas.Operation) []Finding {
	declared := oas.Get(operation.Node, "requestBody")
	if oas.Has(declared, "$ref") {
		return nil
	}

	return optionalBodyWithRequiredPropertiesBody(
		call.Root, oas.Resolve(call.Root, declared), oas.Join(operation.Path, "requestBody"))
}

// optionalBodyWithRequiredPropertiesBody reports an optional body whose schema
// requires properties.
func optionalBodyWithRequiredPropertiesBody(root, body *yaml.Node, path string) []Finding {
	if !oas.IsMap(body) || oas.IsTrue(oas.Get(body, "required")) {
		return nil
	}

	media := oas.MediaTypes(oas.Get(body, "content"), oas.Join(path, "content"))

	return When(slices.ContainsFunc(media, func(entry oas.MediaType) bool {
		schema := oas.Resolve(root, oas.Get(entry.Node, "schema"))

		return optionalBodyWithRequiredPropertiesHas(root, schema, optionalBodyWithRequiredPropertiesDepth)
	}), func() Finding {
		return Finding{
			Node:    body,
			Path:    path,
			Message: "an optional request body must not require properties; mark the body as required",
		}
	})
}

// optionalBodyWithRequiredPropertiesHas reports whether the schema, or one of
// its allOf members, lists required properties.
func optionalBodyWithRequiredPropertiesHas(root, schema *yaml.Node, depth int) bool {
	if depth == 0 || !oas.IsMap(schema) {
		return false
	}

	required := oas.Get(schema, "required")
	members := seq.FlatMapPairs(oas.Items(oas.Get(schema, "allOf")), func(_ int, member *yaml.Node) []*yaml.Node {
		return []*yaml.Node{oas.Resolve(root, member)}
	})

	return (oas.IsSeq(required) && len(required.Content) > 0) ||
		slices.ContainsFunc(members, func(member *yaml.Node) bool {
			return optionalBodyWithRequiredPropertiesHas(root, member, depth-1)
		})
}
