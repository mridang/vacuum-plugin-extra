package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// Names and limits used by this rule.
const (
	preferTokenPaginationOverOffsetMaxDepth = 16
	preferTokenPaginationOverOffsetOffset   = "offset"
	preferTokenPaginationOverOffsetJSON     = "application/json"
	preferTokenPaginationOverOffsetAllOf    = "allOf"
	preferTokenPaginationOverOffsetAnyOf    = "anyOf"
	preferTokenPaginationOverOffsetArray    = "array"
	preferTokenPaginationOverOffsetItems    = "items"
)

// PreferTokenPaginationOverOffset returns the extra-prefer-token-pagination-over-offset rule.
//
// # What it checks
//
// A GET operation on a path that does not end in a path parameter is a
// paginated list operation when its first success response (the first response
// code that starts with 2) has an application/json media type whose schema,
// with allOf members merged, has properties of which at least one is an array
// schema, and the operation itself declares parameters that include a query
// parameter named offset or a page token query parameter named start, token,
// cursor, page or page_token. Parameters inherited from the path item are not
// considered, as in the original. Such an operation that declares a query
// parameter named exactly offset is reported, at the operation, with the
// message that token-based pagination is recommended over offset/limit
// pagination.
//
// # Why
//
// Offset pagination skips rows by count. Inserts and deletes between two calls
// make clients see duplicates or miss items, and large offsets are slow on most
// databases. Token (cursor) pagination, using a start parameter, stays stable
// and cheap.
//
// # Where it comes from
//
// IBM openapi-validator, ibm-prefer-token-pagination.
//
// # Options
//
// None. The names offset and the page token names are fixed, as in the
// original.
//
// # Bad
//
//	paths:
//	  /things:
//	    get:
//	      parameters:
//	        - {name: limit, in: query, schema: {type: integer}}
//	        - {name: offset, in: query, schema: {type: integer}}
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              schema:
//	                type: object
//	                properties:
//	                  things: {type: array, items: {type: string}}
//
// # Good
//
//	parameters:
//	  - {name: limit, in: query, schema: {type: integer}}
//	  - {name: start, in: query, schema: {type: string}}
//
// # Known false positives and limitations
//
// Datasets that are static and indexed, where offsets are cheap and stable, are
// flagged anyway. An offset parameter on an operation that is not recognised as
// a paginated list (no array property in the first success response, or a
// path ending in a parameter) is not reported. Only local references are
// followed.
func PreferTokenPaginationOverOffset() Rule {
	return New(Spec{
		Name:        "prefer-token-pagination-over-offset",
		Description: "Paginated list operations should use token-based pagination, rather than offset/limit pagination.",
		HowToFix:    "Replace the offset parameter with a start page token parameter and return the next page link.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkPreferTokenPaginationOverOffset),
	})
}

// checkPreferTokenPaginationOverOffset reports paginated list operations that use offset.
func checkPreferTokenPaginationOverOffset(call *Call, operation oas.Operation) []Finding {
	if operation.Method != methodGet || strings.HasSuffix(operation.Route, "}") {
		return nil
	}

	hasOffset := slices.ContainsFunc(
		oas.OwnParameters(call.Root, operation),
		preferTokenPaginationOverOffsetIsQuery(preferTokenPaginationOverOffsetOffset),
	)

	return When(hasOffset && preferTokenPaginationOverOffsetListBody(call.Root, operation), func() Finding {
		return Finding{
			Node:    operation.Node,
			Path:    operation.Path,
			Message: "token-based pagination is recommended over offset/limit pagination",
		}
	})
}

func preferTokenPaginationOverOffsetIsQuery(name string) func(oas.Parameter) bool {
	return func(parameter oas.Parameter) bool {
		return parameter.In() == literalQuery && parameter.Name() == name
	}
}

// preferTokenPaginationOverOffsetListBody reports whether the first success response is a page of an array.
func preferTokenPaginationOverOffsetListBody(root *yaml.Node, operation oas.Operation) bool {
	responses := oas.Responses(root, operation)
	first := slices.IndexFunc(responses, func(response oas.Response) bool {
		return strings.HasPrefix(response.Code, "2")
	})
	if first < 0 {
		return false
	}

	schema := preferTokenPaginationOverOffsetSchema(responses[first])

	return schema != nil && preferTokenPaginationOverOffsetHasArray(root, schema)
}

// preferTokenPaginationOverOffsetSchema returns the schema of the first application/json media type.
func preferTokenPaginationOverOffsetSchema(response oas.Response) *yaml.Node {
	for name, media := range oas.Pairs(oas.Get(response.Node, "content")) {
		base, _, _ := strings.Cut(name.Value, ";")
		if Fold(strings.TrimSpace(base)) == preferTokenPaginationOverOffsetJSON {
			return oas.Get(media, "schema")
		}
	}

	return nil
}

// preferTokenPaginationOverOffsetHasArray reports whether the merged schema has an array property.
func preferTokenPaginationOverOffsetHasArray(root, schema *yaml.Node) bool {
	for _, member := range preferTokenPaginationOverOffsetMembers(root, schema, 0) {
		for _, property := range oas.Pairs(oas.Get(member, "properties")) {
			if preferTokenPaginationOverOffsetIsArray(root, property, 0) {
				return true
			}
		}
	}

	return false
}

// preferTokenPaginationOverOffsetMembers returns a schema together with its allOf members, resolved.
func preferTokenPaginationOverOffsetMembers(root, schema *yaml.Node, depth int) []*yaml.Node {
	resolved := oas.Resolve(root, schema)
	if depth > preferTokenPaginationOverOffsetMaxDepth || !oas.IsMap(resolved) {
		return nil
	}

	return slices.Concat(
		[]*yaml.Node{resolved},
		seq.FlatMapPairs(
			oas.Items(oas.Get(resolved, preferTokenPaginationOverOffsetAllOf)),
			func(_ int, member *yaml.Node) []*yaml.Node {
				return preferTokenPaginationOverOffsetMembers(root, member, depth+1)
			},
		),
	)
}

// preferTokenPaginationOverOffsetIsArray applies the IBM schemaHasConstraint semantics for arrays.
func preferTokenPaginationOverOffsetIsArray(root, node *yaml.Node, depth int) bool {
	schema := oas.Resolve(root, node)
	if depth > preferTokenPaginationOverOffsetMaxDepth || !oas.IsMap(schema) {
		return false
	}

	if oas.HasType(schema, preferTokenPaginationOverOffsetArray) ||
		(oas.IsMap(oas.Get(schema, preferTokenPaginationOverOffsetItems)) && !oas.Has(schema, "type")) {
		return true
	}

	for _, member := range oas.Items(oas.Get(schema, preferTokenPaginationOverOffsetAllOf)) {
		if preferTokenPaginationOverOffsetIsArray(root, member, depth+1) {
			return true
		}
	}

	return preferTokenPaginationOverOffsetAll(root, oas.Get(schema, literalOneof), depth) ||
		preferTokenPaginationOverOffsetAll(root, oas.Get(schema, preferTokenPaginationOverOffsetAnyOf), depth)
}

// preferTokenPaginationOverOffsetAll reports whether a non-empty list has only array members.
func preferTokenPaginationOverOffsetAll(root, members *yaml.Node, depth int) bool {
	if !oas.IsSeq(members) || len(members.Content) == 0 {
		return false
	}

	return !slices.ContainsFunc(members.Content, func(member *yaml.Node) bool {
		return !preferTokenPaginationOverOffsetIsArray(root, member, depth+1)
	})
}
