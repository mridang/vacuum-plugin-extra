package rules

import (
	"maps"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	// noCircularSchemaRefsAllOf is the allOf keyword.
	noCircularSchemaRefsAllOf = "allOf"
	// noCircularSchemaRefsAnyOf is the anyOf keyword.
	noCircularSchemaRefsAnyOf = "anyOf"
	// noCircularSchemaRefsOneOf is the oneOf keyword.
	noCircularSchemaRefsOneOf = "oneOf"
)

const (
	// noCircularSchemaRefsIgnoreProperties is the option that ignores property edges.
	noCircularSchemaRefsIgnoreProperties = "ignoreProperties"
	// noCircularSchemaRefsPrefix is the $ref prefix of component schemas.
	noCircularSchemaRefsPrefix = "#/components/schemas/"
)

// NoCircularSchemaRefs returns the extra-no-circular-schema-refs rule.
//
// # What it checks
//
// A component schema must not reference itself, directly or through other
// component schemas, by following $ref through its sub-schemas: properties,
// allOf, anyOf, oneOf, items, prefixItems, additionalProperties and not. Every
// component that sits on such a cycle is reported once, with the chain that
// leads back to it.
//
// # Why
//
// Circular schemas cannot be fully expanded. Tools that dereference or bundle
// the document, generate example payloads or validate recursively either fail
// or loop, and several code generators produce unusable types.
//
// # Where it comes from
//
// The IBM API Handbook ruleset (ibm-no-circular-refs).
//
// # Options
//
//   - ignoreProperties (bool, default false): do not follow object properties,
//     which allows the common tree shape (a node with an optional child list)
//     while still reporting cycles that go through composition keywords.
//
// # Bad
//
//	components:
//	  schemas:
//	    Node:
//	      type: object
//	      properties:
//	        parent:
//	          $ref: "#/components/schemas/Node"
//
// # Good
//
//	components:
//	  schemas:
//	    Node:
//	      type: object
//	      properties:
//	        parentId:
//	          type: string
//
// # Known false positives / limitations
//
// Legitimate recursive data structures such as trees are reported unless
// ignoreProperties is set. Only local references to components.schemas are
// followed.
func NoCircularSchemaRefs() Rule {
	return New(Spec{
		Name:        "no-circular-schema-refs",
		Description: "Component schemas must not reference themselves through $ref cycles.",
		HowToFix:    "Break the cycle, for example by referencing the related resource by identifier.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        noCircularSchemaRefsIgnoreProperties,
				Description: "Do not follow object properties when looking for cycles.",
				Default:     false,
			},
		},
		Check: OnDocument(checkNoCircularSchemaRefs),
	})
}

// checkNoCircularSchemaRefs reports every component on a reference cycle.
func checkNoCircularSchemaRefs(call *Call) []Finding {
	ignoreProperties := call.Bool(noCircularSchemaRefsIgnoreProperties)
	schemas := oas.Dig(call.Root, "components", "schemas")
	graph := maps.Collect(func(yield func(string, []string) bool) {
		for key, value := range oas.Pairs(schemas) {
			if !yield(key.Value, noCircularSchemaRefsEdges(value, ignoreProperties)) {
				return
			}
		}
	})

	live := noCircularSchemaRefsPeel(graph)

	return seq.FlatMapPairs(oas.Pairs(schemas), func(key, _ *yaml.Node) []Finding {
		cycle := noCircularSchemaRefsCycle(live, key.Value)

		return When(cycle != nil, func() Finding {
			return Finding{
				Node:    key,
				Path:    oas.JoinAll("$.components.schemas", key.Value),
				Message: "schema is part of a circular $ref chain: " + strings.Join(cycle, " -> "),
			}
		})
	})
}

