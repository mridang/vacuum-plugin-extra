package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// Keywords that appear more than once in this rule.
const (
	paramNameVsBodyPropertyNameJSON  = "application/json"
	paramNameVsBodyPropertyNameAllOf = "allOf"
)

// ParamNameVsBodyPropertyName returns the extra-param-name-vs-body-property-name rule.
//
// # What it checks
//
// The name of a parameter of an operation, in any location and including
// parameters declared on the path item, must not equal the name of a top-level
// property of the operation's JSON request body schema. The properties of
// allOf, oneOf and anyOf members are included, $ref targets are followed, and
// read-only properties are ignored because clients never send them. The JSON
// request body is the first media type that is application/json or ends in
// +json.
//
// # Why
//
// Many SDK generators flatten parameters and body properties into one argument
// list. Two inputs with the same name then collide, and the generator either
// fails or silently drops one of them.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-unique-parameter-request-property-names).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets/{id}:
//	    put:
//	      parameters:
//	        - {name: id, in: path, required: true, schema: {type: string}}
//	      requestBody:
//	        content:
//	          application/json:
//	            schema:
//	              type: object
//	              properties:
//	                id: {type: string}
//
// # Good
//
//	paths:
//	  /pets/{id}:
//	    put:
//	      parameters:
//	        - {name: id, in: path, required: true, schema: {type: string}}
//	      requestBody:
//	        content:
//	          application/json:
//	            schema:
//	              type: object
//	              properties:
//	                name: {type: string}
//
// # Known false positives / limitations
//
// Only top-level properties are compared, and the comparison is case-sensitive.
// Non JSON request bodies are ignored.
func ParamNameVsBodyPropertyName() Rule {
	return New(Spec{
		Name:        "param-name-vs-body-property-name",
		Description: "Parameter names must not equal top-level property names of the JSON request body.",
		HowToFix:    "Rename the parameter or the request body property so that the names are unique.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkParamNameVsBodyPropertyName),
	})
}

func checkParamNameVsBodyPropertyName(call *Call, operation oas.Operation) []Finding {
	schema := paramNameVsBodyPropertyNameBodySchema(call.Root, operation)
	if schema == nil {
		return nil
	}

	names := paramNameVsBodyPropertyNameCollect(call.Root, schema, nil)

	return seq.FlatMapSlice(oas.Parameters(call.Root, operation), func(parameter oas.Parameter) []Finding {
		return When(slices.Contains(names, parameter.Name()), func() Finding {
			return Finding{
				Node:    parameter.Node,
				Path:    parameter.Path,
				Message: "parameter '" + parameter.Name() + "' has the same name as a property of the request body",
			}
		})
	})
}

// paramNameVsBodyPropertyNameBodySchema returns the schema of the first JSON
// media type of the request body, or nil.
func paramNameVsBodyPropertyNameBodySchema(root *yaml.Node, operation oas.Operation) *yaml.Node {
	body := oas.Resolve(root, oas.Get(operation.Node, "requestBody"))

	for name, media := range oas.Pairs(oas.Get(body, "content")) {
		if paramNameVsBodyPropertyNameIsJSON(name.Value) {
			return oas.Get(media, "schema")
		}
	}

	return nil
}

// paramNameVsBodyPropertyNameIsJSON reports whether the media type is JSON.
func paramNameVsBodyPropertyNameIsJSON(mediaType string) bool {
	base, _, _ := strings.Cut(mediaType, ";")
	folded := Fold(base)

	return folded == paramNameVsBodyPropertyNameJSON || strings.HasSuffix(folded, "+json")
}

// paramNameVsBodyPropertyNameCollect gathers the writable property names of the
// schema and of its composition members. The ancestors guard against reference cycles.
func paramNameVsBodyPropertyNameCollect(root, node *yaml.Node, ancestors []*yaml.Node) []string {
	schema := oas.Resolve(root, node)
	if !oas.IsMap(schema) || slices.Contains(ancestors, schema) {
		return nil
	}

	path := slices.Concat(ancestors, []*yaml.Node{schema})

	return slices.Concat(
		seq.FlatMapPairs(oas.Pairs(oas.Get(schema, "properties")), func(name, property *yaml.Node) []string {
			if oas.IsTrue(oas.Get(oas.Resolve(root, property), "readOnly")) {
				return nil
			}

			return []string{name.Value}
		}),
		seq.FlatMapSlice(
			[]string{paramNameVsBodyPropertyNameAllOf, literalOneof, keywordAnyOf},
			func(keyword string) []string {
				return seq.FlatMapPairs(oas.Items(oas.Get(schema, keyword)), func(_ int, member *yaml.Node) []string {
					return paramNameVsBodyPropertyNameCollect(root, member, path)
				})
			},
		),
	)
}
