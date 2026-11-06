package rules

import (
	"iter"
	"maps"
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const deepobjectExampleKeysMaxDepth = 8

// DeepobjectExampleKeys returns the extra-deepobject-example-keys rule.
//
// # What it checks
//
// For every query parameter with style deepObject whose schema is an object
// with declared properties, the keys of the parameter example (example, and the
// value of every entry of examples) must be properties of the schema. The
// check descends into nested objects: a nested example object is compared with
// the properties of the matching nested schema. Properties are collected
// through local $ref and allOf. A schema that allows additionalProperties, or
// that declares no properties, accepts any key.
//
// # Why
//
// deepObject parameters are serialised as filter[status]=open&filter[page]=2.
// An example whose key is misspelt or no longer part of the schema produces
// request samples and mock servers that the real API rejects.
//
// # Where it comes from
//
// Specmatic (deepObject example keys must match the schema).
//
// # Options
//
// None.
//
// # Bad
//
//	parameters:
//	  - name: filter
//	    in: query
//	    style: deepObject
//	    schema:
//	      type: object
//	      properties:
//	        status: {type: string}
//	    example:
//	      state: open
//
// # Good
//
//	parameters:
//	  - name: filter
//	    in: query
//	    style: deepObject
//	    schema:
//	      type: object
//	      properties:
//	        status: {type: string}
//	    example:
//	      status: open
//
// # Known false positives
//
// Properties contributed by oneOf or anyOf branches are not collected, so an
// example that uses a key declared only inside such a branch is flagged.
func DeepobjectExampleKeys() Rule {
	return New(Spec{
		Name:        "deepobject-example-keys",
		Description: "Examples of deepObject query parameters must use keys defined by the parameter schema.",
		HowToFix:    "Rename the example key to a property of the schema, or add the property to the schema.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkDeepobjectExampleKeys),
	})
}

// checkDeepobjectExampleKeys checks every distinct deepObject query parameter
// once, even when it is inherited by several operations.
func checkDeepobjectExampleKeys(call *Call) []Finding {
	parameters := seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []oas.Parameter {
		return seq.Filter(slices.Values(oas.Parameters(call.Root, operation)), func(parameter oas.Parameter) bool {
			return parameter.In() == literalQuery && oas.StringAt(parameter.Node, "style") == "deepObject"
		})
	})

	return seq.FlatMapIndexed(parameters, func(i int, parameter oas.Parameter) []Finding {
		first := slices.IndexFunc(parameters, func(other oas.Parameter) bool {
			return other.Path == parameter.Path
		})

		if first != i {
			return nil
		}

		return deepobjectExampleKeysParameter(call.Root, parameter)
	})
}

// deepobjectExampleKeysParameter checks the example and examples of a
// parameter against its schema.
func deepobjectExampleKeysParameter(root *yaml.Node, parameter oas.Parameter) []Finding {
	schema := oas.Get(parameter.Node, "schema")
	direct := deepobjectExampleKeysCompare(
		root, schema, oas.Get(parameter.Node, "example"), oas.Join(parameter.Path, "example"), 0)
	named := seq.FlatMapPairs(oas.Pairs(oas.Get(parameter.Node, "examples")), func(name, entry *yaml.Node) []Finding {
		value := oas.Get(oas.Resolve(root, entry), "value")
		path := oas.JoinAll(parameter.Path, "examples", name.Value, "value")

		return deepobjectExampleKeysCompare(root, schema, value, path, 0)
	})

	return slices.Concat(direct, named)
}

// deepobjectExampleKeysCompare reports the example keys that the schema does
// not define, descending into nested objects.
func deepobjectExampleKeysCompare(root, schema, example *yaml.Node, path string, depth int) []Finding {
	if !oas.IsMap(example) || depth > deepobjectExampleKeysMaxDepth {
		return nil
	}

	shape := deepobjectExampleKeysShapeOf(root, schema, 0)

	return seq.FlatMapPairs(oas.Pairs(example), func(key, value *yaml.Node) []Finding {
		child, known := shape.properties[key.Value]
		if !known && !shape.open && len(shape.properties) > 0 {
			return One(Finding{
				Node:    key,
				Path:    oas.Join(path, key.Value),
				Message: "example key '" + key.Value + "' is not a property of the parameter schema",
			})
		}

		return deepobjectExampleKeysCompare(root, child, value, oas.Join(path, key.Value), depth+1)
	})
}

// deepobjectExampleKeysShape holds the properties a schema defines, including
// those of its allOf parts, and whether it accepts further properties.
type deepobjectExampleKeysShape struct {
	properties map[string]*yaml.Node
	open       bool
}

// deepobjectExampleKeysShapeOf returns the shape of a schema.
func deepobjectExampleKeysShapeOf(root, schema *yaml.Node, depth int) deepobjectExampleKeysShape {
	resolved := oas.Resolve(root, schema)
	if resolved == nil || depth > deepobjectExampleKeysMaxDepth {
		return deepobjectExampleKeysShape{properties: map[string]*yaml.Node{}}
	}

	additional := oas.Get(resolved, "additionalProperties")
	parts := seq.FlatMapPairs(
		oas.Items(oas.Get(resolved, "allOf")),
		func(_ int, part *yaml.Node) []deepobjectExampleKeysShape {
			return []deepobjectExampleKeysShape{deepobjectExampleKeysShapeOf(root, part, depth+1)}
		},
	)
	own := func(yield func(string, *yaml.Node) bool) {
		for name, value := range oas.Pairs(oas.Get(resolved, "properties")) {
			if !yield(name.Value, value) {
				return
			}
		}
	}
	inherited := seq.Map(slices.Values(parts), func(part deepobjectExampleKeysShape) iter.Seq2[string, *yaml.Node] {
		return maps.All(part.properties)
	})

	return deepobjectExampleKeysShape{
		properties: maps.Collect(
			deepobjectExampleKeysChain(slices.Concat([]iter.Seq2[string, *yaml.Node]{own}, inherited)),
		),
		open: additional != nil && (oas.IsTrue(additional) || !oas.IsScalar(additional)) ||
			slices.ContainsFunc(parts, func(part deepobjectExampleKeysShape) bool {
				return part.open
			}),
	}
}

// deepobjectExampleKeysChain yields the entries of every sequence in order, so
// that a later entry for a name wins when the result is collected.
func deepobjectExampleKeysChain(parts []iter.Seq2[string, *yaml.Node]) iter.Seq2[string, *yaml.Node] {
	return func(yield func(string, *yaml.Node) bool) {
		for _, part := range parts {
			for name, value := range part {
				if !yield(name, value) {
					return
				}
			}
		}
	}
}
