package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const noRefInExampleHeaders = "headers"

const (
	noRefInExampleExamples = "examples"
	noRefInExampleKeyword  = "example"
)

// NoRefInExample returns the extra-no-ref-in-example rule.
//
// # What it checks
//
// A $ref must not appear anywhere inside the value of an example field, at any
// nesting depth. The rule runs on the document as written, before references
// are resolved. A $ref is allowed inside examples, whose entries are Example
// Objects that may be references. Property names that merely happen to be
// "example" (for instance a schema property called example) are not mistaken
// for the keyword.
//
// # Why
//
// In OpenAPI the example field holds a literal value, not an OpenAPI object. A
// $ref there is not resolved by compliant tools: it is shown as a literal
// {"$ref": "..."} object, which is never what the author meant.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-no-ref-in-example).
//
// # Options
//
// None.
//
// # Bad
//
//	content:
//	  application/json:
//	    example:
//	      $ref: "#/components/examples/Pet"
//
// # Good
//
//	content:
//	  application/json:
//	    examples:
//	      pet:
//	        $ref: "#/components/examples/Pet"
//
// # Known false positives and limitations
//
// An example payload that legitimately contains a property called $ref (for
// instance documentation of a JSON reference format) is flagged. Vendor
// extension subtrees (x-*) are ignored.
func NoRefInExample() Rule {
	return New(Spec{
		Name:        "no-ref-in-example",
		Description: "A $ref must not be used inside an example field; use examples instead.",
		HowToFix:    "Inline the example value, or move the reference into an examples entry.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkNoRefInExample),
	})
}

// checkNoRefInExample walks the document looking for example keywords holding a $ref.
func checkNoRefInExample(call *Call) []Finding {
	return noRefInExampleWalk(call.Root, "$", false)
}

// noRefInExampleNameMaps lists the keys whose children are names chosen by the author, not keywords.
func noRefInExampleNameMaps() []string {
	return []string{
		literalProperties, literalPatternproperties, "$defs", "definitions", "dependentSchemas", "schemas",
		literalParameters, "responses", "requestBodies", noRefInExampleHeaders, "securitySchemes", "links", "callbacks",
		literalPaths, "encoding", "content", noRefInExampleExamples, "variables", "mapping", "pathItems",
	}
}

// noRefInExampleWalk visits a node; names marks a mapping whose keys are author chosen names.
func noRefInExampleWalk(node *yaml.Node, path string, names bool) []Finding {
	switch {
	case oas.IsMap(node):
		return seq.FlatMapPairs(oas.Pairs(node), func(key, value *yaml.Node) []Finding {
			return noRefInExampleEntry(key.Value, value, oas.Join(path, key.Value), names)
		})
	case oas.IsSeq(node):
		return seq.FlatMapPairs(oas.Items(node), func(i int, item *yaml.Node) []Finding {
			return noRefInExampleWalk(item, oas.Index(path, i), false)
		})
	default:
		return nil
	}
}

// noRefInExampleEntry handles one key of a mapping.
func noRefInExampleEntry(key string, value *yaml.Node, path string, names bool) []Finding {
	if !names && key == noRefInExampleKeyword {
		return noRefInExampleRefs(value, path)
	}

	if !names && len(key) > 1 && key[:2] == "x-" {
		return nil
	}

	return noRefInExampleWalk(value, path, !names && slices.Contains(noRefInExampleNameMaps(), key))
}

// noRefInExampleRefs reports every $ref key found below an example value.
func noRefInExampleRefs(example *yaml.Node, path string) []Finding {
	return seq.FlatMap(oas.Walk(example, path), func(node oas.Node) []Finding {
		return When(node.Key == "$ref", func() Finding {
			return Finding{
				Node:    node.Node,
				Path:    node.Path,
				Message: "$ref is not allowed within an example field; use examples instead",
			}
		})
	})
}
