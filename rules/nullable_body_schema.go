package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// NullableBodySchema returns the extra-nullable-body-schema rule.
//
// # What it checks
//
// The schema of a request body media type must not be nullable. Both the
// OpenAPI 3.0 nullable: true flag and an OpenAPI 3.1 type list that includes
// null are reported. A body that may be null cannot be told apart from an
// absent body in many clients and servers; when a body is optional, mark the
// request body as not required, and when a member is optional, make it an
// optional property of an object. The schema is followed through a local $ref
// before it is inspected, and the finding is reported at the media type
// schema. Request bodies declared in components are checked as well.
//
// # Where it comes from
//
// Oxide's OpenAPI lint, which asks for a struct with an optional member
// instead of a nullable request body.
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
//	        nullable: true
//
// # Good
//
//	requestBody:
//	  required: false
//	  content:
//	    application/json:
//	      schema:
//	        type: object
//
// # Known false positives
//
// Nullability inside the body (properties, items) is not reported, and neither
// is nullability introduced only through allOf, anyOf or oneOf branches.
func NullableBodySchema() Rule {
	return New(Spec{
		Name:        "nullable-body-schema",
		Description: "Request body schemas must not be nullable.",
		HowToFix:    "Remove nullable from the body schema and make the body or one of its members optional instead.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkNullableBodySchema),
	})
}

func checkNullableBodySchema(call *Call) []Finding {
	return slices.Concat(
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(call.Root, "components", "requestBodies")),
			func(key, body *yaml.Node) []Finding {
				return nullableBodySchemaBody(call.Root, body, oas.JoinAll("$.components.requestBodies", key.Value))
			},
		),
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
			return nullableBodySchemaBody(
				call.Root,
				oas.Get(operation.Node, "requestBody"),
				oas.Join(operation.Path, "requestBody"),
			)
		}),
	)
}

func nullableBodySchemaBody(root, body *yaml.Node, path string) []Finding {
	if oas.Has(body, "$ref") {
		return nil
	}

	return seq.FlatMapSlice(
		oas.MediaTypes(oas.Get(body, "content"), oas.Join(path, "content")),
		func(media oas.MediaType) []Finding {
			schema := oas.Get(media.Node, "schema")

			return When(nullableBodySchemaIsNullable(oas.Resolve(root, schema)), func() Finding {
				return Finding{
					Node:    schema,
					Path:    oas.Join(media.Path, "schema"),
					Message: "request body schema for " + media.Name + " must not be nullable",
				}
			})
		},
	)
}

func nullableBodySchemaIsNullable(schema *yaml.Node) bool {
	return oas.IsTrue(oas.Get(schema, "nullable")) || slices.Contains(oas.Types(schema), "null")
}
