package oas

import (
	"iter"
	"maps"
	"slices"
	"strings"

	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// Methods lists the HTTP methods that can appear in a path item, in the order
// the OpenAPI specification declares them.
func Methods() []string {
	return []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}
}

// IsMethod reports whether key is an HTTP method keyword of a path item.
func IsMethod(key string) bool {
	for _, method := range Methods() {
		if method == strings.ToLower(key) {
			return true
		}
	}

	return false
}

// Operation is one operation of the document together with its location.
type Operation struct {
	// Route is the path template, such as /pets/{petId}.
	Route string
	// Method is the lower-case HTTP method.
	Method string
	// PathItem is the path item that holds the operation, with $ref resolved.
	PathItem *yaml.Node
	// Node is the operation object.
	Node *yaml.Node
	// Path is the JSONPath of the operation.
	Path string
}

// PathItem is one entry of the paths object.
type PathItem struct {
	// Route is the path template.
	Route string
	// KeyNode is the key node of the route.
	KeyNode *yaml.Node
	// Node is the path item with $ref resolved.
	Node *yaml.Node
	// Path is the JSONPath of the path item.
	Path string
}

// PathItems iterates over the entries of the paths object of root.
func PathItems(root *yaml.Node) iter.Seq[PathItem] {
	return func(yield func(PathItem) bool) {
		for key, value := range Pairs(Get(root, "paths")) {
			if strings.HasPrefix(key.Value, "x-") {
				continue
			}

			item := PathItem{
				Route:   key.Value,
				KeyNode: key,
				Node:    Resolve(root, value),
				Path:    Join("$.paths", key.Value),
			}
			if !yield(item) {
				return
			}
		}
	}
}

// Operations iterates over every operation declared under paths in root.
func Operations(root *yaml.Node) iter.Seq[Operation] {
	return func(yield func(Operation) bool) {
		for item := range PathItems(root) {
			for method := range Pairs(item.Node) {
				if !IsMethod(method.Value) {
					continue
				}

				operation := Operation{
					Route:    item.Route,
					Method:   strings.ToLower(method.Value),
					PathItem: item.Node,
					Node:     Get(item.Node, method.Value),
					Path:     Join(item.Path, method.Value),
				}
				if !yield(operation) {
					return
				}
			}
		}
	}
}

// Parameter is a resolved parameter with its location.
type Parameter struct {
	// Node is the parameter object with $ref resolved.
	Node *yaml.Node
	// Path is the JSONPath of the parameter entry.
	Path string
	// Entry is the item as written in the parameters list, which is the $ref
	// object when the parameter is a reference. Report findings on it, because
	// Node is shared by every operation that references the same component.
	Entry *yaml.Node
	// Inherited is true when the parameter comes from the path item.
	Inherited bool
}

// Name returns the parameter name.
func (p Parameter) Name() string {
	return StringAt(p.Node, "name")
}

// In returns the parameter location: query, header, path or cookie.
func (p Parameter) In() string {
	return StringAt(p.Node, "in")
}

// Required reports whether the parameter is marked required.
func (p Parameter) Required() bool {
	return IsTrue(Get(p.Node, "required"))
}

// Parameters returns the parameters of an operation. Parameters declared at the
// path item are included unless the operation overrides them by name and
// location, as the specification requires.
func Parameters(root *yaml.Node, op Operation) []Parameter {
	own := collectParameters(root, Get(op.Node, "parameters"), Join(op.Path, "parameters"), false)

	seen := maps.Collect(func(yield func(string, bool) bool) {
		for _, parameter := range own {
			if !yield(parameterKey(parameter), true) {
				return
			}
		}
	})

	inherited := collectParameters(
		root,
		Get(op.PathItem, "parameters"),
		Join(Join("$.paths", op.Route), "parameters"),
		true,
	)

	return slices.Concat(own, seq.Filter(slices.Values(inherited), func(parameter Parameter) bool {
		return !seen[parameterKey(parameter)]
	}))
}

func parameterKey(parameter Parameter) string {
	return parameter.In() + "/" + parameter.Name()
}

// OwnParameters returns only the parameters declared on the operation itself.
func OwnParameters(root *yaml.Node, op Operation) []Parameter {
	return collectParameters(root, Get(op.Node, "parameters"), Join(op.Path, "parameters"), false)
}

func collectParameters(root, list *yaml.Node, base string, inherited bool) []Parameter {
	return seq.FlatMapPairs(Items(list), func(i int, item *yaml.Node) []Parameter {
		return []Parameter{{
			Node:      Resolve(root, item),
			Path:      Index(base, i),
			Entry:     item,
			Inherited: inherited,
		}}
	})
}

// Response is one response of an operation.
type Response struct {
	// Code is the status code key, such as 200, 4XX or default.
	Code string
	// Node is the response object with $ref resolved.
	Node *yaml.Node
	// Path is the JSONPath of the response entry.
	Path string
}

// Responses iterates over the responses of an operation.
func Responses(root *yaml.Node, op Operation) []Response {
	base := Join(op.Path, "responses")

	return seq.FlatMapPairs(Pairs(Get(op.Node, "responses")), func(key, value *yaml.Node) []Response {
		if strings.HasPrefix(key.Value, "x-") {
			return nil
		}

		return []Response{{
			Code: key.Value,
			Node: Resolve(root, value),
			Path: Join(base, key.Value),
		}}
	})
}

// IsSuccess reports whether the status code is a 2xx or 2XX code.
func IsSuccess(code string) bool {
	return strings.HasPrefix(code, "2")
}

// IsClientOrServerError reports whether the status code is 4xx or 5xx.
func IsClientOrServerError(code string) bool {
	return strings.HasPrefix(code, "4") || strings.HasPrefix(code, "5")
}
