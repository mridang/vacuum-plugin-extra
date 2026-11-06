package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// NoLinkHeader returns the extra-no-link-header rule.
//
// # What it checks
//
// No header parameter and no response header may be named Link. Names are
// compared case-insensitively. Parameters are read from operations, path items
// and components.parameters; response headers from operation responses,
// components.responses and components.headers.
//
// # Why
//
// The Link header (RFC 8288) is meant for hypermedia relations and is awkward
// to consume from JSON based clients, which cannot read it from the payload.
// Links belong into the JSON body, where they are documented by the schema.
//
// # Where it comes from
//
// Zally (AvoidLinkHeadersRule, Zalando guideline 166 "Do Not Use Link Headers
// with JSON entities").
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
//	          headers:
//	            Link:
//	              schema:
//	                type: string
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
//	                  next:
//	                    type: string
//
// # Known false positives / limitations
//
// APIs that return non JSON media types, or that follow a standard such as the
// GitHub style pagination on purpose, may legitimately use the header.
func NoLinkHeader() Rule {
	return New(Spec{
		Name:        "no-link-header",
		Description: "Header parameters and response headers must not be named Link.",
		HowToFix:    "Remove the Link header and return the links in the response body.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkNoLinkHeader),
	})
}

func checkNoLinkHeader(call *Call) []Finding {
	parameters := documentParameters(call.Root)
	headers := seq.Filter(slices.Values(noLinkHeaderResponseHeaders(call.Root)), func(header oas.Node) bool {
		return strings.EqualFold(header.Key, "link")
	})

	return slices.Concat(
		seq.FlatMapIndexed(parameters, func(i int, parameter oas.Parameter) []Finding {
			first := slices.IndexFunc(parameters, func(other oas.Parameter) bool {
				return other.Node == parameter.Node
			})

			return When(
				first == i && parameter.In() == keywordHeader && strings.EqualFold(parameter.Name(), "link"),
				func() Finding { return noLinkHeaderFinding(parameter.Node, parameter.Path, "header parameter") },
			)
		}),
		seq.FlatMapIndexed(headers, func(i int, header oas.Node) []Finding {
			first := slices.IndexFunc(headers, func(other oas.Node) bool {
				return other.Node == header.Node
			})
			declared := slices.ContainsFunc(parameters, func(parameter oas.Parameter) bool {
				return parameter.Node == header.Node
			})

			return When(
				first == i && !declared,
				func() Finding { return noLinkHeaderFinding(header.Node, header.Path, "response header") },
			)
		}),
	)
}

// noLinkHeaderFinding builds the finding for a Link header of the given kind.
func noLinkHeaderFinding(node *yaml.Node, path string, kind string) Finding {
	return Finding{
		Node:    node,
		Path:    path,
		Message: kind + " named Link must not be used, return links in the response body",
	}
}

// noLinkHeaderResponseHeaders lists every response header with its name in Key.
func noLinkHeaderResponseHeaders(root *yaml.Node) []oas.Node {
	collect := func(response *yaml.Node, path string) []oas.Node {
		return seq.FlatMapPairs(
			oas.Pairs(oas.Get(oas.Resolve(root, response), "headers")),
			func(key, value *yaml.Node) []oas.Node {
				return []oas.Node{{
					Key:  key.Value,
					Node: oas.Resolve(root, value),
					Path: oas.JoinAll(path, "headers", key.Value),
				}}
			},
		)
	}

	return slices.Concat(
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []oas.Node {
			return seq.FlatMapSlice(oas.Responses(root, operation), func(response oas.Response) []oas.Node {
				return collect(response.Node, response.Path)
			})
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "responses")),
			func(key, value *yaml.Node) []oas.Node {
				return collect(value, oas.JoinAll("$.components.responses", key.Value))
			},
		),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "headers")),
			func(key, value *yaml.Node) []oas.Node {
				return []oas.Node{{
					Key:  key.Value,
					Node: oas.Resolve(root, value),
					Path: oas.JoinAll("$.components.headers", key.Value),
				}}
			},
		),
	)
}
