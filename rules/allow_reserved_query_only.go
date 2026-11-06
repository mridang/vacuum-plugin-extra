package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	// allowReservedQueryOnlyQuery is the only location allowReserved applies to.
	allowReservedQueryOnlyQuery = "query"
	// allowReservedQueryOnlyHeader is the location of header objects.
	allowReservedQueryOnlyHeader = "header"
)

// AllowReservedQueryOnly returns the extra-allow-reserved-query-only rule.
//
// # What it checks
//
// The allowReserved keyword of a parameter object is only meaningful for query
// parameters. The rule flags allowReserved: true on path, header and cookie
// parameters and on header objects (response headers and components.headers).
//
// # Why
//
// The specification defines allowReserved as "determines whether the parameter
// value SHOULD allow reserved characters ... to be included without
// percent-encoding. This property only applies to parameters with an in value
// of query." Setting it elsewhere is silently ignored by tooling, so it hides a
// mistake in the description.
//
// # Where it comes from
//
// The KaiZen OpenAPI parser (allowReserved is set only on query parameters,
// also warning on headers).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets/{id}:
//	    get:
//	      parameters:
//	        - name: id
//	          in: path
//	          required: true
//	          allowReserved: true
//	          schema:
//	            type: string
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: filter
//	          in: query
//	          allowReserved: true
//	          schema:
//	            type: string
//
// # Known false positives / limitations
//
// Only parameter and header objects are inspected. The allowReserved keyword of
// an encoding object (multipart and form bodies) is not checked. Parameters
// reached only through an external $ref are not followed.
func AllowReservedQueryOnly() Rule {
	return New(Spec{
		Name:        "allow-reserved-query-only",
		Description: "allowReserved must only be set on query parameters.",
		HowToFix:    "Remove allowReserved from the parameter or header, or move the parameter to the query string.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkAllowReservedQueryOnly),
	})
}

// allowReservedQueryOnlyTarget is a parameter-like object and its location.

type allowReservedQueryOnlyTarget struct {
	node     *yaml.Node
	path     string
	location string
}

func checkAllowReservedQueryOnly(call *Call) []Finding {
	return seq.FlatMapSlice(
		allowReservedQueryOnlyTargets(call.Root),
		func(target allowReservedQueryOnlyTarget) []Finding {
			key, value := oas.Entry(target.node, "allowReserved")

			return When(oas.IsTrue(value) && target.location != allowReservedQueryOnlyQuery, func() Finding {
				return Finding{
					Node:    value,
					Path:    oas.Join(target.path, key.Value),
					Message: "allowReserved only applies to query parameters, not to " + target.location,
				}
			})
		},
	)
}

func allowReservedQueryOnlyTargets(root *yaml.Node) []allowReservedQueryOnlyTarget {
	return slices.Concat(
		seq.FlatMap(oas.PathItems(root), func(item oas.PathItem) []allowReservedQueryOnlyTarget {
			list := oas.Get(item.Node, "parameters")

			return allowReservedQueryOnlyList(root, list, oas.Join(item.Path, "parameters"))
		}),
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []allowReservedQueryOnlyTarget {
			list := oas.Get(operation.Node, "parameters")

			return slices.Concat(
				allowReservedQueryOnlyList(root, list, oas.Join(operation.Path, "parameters")),
				seq.FlatMapSlice(
					oas.Responses(root, operation),
					func(response oas.Response) []allowReservedQueryOnlyTarget {
						return allowReservedQueryOnlyHeaders(root, response.Node, response.Path)
					},
				),
			)
		}),
		allowReservedQueryOnlyComponents(root),
	)
}

func allowReservedQueryOnlyComponents(root *yaml.Node) []allowReservedQueryOnlyTarget {
	components := oas.Get(root, "components")

	return slices.Concat(
		seq.FlatMapPairs(
			oas.Pairs(oas.Get(components, "parameters")),
			func(key, value *yaml.Node) []allowReservedQueryOnlyTarget {
				path := oas.JoinAll("$.components.parameters", key.Value)

				return []allowReservedQueryOnlyTarget{allowReservedQueryOnlyOne(root, value, path, false)}
			},
		),
		seq.FlatMapPairs(
			oas.Pairs(oas.Get(components, "responses")),
			func(key, value *yaml.Node) []allowReservedQueryOnlyTarget {
				path := oas.JoinAll("$.components.responses", key.Value)

				return allowReservedQueryOnlyHeaders(root, oas.Resolve(root, value), path)
			},
		),
		seq.FlatMapPairs(
			oas.Pairs(oas.Get(components, "headers")),
			func(key, value *yaml.Node) []allowReservedQueryOnlyTarget {
				path := oas.JoinAll("$.components.headers", key.Value)

				return []allowReservedQueryOnlyTarget{allowReservedQueryOnlyOne(root, value, path, true)}
			},
		),
	)
}

func allowReservedQueryOnlyList(root, list *yaml.Node, path string) []allowReservedQueryOnlyTarget {
	return seq.FlatMapPairs(oas.Items(list), func(i int, item *yaml.Node) []allowReservedQueryOnlyTarget {
		return []allowReservedQueryOnlyTarget{allowReservedQueryOnlyOne(root, item, oas.Index(path, i), false)}
	})
}

func allowReservedQueryOnlyHeaders(root, response *yaml.Node, path string) []allowReservedQueryOnlyTarget {
	return seq.FlatMapPairs(
		oas.Pairs(oas.Get(response, "headers")),
		func(key, value *yaml.Node) []allowReservedQueryOnlyTarget {
			return []allowReservedQueryOnlyTarget{
				allowReservedQueryOnlyOne(root, value, oas.JoinAll(path, "headers", key.Value), true),
			}
		},
	)
}

func allowReservedQueryOnlyOne(root, node *yaml.Node, path string, header bool) allowReservedQueryOnlyTarget {
	resolved := oas.Resolve(root, node)
	if header {
		return allowReservedQueryOnlyTarget{node: resolved, path: path, location: allowReservedQueryOnlyHeader}
	}

	return allowReservedQueryOnlyTarget{node: resolved, path: path, location: oas.StringAt(resolved, "in")}
}
