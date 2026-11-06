package rules

import (
	"net/url"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	linkTargetExistsOperationRefMethods = "operationRefMethods"
	linkTargetExistsPathsPointer        = "#/paths/"
)

// LinkTargetExists returns the extra-link-target-exists rule.
//
// # What it checks
//
// Every Link object, in operation responses, in components.responses and in
// components.links, must point at an operation that exists in the document:
//
//   - operationId must equal the operationId of some operation of the document.
//   - A relative operationRef (a JSON pointer starting with "#/paths/") must
//     resolve to an existing operation under that path. External operationRef
//     values (URLs and files) are not resolved and are accepted.
//   - A link must define exactly one of operationId and operationRef.
//
// # Why
//
// A link whose target does not exist is dead documentation: generators drop it
// or fail, and consumers follow a relation that cannot be called. Typos in
// operationIds are the usual cause, often after a rename.
//
// # Where it comes from
//
// Merges speakeasy's link-operation-id-resolves and KaiZen's link-target-exists
// (its LinkValidator, which also rejects links with both or neither of the two
// keys).
//
// # Options
//
//   - operationRefMethods (list of strings, default empty): the HTTP methods a
//     relative operationRef may target. KaiZen only accepts get; the default
//     accepts any method, as the OpenAPI specification does.
//
// # Bad
//
//	paths:
//	  /users:
//	    post:
//	      operationId: createUser
//	      responses:
//	        "201":
//	          description: created
//	          links:
//	            fetch:
//	              operationId: getUsr
//
// # Good
//
//	fetch:
//	  operationId: getUser
//
// # Known false positives and limitations
//
// Operations declared only in an external document, or under path item $refs
// that cannot be resolved locally, are not seen, so links to them are flagged
// when given by operationId. Link parameters and requestBody expressions are
// not validated.
func LinkTargetExists() Rule {
	return New(Spec{
		Name:        "link-target-exists",
		Description: "Link objects must reference an operation that exists in the document.",
		HowToFix:    "Point the link at an existing operationId or at a valid relative operationRef.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        linkTargetExistsOperationRefMethods,
				Description: "HTTP methods a relative operationRef may target; empty accepts any method.",
				Default:     []any{},
			},
		},
		Check: OnDocument(checkLinkTargetExists),
	})
}

type linkTargetExistsLink struct {
	node *yaml.Node
	path string
}

// checkLinkTargetExists validates the target of every link.
func checkLinkTargetExists(call *Call) []Finding {
	ids := linkTargetExistsOperationIDs(call.Root)
	methods := call.Strings(linkTargetExistsOperationRefMethods)

	return seq.FlatMapSlice(linkTargetExistsLinks(call.Root), func(link linkTargetExistsLink) []Finding {
		return linkTargetExistsCheck(call.Root, link, ids, methods)
	})
}

// linkTargetExistsCheck validates one link object.
func linkTargetExistsCheck(
	root *yaml.Node,
	link linkTargetExistsLink,
	ids []string,
	methods []string,
) []Finding {
	idKey, idNode := oas.Entry(link.node, "operationId")
	refKey, refNode := oas.Entry(link.node, "operationRef")

	switch {
	case idNode == nil && refNode == nil:
		return []Finding{{Node: link.node, Path: link.path, Message: "link must define operationId or operationRef"}}
	case idNode != nil && refNode != nil:
		return []Finding{
			{Node: link.node, Path: link.path, Message: "link must not define both operationId and operationRef"},
		}
	case idNode != nil && oas.IsScalar(idNode) && !slices.Contains(ids, idNode.Value):
		return []Finding{{
			Node:    idNode,
			Path:    oas.Join(link.path, idKey.Value),
			Message: "link operationId '" + idNode.Value + "' does not match any operation",
		}}
	case refNode != nil && oas.IsScalar(refNode) && !linkTargetExistsRefResolves(root, refNode.Value, methods):
		return []Finding{{
			Node:    refNode,
			Path:    oas.Join(link.path, refKey.Value),
			Message: "link operationRef '" + refNode.Value + "' does not resolve to an operation",
		}}
	default:
		return nil
	}
}

// linkTargetExistsOperationIDs returns the operationIds of the document.
func linkTargetExistsOperationIDs(root *yaml.Node) []string {
	return seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []string {
		id := oas.StringAt(operation.Node, "operationId")
		if id == "" {
			return nil
		}

		return []string{id}
	})
}

// linkTargetExistsRefResolves reports whether a relative operationRef points at an operation.
func linkTargetExistsRefResolves(root *yaml.Node, ref string, methods []string) bool {
	pointer, relative := strings.CutPrefix(ref, linkTargetExistsPathsPointer)
	if !relative {
		return !strings.HasPrefix(ref, "#")
	}

	route, rawMethod, found := strings.CutLast(pointer, "/")
	if !found {
		return false
	}

	method := strings.ToLower(rawMethod)
	item := oas.Resolve(root, oas.Get(oas.Get(root, "paths"), linkTargetExistsRoute(route)))

	if !oas.IsMethod(method) || !oas.IsMap(oas.Get(item, method)) {
		return false
	}

	return len(methods) == 0 || slices.Contains(methods, method)
}

// linkTargetExistsRoute decodes the percent and JSON pointer escapes of a route.
func linkTargetExistsRoute(escaped string) string {
	decoded, err := url.PathUnescape(escaped)
	if err != nil {
		return linkTargetExistsPointer(escaped)
	}

	return linkTargetExistsPointer(decoded)
}

// linkTargetExistsPointer decodes the JSON pointer escapes of a route.
func linkTargetExistsPointer(route string) string {
	return strings.ReplaceAll(strings.ReplaceAll(route, "~1", "/"), "~0", "~")
}

// linkTargetExistsLinks collects the inline link objects of the document.
func linkTargetExistsLinks(root *yaml.Node) []linkTargetExistsLink {
	return slices.Concat(
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []linkTargetExistsLink {
			return seq.FlatMapPairs(
				oas.Pairs(oas.Get(operation.Node, "responses")),
				func(code, response *yaml.Node) []linkTargetExistsLink {
					path := oas.JoinAll(operation.Path, "responses", code.Value)

					return linkTargetExistsIn(oas.Get(response, "links"), oas.Join(path, "links"))
				},
			)
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "responses")),
			func(name, response *yaml.Node) []linkTargetExistsLink {
				path := oas.JoinAll("$.components.responses", name.Value)

				return linkTargetExistsIn(oas.Get(response, "links"), oas.Join(path, "links"))
			},
		),
		linkTargetExistsIn(oas.Dig(root, "components", "links"), "$.components.links"),
	)
}

// linkTargetExistsIn returns the inline link objects of a links map.
func linkTargetExistsIn(links *yaml.Node, path string) []linkTargetExistsLink {
	return seq.FlatMapPairs(oas.Pairs(links), func(name, link *yaml.Node) []linkTargetExistsLink {
		if !oas.IsMap(link) || oas.Has(link, "$ref") {
			return nil
		}

		return []linkTargetExistsLink{{node: link, path: oas.Join(path, name.Value)}}
	})
}
