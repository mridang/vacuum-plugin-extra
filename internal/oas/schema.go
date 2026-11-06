package oas

import (
	"iter"
	"maps"
	"slices"

	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// Schema is one schema object found while walking the document.
type Schema struct {
	// Node is the schema object. It is not $ref-resolved: a reference is
	// reported as the node holding the $ref keyword.
	Node *yaml.Node
	// Path is the JSONPath of the schema.
	Path string
	// Component is the components.schemas key when the schema is a top-level
	// component, and empty otherwise.
	Component string
}

// subSchemaKeys are the keywords whose value is a single schema.
func subSchemaKeys() []string {
	return []string{
		"items",
		"additionalProperties",
		"not",
		"contains",
		"propertyNames",
		"if",
		"then",
		"else",
		"unevaluatedProperties",
		"unevaluatedItems",
	}
}

// subSchemaListKeys are the keywords whose value is a list of schemas.
func subSchemaListKeys() []string {
	return []string{"allOf", "anyOf", "oneOf", "prefixItems"}
}

// subSchemaMapKeys are the keywords whose value is a map of schemas.
func subSchemaMapKeys() []string {
	return []string{"properties", "patternProperties", "dependentSchemas", "$defs"}
}

// Schemas iterates over every schema in root: component schemas, parameter and
// header schemas, request and response media type schemas, and all nested
// schemas below them. Each schema node is reported once.
//
// The walk records only a cheap trail for every schema; the JSONPath strings
// are rendered afterwards. A node that was already reported is not entered
// again, which also terminates cyclic aliases.
func Schemas(root *yaml.Node) iter.Seq[Schema] {
	return slices.Values(slices.Collect(func(yield func(Schema) bool) {
		walkRoot(root, func(c candidate) {
			yield(Schema{Node: c.node, Path: c.trail.String(), Component: c.component})
		})
	}))
}

// candidate is a schema found by the walk, with the trail leading to it.
type candidate struct {
	node      *yaml.Node
	trail     *trail
	component string
}

// trail is a JSONPath held as a chain of segments, so that extending a path
// costs one small allocation instead of a string concatenation.
type trail struct {
	parent *trail
	// text is the whole path for a start trail, and the key for a key segment.
	text string
	// index is the array index of an index segment.
	index int
	kind  segmentKind
}

type segmentKind int

const (
	startSegment segmentKind = iota
	keySegment
	indexSegment
)

func start(path string) *trail {
	return &trail{text: path, kind: startSegment}
}

func (t *trail) key(key string) *trail {
	return &trail{parent: t, text: key, kind: keySegment}
}

func (t *trail) at(i int) *trail {
	return &trail{parent: t, index: i, kind: indexSegment}
}

func (t *trail) String() string {
	switch t.kind {
	case keySegment:
		return Join(t.parent.String(), t.text)
	case indexSegment:
		return Index(t.parent.String(), t.index)
	case startSegment:
		return t.text
	default:
		return t.text
	}
}

// visited is an immutable set of the nodes reported so far. It is a list of
// maps whose sizes shrink from the first to the last, merged like a binary
// counter, so that adding a node costs O(log n) amortised and every earlier
// version of the set stays valid.
type visited []map[*yaml.Node]struct{}

func (v visited) has(node *yaml.Node) bool {
	return slices.ContainsFunc(v, func(level map[*yaml.Node]struct{}) bool {
		_, found := level[node]

		return found
	})
}

func (v visited) with(node *yaml.Node) visited {
	return v.withLevel(map[*yaml.Node]struct{}{node: {}})
}

func (v visited) withLevel(level map[*yaml.Node]struct{}) visited {
	if len(v) == 0 || len(v[len(v)-1]) > len(level) {
		return slices.Concat(v, visited{level})
	}

	return v[:len(v)-1].withLevel(maps.Collect(func(yield func(*yaml.Node, struct{}) bool) {
		for node := range v[len(v)-1] {
			if !yield(node, struct{}{}) {
				return
			}
		}

		for node := range level {
			if !yield(node, struct{}{}) {
				return
			}
		}
	}))
}

// forEach threads the visited set through f for every item, in order.
func forEach[T any](seen visited, items []T, f func(T, visited) visited) visited {
	if len(items) == 0 {
		return seen
	}

	return forEach(f(items[0], seen), items[1:], f)
}

type entry struct {
	key, value *yaml.Node
}

func entriesOf(n *yaml.Node) []entry {
	return slices.Collect(func(yield func(entry) bool) {
		for key, value := range Pairs(n) {
			if !yield(entry{key: key, value: value}) {
				return
			}
		}
	})
}

func itemsOf(n *yaml.Node) []*yaml.Node {
	if !IsSeq(n) {
		return nil
	}

	return n.Content
}

func walkRoot(root *yaml.Node, report func(candidate)) visited {
	componentSchemas := start("$.components.schemas")

	afterSchemas := forEach(nil, entriesOf(Dig(root, "components", "schemas")), func(e entry, seen visited) visited {
		return walkSchema(e.value, componentSchemas.key(e.key.Value), e.key.Value, seen, report)
	})

	afterComponents := walkComponents(Get(root, "components"), afterSchemas, report)

	operations := slices.Collect(Operations(root))

	afterOperations := forEach(afterComponents, operations, func(op Operation, seen visited) visited {
		return walkOperation(op, seen, report)
	})

	return forEach(afterOperations, slices.Collect(PathItems(root)), func(item PathItem, seen visited) visited {
		return walkParameterList(Get(item.Node, "parameters"), start(item.Path).key("parameters"), seen, report)
	})
}

func walkComponents(components *yaml.Node, seen visited, report func(candidate)) visited {
	parameters := start("$.components.parameters")
	headers := start("$.components.headers")
	requestBodies := start("$.components.requestBodies")
	responses := start("$.components.responses")

	afterParameters := forEach(seen, entriesOf(Get(components, "parameters")), func(e entry, s visited) visited {
		return walkParameter(e.value, parameters.key(e.key.Value), s, report)
	})

	afterHeaders := forEach(afterParameters, entriesOf(Get(components, "headers")), func(e entry, s visited) visited {
		return walkParameter(e.value, headers.key(e.key.Value), s, report)
	})

	afterBodies := forEach(afterHeaders, entriesOf(Get(components, "requestBodies")), func(e entry, s visited) visited {
		return walkContent(Get(e.value, "content"), requestBodies.key(e.key.Value).key("content"), s, report)
	})

	return forEach(afterBodies, entriesOf(Get(components, "responses")), func(e entry, s visited) visited {
		return walkResponse(e.value, responses.key(e.key.Value), s, report)
	})
}

func walkOperation(operation Operation, seen visited, report func(candidate)) visited {
	path := start(operation.Path)
	responses := path.key("responses")

	afterParameters := walkParameterList(Get(operation.Node, "parameters"), path.key("parameters"), seen, report)

	afterBody := walkContent(
		Dig(operation.Node, "requestBody", "content"),
		path.key("requestBody").key("content"),
		afterParameters,
		report,
	)

	return forEach(afterBody, entriesOf(Get(operation.Node, "responses")), func(e entry, s visited) visited {
		return walkResponse(e.value, responses.key(e.key.Value), s, report)
	})
}

func walkResponse(response *yaml.Node, path *trail, seen visited, report func(candidate)) visited {
	afterContent := walkContent(Get(response, "content"), path.key("content"), seen, report)
	headers := path.key("headers")

	return forEach(afterContent, entriesOf(Get(response, "headers")), func(e entry, s visited) visited {
		return walkParameter(e.value, headers.key(e.key.Value), s, report)
	})
}

func walkParameterList(list *yaml.Node, path *trail, seen visited, report func(candidate)) visited {
	return forEachIndexed(seen, itemsOf(list), func(i int, item *yaml.Node, s visited) visited {
		return walkParameter(item, path.at(i), s, report)
	})
}

// forEachIndexed is forEach passing the index of every item.
func forEachIndexed(seen visited, items []*yaml.Node, f func(int, *yaml.Node, visited) visited) visited {
	return forEach(seen, slices.Collect(func(yield func(indexed) bool) {
		for i, item := range items {
			if !yield(indexed{i: i, node: item}) {
				return
			}
		}
	}), func(it indexed, s visited) visited {
		return f(it.i, it.node, s)
	})
}

type indexed struct {
	i    int
	node *yaml.Node
}

func walkParameter(parameter *yaml.Node, path *trail, seen visited, report func(candidate)) visited {
	afterSchema := walkSchema(Get(parameter, "schema"), path.key("schema"), "", seen, report)

	return walkContent(Get(parameter, "content"), path.key("content"), afterSchema, report)
}

func walkContent(content *yaml.Node, path *trail, seen visited, report func(candidate)) visited {
	return forEach(seen, entriesOf(content), func(e entry, s visited) visited {
		return walkSchema(Get(e.value, "schema"), path.key(e.key.Value).key("schema"), "", s, report)
	})
}

// walkSchema reports the schema at node and every schema below it, depth first,
// and returns the visited set extended with them. A node that was already
// visited is skipped together with everything below it.
func walkSchema(node *yaml.Node, path *trail, component string, seen visited, report func(candidate)) visited {
	if !IsMap(node) || seen.has(node) {
		return seen
	}

	report(candidate{node: node, trail: path, component: component})

	afterSingles := forEach(seen.with(node), subSchemaKeys(), func(key string, s visited) visited {
		child := Get(node, key)
		if !IsMap(child) {
			return s
		}

		return walkSchema(child, path.key(key), "", s, report)
	})

	afterLists := forEach(afterSingles, subSchemaListKeys(), func(key string, s visited) visited {
		items := itemsOf(Get(node, key))
		if len(items) == 0 {
			return s
		}

		base := path.key(key)

		return forEachIndexed(s, items, func(i int, item *yaml.Node, inner visited) visited {
			return walkSchema(item, base.at(i), "", inner, report)
		})
	})

	return forEach(afterLists, subSchemaMapKeys(), func(key string, s visited) visited {
		entries := entriesOf(Get(node, key))
		if len(entries) == 0 {
			return s
		}

		base := path.key(key)

		return forEach(s, entries, func(e entry, inner visited) visited {
			return walkSchema(e.value, base.key(e.key.Value), "", inner, report)
		})
	})
}

// Types returns the declared types of a schema. In OpenAPI 3.1 the type
// keyword may be a list; in 3.0 it is a single string.
func Types(schema *yaml.Node) []string {
	typeNode := Get(schema, "type")
	if IsSeq(typeNode) {
		return Strings(typeNode)
	}

	if text := Text(typeNode); text != "" {
		return []string{text}
	}

	return nil
}

// HasType reports whether the schema declares the given type.
func HasType(schema *yaml.Node, want string) bool {
	return slices.Contains(Types(schema), want)
}

// MediaType is one entry of a content map.
type MediaType struct {
	// Name is the media type key, such as application/json.
	Name string
	// Node is the media type object.
	Node *yaml.Node
	// Path is the JSONPath of the media type object.
	Path string
}

// MediaTypes returns the entries of a content map.
func MediaTypes(content *yaml.Node, path string) []MediaType {
	return seq.FlatMapPairs(Pairs(content), func(key, value *yaml.Node) []MediaType {
		return []MediaType{{Name: key.Value, Node: value, Path: Join(path, key.Value)}}
	})
}
