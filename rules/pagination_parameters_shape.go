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
	paginationParametersShapeGet      = "get"
	paginationParametersShapeMaxDepth = 16
	paginationParametersShapeLimit    = "limit"
	paginationParametersShapeOffset   = "offset"
	paginationParametersShapeStart    = "start"
	paginationParametersShapeTotal    = "total_count"
	paginationParametersShapeHref     = "href"
	paginationParametersShapeInteger  = "integer"
	paginationParametersShapeString   = "string"
	paginationParametersShapeArray    = "array"
	paginationParametersShapeProps    = "properties"
	paginationParametersShapeAllOf    = "allOf"
	paginationParametersShapeAnyOf    = "anyOf"
	paginationParametersShapeJSON     = "application/json"
	paginationParametersShapeSchemas  = "schema"
	paginationParametersShapeFalse    = "false"
)

// paginationParametersShapeTokenNames lists the names recognised as page token parameters.
func paginationParametersShapeTokenNames() []string {
	return []string{paginationParametersShapeStart, "token", "cursor", "page", "page_token"}
}

// PaginationParametersShape returns the extra-pagination-parameters-shape rule.
//
// # What it checks
//
// A GET operation is a paginated list operation when all of these hold: its
// path does not end in a path parameter; its first success response (the first
// response code that starts with 2) has an application/json media type whose
// schema, with allOf members merged, has properties and at least one of them is
// an array schema; and the operation itself declares parameters including an
// offset query parameter or one page token query parameter named start, token,
// cursor, page or page_token. Parameters inherited from the path item are not
// considered, as in the original. For such an operation:
//
//   - A limit query parameter must be an integer schema, optional, and have a
//     default and a maximum that are both truthy (a default or maximum of 0
//     counts as missing, exactly like the original).
//   - An offset query parameter must be an integer schema and optional.
//   - An offset parameter requires a limit parameter (reported on the
//     operation).
//   - The page token parameter (the first matching one) should be named start,
//     and must be a string schema and optional.
//   - When there is a limit parameter, the response schema must have a limit
//     property that is an integer and is listed in required; likewise offset
//     for an offset parameter.
//   - A total_count response property, when present, must be an integer and
//     listed in required.
//   - The response must have first and next properties, and may have last and
//     previous; each present one must be an object with an href string
//     property (allOf merged).
//
// Schema checks use the IBM composition semantics: a type test holds for a
// schema when the schema, any allOf member, all members of a oneOf, or all
// members of an anyOf satisfies it. required lists of all allOf members are
// merged.
//
// # Why
//
// Pagination parameters are used by every client of every list endpoint.
// Letting limit be required, unbounded or untyped, or letting the response omit
// the values that were applied and the links to other pages, forces each client
// to special-case each endpoint.
//
// # Where it comes from
//
// IBM openapi-validator, ibm-pagination-style (checks 1 to 12; check 7 lives in
// the separate collection-array-property rule).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /things:
//	    get:
//	      parameters:
//	        - {name: limit, in: query, required: true, schema: {type: integer}}
//	        - {name: offset, in: query, schema: {type: integer}}
//
// # Good
//
//	paths:
//	  /things:
//	    get:
//	      parameters:
//	        - {name: limit, in: query, schema: {type: integer, default: 10, maximum: 100}}
//	        - {name: start, in: query, schema: {type: string}}
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              schema:
//	                type: object
//	                required: [limit, first, next, things]
//	                properties:
//	                  limit: {type: integer}
//	                  first: {type: object, properties: {href: {type: string}}}
//	                  next: {type: object, properties: {href: {type: string}}}
//	                  things: {type: array, items: {type: string}}
//
// # Known false positives and limitations
//
// Local references are followed; external ones are not. Where the original
// reports every response property check at a fixed application/json path that
// Spectral remaps for allOf, this rule reports the location where the property
// (or the response schema for a missing property) is actually written.
func PaginationParametersShape() Rule {
	return New(Spec{
		Name:        "pagination-parameters-shape",
		Description: "Paginated list operations must define limit, offset or page token parameters, response properties and page links consistently.",
		HowToFix:    "Make limit an optional integer with default and maximum, offset an optional integer, the page token an optional string named start, and echo them with page links in the response.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkPaginationParametersShape),
	})
}

// paginationParametersShapeMember is one schema of the allOf merge of a response schema.
type paginationParametersShapeMember struct {
	node *yaml.Node
	path string
}

// paginationParametersShapeState carries the document.
type paginationParametersShapeState struct {
	root *yaml.Node
}

