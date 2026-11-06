package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// requestResponseMustBeModelsMaxDepth bounds the recursion through schemas.
const requestResponseMustBeModelsMaxDepth = 16

// Keyword names and media types used by this rule.
const (
	requestResponseMustBeModelsAllOf     = "allOf"
	requestResponseMustBeModelsAnyOf     = "anyOf"
	requestResponseMustBeModelsJSON      = "application/json"
	requestResponseMustBeModelsJSONPatch = "application/json-patch+json"
	requestResponseMustBeModelsMergeJSON = "application/merge-patch+json"
	requestResponseMustBeModelsObject    = "object"
)

// RequestResponseMustBeModels returns the extra-request-response-must-be-models
// rule.
//
// # What it checks
//
// Every request body schema and every response body schema whose media type
// uses a JSON representation (application/json, application/json-patch+json
// and application/merge-patch+json, each optionally followed by parameters)
// and whose schema is an object schema must define at least one static
// property. Schemas are followed through local $ref.
//
// A schema is an object schema when it, or a schema composed into it, has
// type object (alone or in a type array), or has no type but defines
// properties, patternProperties, or an additionalProperties that is true or a
// schema. The composition semantics are the ones IBM uses everywhere: a
// predicate holds for a schema when the schema itself satisfies it, or when any
// allOf member satisfies it, or when all members of a non-empty oneOf satisfy
// it, or when all members of a non-empty anyOf satisfy it. The schema "defines
// properties" under the same composition semantics when a properties map has at
// least one entry.
//
// Schemas that are not object schemas (arrays, primitives) are deliberately not
// reported by this rule, exactly like the original: other rules of the family
// cover them. A body that is a dictionary (additionalProperties only) is an
// object schema without properties and is reported.
//
// # Why
//
// Models can evolve by adding properties without breaking clients, and SDK
// generators produce a named type for them. A dictionary or an empty object
// cannot be turned into a typed model.
//
// # Where it comes from
//
// IBM openapi-validator, ibm-accept-and-return-models (Request and response
// bodies must be models - their schemas must define properties). The original
// depends on its sibling rules to catch non-object bodies; so does this one.
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
//	            application/json:
//	              schema:
//	                type: object
//	                additionalProperties:
//	                  type: string
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
//	                type: object
//	                properties:
//	                  pets:
//	                    type: array
//	                    items:
//	                      type: string
//
// # Known false positives and limitations
//
// Only local references are followed, so a body whose schema is an external
// $ref is not analysed. The finding is placed on the schema as written in the
// operation, whereas the original reports the resolved location.
func RequestResponseMustBeModels() Rule {
	return New(Spec{
		Name:        "request-response-must-be-models",
		Description: "JSON request and response body object schemas must define static properties (models, not dictionaries).",
		HowToFix:    "Define the properties of the body in its object schema.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkRequestResponseMustBeModels),
	})
}

// checkRequestResponseMustBeModels inspects the request and response bodies of
// one operation.
func checkRequestResponseMustBeModels(call *Call, operation oas.Operation) []Finding {
	body := oas.Resolve(call.Root, oas.Get(operation.Node, "requestBody"))
	request := requestResponseMustBeModelsContent(
		call.Root,
		oas.Get(body, "content"),
		oas.JoinAll(operation.Path, "requestBody", "content"),
	)

	return slices.Concat(
		request,
		seq.FlatMapSlice(oas.Responses(call.Root, operation), func(response oas.Response) []Finding {
			content := oas.Get(response.Node, "content")

			return requestResponseMustBeModelsContent(call.Root, content, oas.Join(response.Path, "content"))
		}),
	)
}

// requestResponseMustBeModelsContent checks the media types of one content map.
func requestResponseMustBeModelsContent(root, content *yaml.Node, path string) []Finding {
	return seq.FlatMapSlice(oas.MediaTypes(content, path), func(media oas.MediaType) []Finding {
		schema := oas.Get(media.Node, "schema")
		violation := requestResponseMustBeModelsIsJSON(media.Name) && oas.IsMap(schema) &&
			requestResponseMustBeModelsHas(root, schema, requestResponseMustBeModelsIsObject, 0) &&
			!requestResponseMustBeModelsHas(root, schema, requestResponseMustBeModelsDefinesProps, 0)

		return When(violation, func() Finding {
			return Finding{
				Node:    schema,
				Path:    oas.Join(media.Path, "schema"),
				Message: "request and response bodies must be models - their schemas must define `properties`",
			}
		})
	})
}

// requestResponseMustBeModelsIsJSON reports whether the media type uses JSON representation.
func requestResponseMustBeModelsIsJSON(name string) bool {
	base, _, _ := strings.Cut(name, ";")
	folded := Fold(strings.TrimSpace(base))

	return folded == requestResponseMustBeModelsJSON ||
		folded == requestResponseMustBeModelsJSONPatch ||
		folded == requestResponseMustBeModelsMergeJSON
}

// requestResponseMustBeModelsIsObject reports whether a simple schema is an object schema.
func requestResponseMustBeModelsIsObject(schema *yaml.Node) bool {
	if oas.HasType(schema, requestResponseMustBeModelsObject) {
		return true
	}

	additional := oas.Get(schema, "additionalProperties")

	return !oas.Has(schema, "type") && (oas.IsMap(oas.Get(schema, "properties")) ||
		oas.IsMap(oas.Get(schema, "patternProperties")) ||
		oas.IsMap(additional) ||
		oas.IsTrue(additional))
}

// requestResponseMustBeModelsDefinesProps reports whether a simple schema has properties.
func requestResponseMustBeModelsDefinesProps(schema *yaml.Node) bool {
	properties := oas.Get(schema, "properties")

	return oas.IsMap(properties) && !oas.Empty(properties)
}

// requestResponseMustBeModelsHas applies the IBM composition semantics of a predicate.
func requestResponseMustBeModelsHas(root, schema *yaml.Node, predicate func(*yaml.Node) bool, depth int) bool {
	resolved := oas.Resolve(root, schema)
	if depth > requestResponseMustBeModelsMaxDepth || !oas.IsMap(resolved) {
		return false
	}

	allOf := oas.Get(resolved, requestResponseMustBeModelsAllOf)

	return predicate(resolved) ||
		(oas.IsSeq(allOf) && slices.ContainsFunc(allOf.Content, func(member *yaml.Node) bool {
			return requestResponseMustBeModelsHas(root, member, predicate, depth+1)
		})) ||
		slices.ContainsFunc([]string{literalOneof, requestResponseMustBeModelsAnyOf}, func(keyword string) bool {
			return requestResponseMustBeModelsAll(root, oas.Get(resolved, keyword), predicate, depth)
		})
}

// requestResponseMustBeModelsAll reports whether every member satisfies the predicate.
func requestResponseMustBeModelsAll(root, members *yaml.Node, predicate func(*yaml.Node) bool, depth int) bool {
	return oas.IsSeq(members) && len(members.Content) > 0 &&
		!slices.ContainsFunc(members.Content, func(member *yaml.Node) bool {
			return !requestResponseMustBeModelsHas(root, member, predicate, depth+1)
		})
}
