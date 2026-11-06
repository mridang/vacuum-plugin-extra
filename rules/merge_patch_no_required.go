package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// mergePatchNoRequiredMediaType is the media type of a JSON merge patch.
const mergePatchNoRequiredMediaType = "application/merge-patch+json"

// mergePatchNoRequiredMaxDepth bounds the recursion through schemas.
const mergePatchNoRequiredMaxDepth = 16

// MergePatchNoRequired returns the extra-merge-patch-no-required rule.
//
// # What it checks
//
// The request body of a PATCH operation that is sent as
// application/merge-patch+json must not use required, or a minProperties
// greater than zero, anywhere in its schema.
// The schema is inspected after following $ref, and the properties, items,
// additionalProperties, allOf, anyOf, oneOf and not keywords are descended
// into. Any non-empty required list is reported. When the required list lives
// in a referenced schema, the finding is reported on the schema of the request
// body that reaches it.
//
// A JSON merge patch (RFC 7396) only carries the fields that change, so a
// required property would force clients to resend fields they do not want to
// modify.
//
// # Where it comes from
//
// IBM openapi-validator, rule ibm-dont-require-merge-patch-properties.
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets/{id}:
//	    patch:
//	      requestBody:
//	        content:
//	          application/merge-patch+json:
//	            schema:
//	              type: object
//	              required: [name]
//	              properties:
//	                name:
//	                  type: string
//
// # Good
//
//	paths:
//	  /pets/{id}:
//	    patch:
//	      requestBody:
//	        content:
//	          application/merge-patch+json:
//	            schema:
//	              type: object
//	              properties:
//	                name:
//	                  type: string
//
// # Known false positives
//
// A schema shared between a create request and a merge patch request has to be
// split, because the shared required list is reported for the patch. Only
// local references are followed.
func MergePatchNoRequired() Rule {
	return New(Spec{
		Name:        "merge-patch-no-required",
		Description: "Schemas of merge patch request bodies must not define required properties.",
		HowToFix:    "Remove required from the schema, or use a separate schema for the merge patch request.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkMergePatchNoRequired),
	})
}

func checkMergePatchNoRequired(call *Call, operation oas.Operation) []Finding {
	if operation.Method != methodPatch {
		return nil
	}

	body := oas.Resolve(call.Root, oas.Get(operation.Node, "requestBody"))
	bodyPath := oas.JoinAll(operation.Path, "requestBody", "content")

	return seq.FlatMapPairs(oas.Pairs(oas.Get(body, "content")), func(name, media *yaml.Node) []Finding {
		base, _, _ := strings.Cut(name.Value, ";")
		if !strings.EqualFold(strings.TrimSpace(base), mergePatchNoRequiredMediaType) {
			return nil
		}

		path := oas.JoinAll(bodyPath, name.Value, "schema")
		walker := mergePatchNoRequiredWalker{root: call.Root, anchor: path}

		return walker.visit(oas.Get(media, "schema"), mergePatchNoRequiredStep{path: path})
	})
}

type mergePatchNoRequiredWalker struct {
	root   *yaml.Node
	anchor string
}

type mergePatchNoRequiredStep struct {
	path   string
	depth  int
	viaRef bool
}

func (w mergePatchNoRequiredWalker) visit(schema *yaml.Node, step mergePatchNoRequiredStep) []Finding {
	if step.depth > mergePatchNoRequiredMaxDepth || !oas.IsMap(schema) {
		return nil
	}

	resolved := oas.Resolve(w.root, schema)
	next := mergePatchNoRequiredStep{path: step.path, depth: step.depth, viaRef: step.viaRef || resolved != schema}

	return slices.Concat(w.required(resolved, next), w.minProperties(resolved, next), w.children(resolved, next))
}

func (w mergePatchNoRequiredWalker) location(step mergePatchNoRequiredStep, key string) string {
	if step.viaRef {
		return w.anchor
	}

	return oas.Join(step.path, key)
}

func (w mergePatchNoRequiredWalker) required(schema *yaml.Node, step mergePatchNoRequiredStep) []Finding {
	required := oas.Get(schema, "required")

	return When(oas.IsSeq(required) && len(required.Content) > 0, func() Finding {
		return Finding{
			Node:    required,
			Path:    w.location(step, "required"),
			Message: "schemas of merge patch request bodies must not define required properties",
		}
	})
}

func (w mergePatchNoRequiredWalker) minProperties(schema *yaml.Node, step mergePatchNoRequiredStep) []Finding {
	node := oas.Get(schema, "minProperties")
	value, ok := oas.Int(node)

	return When(ok && value > 0, func() Finding {
		return Finding{
			Node:    node,
			Path:    w.location(step, "minProperties"),
			Message: "schemas of merge patch request bodies must not set a minProperties above zero",
		}
	})
}

func (w mergePatchNoRequiredWalker) child(schema *yaml.Node, step mergePatchNoRequiredStep, path string) []Finding {
	return w.visit(schema, mergePatchNoRequiredStep{path: path, depth: step.depth + 1, viaRef: step.viaRef})
}

func (w mergePatchNoRequiredWalker) children(schema *yaml.Node, step mergePatchNoRequiredStep) []Finding {
	return slices.Concat(
		seq.FlatMapSlice(
			[]string{keywordItems, mergePatchNoRequiredAdditional, keywordNot},
			func(key string) []Finding {
				return w.child(oas.Get(schema, key), step, oas.Join(step.path, key))
			},
		),
		seq.FlatMapSlice([]string{mergePatchNoRequiredAllOf, keywordAnyOf, literalOneof}, func(key string) []Finding {
			return seq.FlatMapPairs(oas.Items(oas.Get(schema, key)), func(i int, member *yaml.Node) []Finding {
				return w.child(member, step, oas.Index(oas.Join(step.path, key), i))
			})
		}),
		seq.FlatMapPairs(oas.Pairs(oas.Get(schema, "properties")), func(name, property *yaml.Node) []Finding {
			return w.child(property, step, oas.JoinAll(step.path, "properties", name.Value))
		}),
	)
}

const (
	mergePatchNoRequiredAllOf      = "allOf"
	mergePatchNoRequiredAdditional = "additionalProperties"
)