// paginationParametersShapeList describes a recognised paginated list operation.
type paginationParametersShapeList struct {
	operation  oas.Operation
	params     []oas.Parameter
	members    []paginationParametersShapeMember
	schemaPath string
}

// paginationParametersShapeLink is one of the page links of a paginated response.
type paginationParametersShapeLink struct {
	name     string
	required bool
}

// checkPaginationParametersShape inspects every candidate paginated list operation.
func checkPaginationParametersShape(call *Call) []Finding {
	state := &paginationParametersShapeState{root: call.Root}

	return seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
		if operation.Method != paginationParametersShapeGet || strings.HasSuffix(operation.Route, "}") {
			return nil
		}

		list, ok := state.recognise(operation)
		if !ok {
			return nil
		}

		return state.check(list)
	})
}

// paginationParametersShapeIf returns the findings of build when condition holds, and none otherwise.
func paginationParametersShapeIf(condition bool, build func() []Finding) []Finding {
	if !condition {
		return nil
	}

	return build()
}

// recognise decides whether the operation is a paginated list operation.
func (s *paginationParametersShapeState) recognise(operation oas.Operation) (paginationParametersShapeList, bool) {
	none := paginationParametersShapeList{operation: operation}

	successes := seq.Filter(slices.Values(oas.Responses(s.root, operation)), func(response oas.Response) bool {
		return strings.HasPrefix(response.Code, "2")
	})
	if len(successes) == 0 {
		return none, false
	}

	schema, path := s.jsonSchema(successes[0])
	if schema == nil {
		return none, false
	}

	list := paginationParametersShapeList{
		operation:  operation,
		params:     oas.OwnParameters(s.root, operation),
		members:    s.members(schema, path, 0),
		schemaPath: path,
	}
	if list.members == nil || !s.hasArrayProperty(list) {
		return none, false
	}

	return list, paginationParametersShapeFind(list.params, paginationParametersShapeOffset) != nil ||
		paginationParametersShapeToken(list.params) != nil
}

// jsonSchema returns the schema of the first application/json media type of a response.
func (s *paginationParametersShapeState) jsonSchema(response oas.Response) (*yaml.Node, string) {
	content := oas.Get(response.Node, "content")

	for _, media := range oas.MediaTypes(content, oas.Join(response.Path, "content")) {
		base, _, _ := strings.Cut(media.Name, ";")
		if Fold(strings.TrimSpace(base)) != paginationParametersShapeJSON {
			continue
		}

		schema := oas.Get(media.Node, paginationParametersShapeSchemas)
		if schema == nil {
			return nil, ""
		}

		return schema, oas.Join(media.Path, paginationParametersShapeSchemas)
	}

	return nil, ""
}

// paginationParametersShapeRefPath turns a local reference into the JSONPath of its target.
func paginationParametersShapeRefPath(ref string) string {
	pointer, ok := strings.CutPrefix(ref, "#/")
	if !ok {
		return ""
	}

	segments := seq.Map(slices.Values(strings.Split(pointer, "/")), func(segment string) string {
		return strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
	})

	return oas.JoinAll("$", segments...)
}

// memberPath returns the path a schema is reported at, following a local reference to its target.
func (s *paginationParametersShapeState) memberPath(schema, resolved *yaml.Node, path string) string {
	ref := oas.Ref(schema)
	if ref == "" || resolved == schema {
		return path
	}

	if target := paginationParametersShapeRefPath(ref); target != "" {
		return target
	}

	return path
}

// members returns the allOf-merged schemas of a schema; later members override earlier ones.
func (s *paginationParametersShapeState) members(
	schema *yaml.Node,
	path string,
	depth int,
) []paginationParametersShapeMember {
	resolved := oas.Resolve(s.root, schema)
	if depth > paginationParametersShapeMaxDepth || !oas.IsMap(resolved) {
		return nil
	}

	own := s.memberPath(schema, resolved, path)

	return slices.Concat(
		seq.FlatMapPairs(
			oas.Items(oas.Get(resolved, paginationParametersShapeAllOf)),
			func(i int, member *yaml.Node) []paginationParametersShapeMember {
				return s.members(member, oas.Index(oas.Join(own, paginationParametersShapeAllOf), i), depth+1)
			},
		),
		[]paginationParametersShapeMember{{node: resolved, path: own}},
	)
}

// property finds a property of the merged schema and the JSONPath where it is written.
func (s *paginationParametersShapeState) property(
	members []paginationParametersShapeMember,
	name string,
) (*yaml.Node, string) {
	for _, member := range slices.Backward(members) {
		if property := oas.Get(oas.Get(member.node, paginationParametersShapeProps), name); property != nil {
			return property, oas.JoinAll(member.path, paginationParametersShapeProps, name)
		}
	}

	return nil, ""
}