// noCircularSchemaRefsEdges collects the component names referenced below schema.
func noCircularSchemaRefsEdges(schema *yaml.Node, ignoreProperties bool) []string {
	if !oas.IsMap(schema) {
		return nil
	}

	if name, ok := strings.CutPrefix(oas.Ref(schema), noCircularSchemaRefsPrefix); ok {
		return []string{name}
	}

	return slices.Concat(
		seq.FlatMapSlice([]string{keywordItems, "additionalProperties", keywordNot}, func(key string) []string {
			return noCircularSchemaRefsEdges(oas.Get(schema, key), ignoreProperties)
		}),
		seq.FlatMapSlice(
			[]string{noCircularSchemaRefsAllOf, noCircularSchemaRefsAnyOf, noCircularSchemaRefsOneOf, "prefixItems"},
			func(key string) []string {
				return seq.FlatMapPairs(oas.Items(oas.Get(schema, key)), func(_ int, member *yaml.Node) []string {
					return noCircularSchemaRefsEdges(member, ignoreProperties)
				})
			},
		),
		noCircularSchemaRefsPropertyEdges(schema, ignoreProperties),
	)
}

// noCircularSchemaRefsPropertyEdges collects the references below the
// properties of schema, unless properties are ignored.
func noCircularSchemaRefsPropertyEdges(schema *yaml.Node, ignoreProperties bool) []string {
	if ignoreProperties {
		return nil
	}

	return seq.FlatMapPairs(oas.Pairs(oas.Get(schema, "properties")), func(_, property *yaml.Node) []string {
		return noCircularSchemaRefsEdges(property, ignoreProperties)
	})
}

// noCircularSchemaRefsPeel drops, until nothing more changes, every node that
// has no incoming edge or no outgoing edge among the remaining nodes. What is
// left is the only part of the graph in which a cycle can be found.
func noCircularSchemaRefsPeel(graph map[string][]string) map[string][]string {
	heads := slices.Compact(slices.Sorted(slices.Values(slices.Concat(slices.Collect(maps.Values(graph))...))))
	next := maps.Collect(func(yield func(string, []string) bool) {
		for name, edges := range graph {
			kept := slices.DeleteFunc(slices.Clone(edges), func(edge string) bool {
				_, inGraph := graph[edge]

				return !inGraph || !slices.Contains(heads, edge)
			})
			if slices.Contains(heads, name) && len(kept) > 0 && !yield(name, kept) {
				return
			}
		}
	})

	if len(next) == len(graph) {
		return next
	}

	return noCircularSchemaRefsPeel(next)
}

// noCircularSchemaRefsCycle returns the chain that leads from start back to
// start, or nil when start is not on a cycle. The graph must already be peeled.
// The depth-first search threads its visited set through the recursion by
// returning it, so nothing is mutated.
func noCircularSchemaRefsCycle(graph map[string][]string, start string) []string {
	if _, ok := graph[start]; !ok {
		return nil
	}

	found, _ := noCircularSchemaRefsExplore(graph, start, graph[start], []string{start}, nil)

	return found
}

// noCircularSchemaRefsExplore walks the successors in nexts depth first and
// returns the chain that closes on start, plus the visited set.
func noCircularSchemaRefsExplore(
	graph map[string][]string,
	start string,
	nexts, chain, visited []string,
) ([]string, []string) {
	if len(nexts) == 0 {
		return nil, visited
	}

	next := nexts[0]
	if next == start {
		return slices.Concat(chain, []string{start}), visited
	}

	index, seen := slices.BinarySearch(visited, next)
	if seen {
		return noCircularSchemaRefsExplore(graph, start, nexts[1:], chain, visited)
	}

	found, afterChild := noCircularSchemaRefsExplore(
		graph,
		start,
		graph[next],
		slices.Concat(chain, []string{next}),
		slices.Concat(visited[:index], []string{next}, visited[index:]),
	)
	if found != nil {
		return found, afterChild
	}

	return noCircularSchemaRefsExplore(graph, start, nexts[1:], chain, afterChild)
}
