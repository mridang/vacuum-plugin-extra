package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const noInlineSchemaBoolean = "boolean"

const (
	noInlineSchemaParameters      = "parameters"
	noInlineSchemaRequestBodies   = "requestBodies"
	noInlineSchemaResponses       = "responses"
	noInlineSchemaAllowPrimitives = "allowPrimitives"
)

// NoInlineSchema returns the extra-no-inline-schema rule.
//
// # What it checks
//
// The schema of an operation parameter, of a request body media type and of a
// response media type must be a $ref to a reusable schema instead of an inline
// definition. Path-level and operation-level parameters are covered; both the
// schema and the content schema forms of a parameter are checked. Each of the
// three locations can be switched off by an option.
//
// # Why
//
// Code generators and API portals name an inline schema after its position in
// the document, which yields unstable and unreadable type names such as
// GetPetsResponse200. Reusable named schemas give stable names, enable reuse
// and keep the paths section small.
//
// # Where it comes from
//
// rate-my-openapi, the three apimatic-no-inline rules for parameter, request
// body and response schemas, merged into one rule with an option per location.
//
// # Options
//
//   - parameters (bool, default true): check parameter schemas.
//   - requestBodies (bool, default true): check request body schemas.
//   - responses (bool, default true): check response schemas.
//   - allowPrimitives (bool, default false): accept inline schemas whose type is
//     a plain string, number, integer or boolean, which have no useful name to
//     extract. Mainly useful for parameters.
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
//
// # Good
//
//	schema:
//	  $ref: "#/components/schemas/NewPet"
//
// # Known false positives and limitations
//
// With the defaults every inline parameter schema is reported, including
// trivial ones such as type: string; enable allowPrimitives to relax that.
// Reusable parameters, request bodies and responses declared under components
// are not checked, since extracting an inline schema out of them is their own
// refactoring.
func NoInlineSchema() Rule {
	return New(Spec{
		Name:        "no-inline-schema",
		Description: "Parameter, request body and response schemas must be $refs to reusable schemas.",
		HowToFix:    "Move the inline schema to components/schemas and reference it with $ref.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{Name: noInlineSchemaParameters, Description: "Check parameter schemas.", Default: true},
			{Name: noInlineSchemaRequestBodies, Description: "Check request body schemas.", Default: true},
			{Name: noInlineSchemaResponses, Description: "Check response schemas.", Default: true},
			{
				Name:        noInlineSchemaAllowPrimitives,
				Description: "Accept inline string, number, integer and boolean schemas.",
				Default:     false,
			},
		},
		Check: Concat(
			PerPathItem(checkNoInlineSchemaPathItem),
			PerOperation(checkNoInlineSchemaOperation),
		),
	})
}

// checkNoInlineSchemaPathItem reports inline schemas of the parameters shared by the operations of a path item.
func checkNoInlineSchemaPathItem(call *Call, item oas.PathItem) []Finding {
	return noInlineSchemaParameterList(call, oas.Get(item.Node, "parameters"), oas.Join(item.Path, "parameters"))
}

// checkNoInlineSchemaOperation reports inline schemas at the enabled locations of an operation.
func checkNoInlineSchemaOperation(call *Call, operation oas.Operation) []Finding {
	return slices.Concat(
		noInlineSchemaParameterList(
			call,
			oas.Get(operation.Node, "parameters"),
			oas.Join(operation.Path, "parameters"),
		),
		noInlineSchemaRequestBody(call, operation),
		noInlineSchemaResponseContent(call, operation),
	)
}

// noInlineSchemaRequestBody checks the request body content schemas of an operation.
func noInlineSchemaRequestBody(call *Call, operation oas.Operation) []Finding {
	if !call.Bool(noInlineSchemaRequestBodies) {
		return nil
	}

	return noInlineSchemaContent(
		call,
		oas.Dig(operation.Node, "requestBody", "content"),
		oas.JoinAll(operation.Path, "requestBody", "content"),
		"request body",
	)
}

// noInlineSchemaResponseContent checks the content schemas of the responses of an operation.
func noInlineSchemaResponseContent(call *Call, operation oas.Operation) []Finding {
	if !call.Bool(noInlineSchemaResponses) {
		return nil
	}

	return seq.FlatMapPairs(oas.Pairs(oas.Get(operation.Node, "responses")), func(code, response *yaml.Node) []Finding {
		path := oas.JoinAll(operation.Path, "responses", code.Value, "content")

		return noInlineSchemaContent(call, oas.Get(response, "content"), path, "response")
	})
}

// noInlineSchemaParameterList checks the schemas of a parameter list.
func noInlineSchemaParameterList(call *Call, list *yaml.Node, path string) []Finding {
	if !call.Bool(noInlineSchemaParameters) {
		return nil
	}

	return seq.FlatMapPairs(oas.Items(list), func(i int, parameter *yaml.Node) []Finding {
		base := oas.Index(path, i)

		return slices.Concat(
			noInlineSchemaReport(call, oas.Get(parameter, "schema"), oas.Join(base, "schema"), "parameter"),
			noInlineSchemaContent(call, oas.Get(parameter, "content"), oas.Join(base, "content"), "parameter"),
		)
	})
}

// noInlineSchemaContent checks the schema of every media type of a content map.
func noInlineSchemaContent(call *Call, content *yaml.Node, path string, what string) []Finding {
	return seq.FlatMapSlice(oas.MediaTypes(content, path), func(media oas.MediaType) []Finding {
		return noInlineSchemaReport(call, oas.Get(media.Node, "schema"), oas.Join(media.Path, "schema"), what)
	})
}

// noInlineSchemaReport returns a finding when the schema is defined inline.
func noInlineSchemaReport(call *Call, schema *yaml.Node, path string, what string) []Finding {
	if !oas.IsMap(schema) || oas.Has(schema, "$ref") {
		return nil
	}

	return When(!call.Bool(noInlineSchemaAllowPrimitives) || !noInlineSchemaPrimitive(schema), func() Finding {
		return Finding{
			Node:    schema,
			Path:    path,
			Message: what + " schema should be moved to components/schemas and referenced",
		}
	})
}

// noInlineSchemaPrimitive reports whether the schema is a plain scalar type.
func noInlineSchemaPrimitive(schema *yaml.Node) bool {
	types := oas.Types(schema)
	if len(types) == 0 {
		return false
	}

	for _, kind := range types {
		switch kind {
		case literalString, keywordNumber, keywordInteger, noInlineSchemaBoolean, keywordNull:
		default:
			return false
		}
	}

	return true
}
