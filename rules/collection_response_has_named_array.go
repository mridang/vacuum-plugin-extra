package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const collectionResponseHasNamedArrayGet = "get"

const collectionResponseHasNamedArrayPrefix = "operationIdPrefix"

// CollectionResponseHasNamedArray returns the extra-collection-response-has-named-array rule.
//
// # What it checks
//
// For every list operation, each JSON success (2xx) response schema must define
// an array property named after the last segment of the path. A GET is a list
// operation when its operationId starts with the configured prefix (default
// "list"), or when the last path segment is not a path parameter and a sibling
// path of the form {route}/{param} exists. Properties of allOf members count,
// and $ref schemas are resolved. Binary (file download) responses are skipped.
//
// # Why
//
// Wrapping a collection in an object with a predictably named array lets the
// API grow pagination metadata without breaking clients, and gives generated
// SDKs a consistent way to reach the items: GET /things returns an object with
// a "things" array.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-collection-array-property).
//
// # Options
//
//   - operationIdPrefix (string, default "list"): operationIds starting with
//     this text mark a GET as a list operation. Set it to an empty string to
//     rely on the path heuristic alone.
//
// # Bad
//
//	paths:
//	  /things:
//	    get:
//	      operationId: list_things
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              schema:
//	                type: object
//	                properties:
//	                  results:
//	                    type: array
//	                    items: {type: string}
//
// # Good
//
//	schema:
//	  type: object
//	  properties:
//	    things:
//	      type: array
//	      items: {type: string}
//
// # Known false positives and limitations
//
// The path heuristic treats any GET whose path has a sibling item path as a
// list, so an unusual singleton route next to an item route is flagged. A
// top-level array response is flagged because it has no named property.
func CollectionResponseHasNamedArray() Rule {
	return New(Spec{
		Name:        "collection-response-has-named-array",
		Description: "List operation responses must contain an array property named after the last path segment.",
		HowToFix:    "Wrap the collection in an object whose array property is named after the last path segment.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        collectionResponseHasNamedArrayPrefix,
				Description: "operationId prefix that marks a GET as a list operation.",
				Default:     keywordList,
			},
		},
		Check: PerOperation(checkCollectionResponseHasNamedArray),
	})
}

// checkCollectionResponseHasNamedArray inspects the success responses of one list operation.
func checkCollectionResponseHasNamedArray(call *Call, operation oas.Operation) []Finding {
	prefix := call.String(collectionResponseHasNamedArrayPrefix)
	if operation.Method != collectionResponseHasNamedArrayGet ||
		!collectionResponseHasNamedArrayIsList(call.Root, operation, prefix) {
		return nil
	}

	name := operation.Route[strings.LastIndex(operation.Route, "/")+1:]

	return collectionResponseHasNamedArrayOperation(call.Root, operation, name)
}

// collectionResponseHasNamedArrayIsList reports whether the operation lists a collection.
func collectionResponseHasNamedArrayIsList(root *yaml.Node, operation oas.Operation, prefix string) bool {
	operationID := oas.StringAt(operation.Node, "operationId")
	if prefix != "" && strings.HasPrefix(operationID, prefix) {
		return true
	}

	last := operation.Route[strings.LastIndex(operation.Route, "/")+1:]
	if strings.ContainsAny(last, "{}") {
		return false
	}

	for item := range oas.PathItems(root) {
		rest, found := strings.CutPrefix(item.Route, operation.Route+"/")
		if found && strings.HasPrefix(rest, "{") && strings.HasSuffix(rest, "}") && !strings.Contains(rest, "/") {
			return true
		}
	}

	return false
}

// collectionResponseHasNamedArrayOperation checks the 2xx JSON responses of one operation.
func collectionResponseHasNamedArrayOperation(root *yaml.Node, operation oas.Operation, name string) []Finding {
	return seq.FlatMapSlice(oas.Responses(root, operation), func(response oas.Response) []Finding {
		if !oas.IsSuccess(response.Code) {
			return nil
		}

		mediaTypes := oas.MediaTypes(oas.Get(response.Node, "content"), oas.Join(response.Path, "content"))

		return seq.FlatMapSlice(mediaTypes, func(media oas.MediaType) []Finding {
			schema := oas.Resolve(root, oas.Get(media.Node, "schema"))

			return When(
				oas.IsMap(schema) && strings.Contains(strings.ToLower(media.Name), "json") &&
					!collectionResponseHasNamedArrayBinary(schema) &&
					!collectionResponseHasNamedArrayFound(root, schema, name, 0),
				func() Finding {
					return Finding{
						Node:    schema,
						Path:    oas.Join(media.Path, "schema"),
						Message: "collection list operation response schema should define array property with name: " + name,
					}
				},
			)
		})
	})
}

// collectionResponseHasNamedArrayBinary reports whether the schema is a binary string.
func collectionResponseHasNamedArrayBinary(schema *yaml.Node) bool {
	return oas.HasType(schema, "string") && oas.StringAt(schema, "format") == "binary"
}

const collectionResponseHasNamedArrayMaxDepth = 8

// collectionResponseHasNamedArrayFound looks for the array property in the schema and its allOf members.
func collectionResponseHasNamedArrayFound(root, schema *yaml.Node, name string, depth int) bool {
	if depth > collectionResponseHasNamedArrayMaxDepth {
		return false
	}

	property := oas.Resolve(root, oas.Dig(schema, "properties", name))
	if oas.HasType(property, "array") {
		return true
	}

	for _, member := range oas.Items(oas.Get(schema, "allOf")) {
		if collectionResponseHasNamedArrayFound(root, oas.Resolve(root, member), name, depth+1) {
			return true
		}
	}

	return false
}