// required reports whether the merged required lists contain the name.
func paginationParametersShapeRequired(members []paginationParametersShapeMember, name string) bool {
	return slices.ContainsFunc(members, func(member paginationParametersShapeMember) bool {
		return slices.Contains(oas.Strings(oas.Get(member.node, "required")), name)
	})
}

// hasArrayProperty reports whether the merged schema has properties with an array one.
func (s *paginationParametersShapeState) hasArrayProperty(list paginationParametersShapeList) bool {
	for _, member := range list.members {
		for _, property := range oas.Pairs(oas.Get(member.node, paginationParametersShapeProps)) {
			if s.typed(property, paginationParametersShapeArray) {
				return true
			}
		}
	}

	return false
}

// check runs the pagination style checks for one recognised operation.
func (s *paginationParametersShapeState) check(list paginationParametersShapeList) []Finding {
	limit := paginationParametersShapeFind(list.params, paginationParametersShapeLimit)
	offset := paginationParametersShapeFind(list.params, paginationParametersShapeOffset)
	token := paginationParametersShapeToken(list.params)

	return slices.Concat(
		s.checkLimit(limit),
		s.checkOffset(offset),
		When(offset != nil && limit == nil, func() Finding {
			return Finding{
				Node: list.operation.Node,
				Path: list.operation.Path,
				Message: "the operation must define a 'limit' query parameter " +
					"if the 'offset' query parameter is defined",
			}
		}),
		s.checkToken(token),
		s.checkResponse(list, limit != nil, offset != nil),
	)
}

// paginationParametersShapeFind returns the first query parameter with the exact name.
func paginationParametersShapeFind(params []oas.Parameter, name string) *oas.Parameter {
	for i := range params {
		if params[i].Name() == name && params[i].In() == literalQuery {
			return &params[i]
		}
	}

	return nil
}

// paginationParametersShapeToken returns the first query parameter that looks like a page token.
func paginationParametersShapeToken(params []oas.Parameter) *oas.Parameter {
	names := paginationParametersShapeTokenNames()

	for i := range params {
		if params[i].In() == literalQuery && slices.Contains(names, params[i].Name()) {
			return &params[i]
		}
	}

	return nil
}

// paginationParametersShapeTruthy mirrors JavaScript truthiness for scalar keyword values.
func paginationParametersShapeTruthy(n *yaml.Node) bool {
	if n == nil || oas.IsNull(n) {
		return false
	}

	if !oas.IsScalar(n) {
		return true
	}

	return !slices.Contains([]string{"", "0", paginationParametersShapeFalse}, oas.Text(n))
}

// checkLimit checks the limit parameter, when there is one.
func (s *paginationParametersShapeState) checkLimit(parameter *oas.Parameter) []Finding {
	if parameter == nil {
		return nil
	}

	schema := oas.Get(parameter.Node, paginationParametersShapeSchemas)
	resolved := oas.Resolve(s.root, schema)

	return When(
		schema == nil || !s.typed(schema, paginationParametersShapeInteger) || parameter.Required() ||
			!paginationParametersShapeTruthy(oas.Get(resolved, "default")) ||
			!paginationParametersShapeTruthy(oas.Get(resolved, "maximum")),
		func() Finding {
			return Finding{
				Node:    parameter.Node,
				Path:    parameter.Path,
				Message: "the 'limit' parameter must be of type integer and optional with default and maximum values",
			}
		},
	)
}

// checkOffset checks the offset parameter, when there is one.
func (s *paginationParametersShapeState) checkOffset(parameter *oas.Parameter) []Finding {
	if parameter == nil {
		return nil
	}

	schema := oas.Get(parameter.Node, paginationParametersShapeSchemas)

	return When(
		schema == nil || !s.typed(schema, paginationParametersShapeInteger) || parameter.Required(),
		func() Finding {
			return Finding{
				Node:    parameter.Node,
				Path:    parameter.Path,
				Message: "the 'offset' parameter must be of type integer and optional",
			}
		},
	)
}

