package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// ContentMustHaveSchema returns the extra-content-must-have-schema rule.
//
// # What it checks
//
// Every entry of a content map must contain a schema. This applies to request
// bodies, responses, response headers and parameters that use content instead
// of schema, including the reusable ones declared under components. A media
// type with no schema tells tools nothing about the payload, so generated
// clients and validators fall back to untyped data. Objects that are $ref
// references are skipped because their target is checked where it is declared.
//
// # Where it comes from
//
// IBM's openapi-validator (ibm-content-contains-schema).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json: {}
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              schema:
//	                type: array
//	                items:
//	                  type: string
//
// # Known false positives
//
// Media types that carry opaque bytes (for example application/octet-stream)
// are flagged even though a schema adds little; use type: string with format:
// binary to satisfy the rule.
func ContentMustHaveSchema() Rule {
	return New(Spec{
		Name:        "content-must-have-schema",
		Description: "Every content entry of a request body, response or parameter must contain a schema.",
		HowToFix:    "Add a schema to the media type object.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkContentMustHaveSchema),
	})
}

func checkContentMustHaveSchema(call *Call) []Finding {
	return seq.FlatMapSlice(contentMustHaveSchemaOwners(call.Root), func(owner contentMustHaveSchemaOwner) []Finding {
		return seq.FlatMapSlice(
			oas.MediaTypes(oas.Get(owner.node, "content"), oas.Join(owner.path, "content")),
			func(media oas.MediaType) []Finding {
				return When(
					oas.IsMap(media.Node) && !oas.Has(media.Node, "schema") && !oas.Has(media.Node, "$ref"),
					func() Finding {
						return Finding{
							Node:    media.Node,
							Path:    media.Path,
							Message: "content entry " + media.Name + " must define a schema",
						}
					},
				)
			},
		)
	})
}

type contentMustHaveSchemaOwner struct {
	node *yaml.Node
	path string
}

func contentMustHaveSchemaOwners(root *yaml.Node) []contentMustHaveSchemaOwner {
	components := oas.Get(root, "components")

	return slices.Concat(
		seq.FlatMapSlice(
			[]string{literalParameters, keywordHeaders, "requestBodies", "responses"},
			func(section string) []contentMustHaveSchemaOwner {
				return seq.FlatMapPairs(
					oas.Pairs(oas.Get(components, section)),
					func(key, value *yaml.Node) []contentMustHaveSchemaOwner {
						return contentMustHaveSchemaWithHeaders(value, oas.JoinAll("$.components", section, key.Value))
					},
				)
			},
		),
		seq.FlatMap(oas.Operations(root), contentMustHaveSchemaOperation),
		seq.FlatMap(oas.PathItems(root), func(item oas.PathItem) []contentMustHaveSchemaOwner {
			return contentMustHaveSchemaParameters(item.Node, item.Path)
		}),
	)
}

func contentMustHaveSchemaOwnerOf(node *yaml.Node, path string) []contentMustHaveSchemaOwner {
	if !oas.IsMap(node) || oas.Has(node, "$ref") {
		return nil
	}

	return []contentMustHaveSchemaOwner{{node: node, path: path}}
}

func contentMustHaveSchemaParameters(owner *yaml.Node, path string) []contentMustHaveSchemaOwner {
	return seq.FlatMapPairs(
		oas.Items(oas.Get(owner, "parameters")),
		func(i int, parameter *yaml.Node) []contentMustHaveSchemaOwner {
			return contentMustHaveSchemaOwnerOf(parameter, oas.Index(oas.Join(path, "parameters"), i))
		},
	)
}

func contentMustHaveSchemaOperation(operation oas.Operation) []contentMustHaveSchemaOwner {
	return slices.Concat(
		contentMustHaveSchemaParameters(operation.Node, operation.Path),
		contentMustHaveSchemaOwnerOf(oas.Get(operation.Node, "requestBody"), oas.Join(operation.Path, "requestBody")),
		seq.FlatMapPairs(
			oas.Pairs(oas.Get(operation.Node, "responses")),
			func(key, response *yaml.Node) []contentMustHaveSchemaOwner {
				return contentMustHaveSchemaWithHeaders(response, oas.JoinAll(operation.Path, "responses", key.Value))
			},
		),
	)
}

func contentMustHaveSchemaWithHeaders(node *yaml.Node, path string) []contentMustHaveSchemaOwner {
	return slices.Concat(
		contentMustHaveSchemaOwnerOf(node, path),
		seq.FlatMapPairs(
			oas.Pairs(oas.Get(node, "headers")),
			func(name, header *yaml.Node) []contentMustHaveSchemaOwner {
				return contentMustHaveSchemaOwnerOf(header, oas.JoinAll(path, "headers", name.Value))
			},
		),
	)
}