// checkToken checks the page token parameter, when there is one.
func (s *paginationParametersShapeState) checkToken(parameter *oas.Parameter) []Finding {
	if parameter == nil {
		return nil
	}

	schema := oas.Get(parameter.Node, paginationParametersShapeSchemas)

	return slices.Concat(
		When(parameter.Name() != paginationParametersShapeStart, func() Finding {
			return Finding{
				Node:    parameter.Node,
				Path:    parameter.Path,
				Message: "the '" + parameter.Name() + "' parameter should be named 'start'",
			}
		}),
		When(
			schema == nil || !s.typed(schema, paginationParametersShapeString) || parameter.Required(),
			func() Finding {
				return Finding{
					Node:    parameter.Node,
					Path:    parameter.Path,
					Message: "the '" + parameter.Name() + "' parameter must be of type string and optional",
				}
			},
		),
	)
}

// checkResponse checks the response properties of a paginated list.
func (s *paginationParametersShapeState) checkResponse(
	list paginationParametersShapeList,
	hasLimit, hasOffset bool,
) []Finding {
	total, _ := s.property(list.members, paginationParametersShapeTotal)

	return slices.Concat(
		paginationParametersShapeIf(hasLimit, func() []Finding {
			return s.checkCount(
				list,
				paginationParametersShapeLimit,
				"a paginated list operation with a 'limit' query "+
					"parameter must include a 'limit' property in the response body schema",
			)
		}),
		paginationParametersShapeIf(hasOffset, func() []Finding {
			return s.checkCount(
				list,
				paginationParametersShapeOffset,
				"a paginated list operation with an 'offset' query "+
					"parameter must include an 'offset' property in the response body schema",
			)
		}),
		paginationParametersShapeIf(total != nil, func() []Finding {
			return s.checkCount(list, paginationParametersShapeTotal, "")
		}),
		seq.FlatMapSlice(
			[]paginationParametersShapeLink{{"first", true}, {"last", false}, {"previous", false}, {"next", true}},
			func(link paginationParametersShapeLink) []Finding {
				return s.checkLink(list, link.name, link.required)
			},
		),
	)
}

// checkCount checks a response property that must be a required integer.
func (s *paginationParametersShapeState) checkCount(
	list paginationParametersShapeList,
	name, missing string,
) []Finding {
	property, path := s.property(list.members, name)
	if property == nil {
		return One(Finding{Path: list.schemaPath, Message: missing})
	}

	return When(
		!s.typed(property, paginationParametersShapeInteger) || !paginationParametersShapeRequired(list.members, name),
		func() Finding {
			return Finding{
				Node: property,
				Path: path,
				Message: "the '" + name + "' property in the response body of a paginated list operation " +
					"must be of type integer and required",
			}
		},
	)
}

// checkLink checks one of the first, last, previous and next page links.
func (s *paginationParametersShapeState) checkLink(
	list paginationParametersShapeList,
	name string,
	required bool,
) []Finding {
	property, path := s.property(list.members, name)
	if property == nil {
		return When(required, func() Finding {
			return Finding{
				Path:    list.schemaPath,
				Message: "a paginated list operation should include a '" + name + "' property in the response body schema",
			}
		})
	}

	href, _ := s.property(s.members(property, path, 0), paginationParametersShapeHref)

	return When(href == nil || !s.typed(href, paginationParametersShapeString), func() Finding {
		return Finding{
			Node:    property,
			Path:    path,
			Message: "the '" + name + "' property should be an object with an 'href' string property",
		}
	})
}

// typed applies the IBM schemaHasConstraint semantics for a schema type.
func (s *paginationParametersShapeState) typed(schema *yaml.Node, kind string) bool {
	return s.has(schema, kind, 0)
}

// has reports whether the schema, any allOf member, all oneOf or all anyOf members have the type.
func (s *paginationParametersShapeState) has(node *yaml.Node, kind string, depth int) bool {
	schema := oas.Resolve(s.root, node)
	if depth > paginationParametersShapeMaxDepth || !oas.IsMap(schema) {
		return false
	}

	if oas.HasType(schema, kind) || (kind == paginationParametersShapeArray &&
		oas.IsMap(oas.Get(schema, "items")) && !oas.Has(schema, "type")) {
		return true
	}

	for _, member := range oas.Items(oas.Get(schema, paginationParametersShapeAllOf)) {
		if s.has(member, kind, depth+1) {
			return true
		}
	}

	return s.all(oas.Get(schema, literalOneof), kind, depth) ||
		s.all(oas.Get(schema, paginationParametersShapeAnyOf), kind, depth)
}

// all reports whether a non-empty list has only members of the type.
func (s *paginationParametersShapeState) all(members *yaml.Node, kind string, depth int) bool {
	if !oas.IsSeq(members) || len(members.Content) == 0 {
		return false
	}

	return !slices.ContainsFunc(members.Content, func(member *yaml.Node) bool {
		return !s.has(member, kind, depth+1)
	})
}
