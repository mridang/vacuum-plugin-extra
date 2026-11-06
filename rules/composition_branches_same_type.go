package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// compositionBranchesSameTypeMaxDepth bounds the recursion through nested
// compositions and references so that cyclic schemas terminate.
const compositionBranchesSameTypeMaxDepth = 16

// compositionBranchesSameTypeMinDistinct is the number of distinct types that
// makes a composition inconsistent.
const compositionBranchesSameTypeMinDistinct = 2

// CompositionBranchesSameType returns the extra-composition-branches-same-type
// rule.
//
// # What it checks
//
// For every schema that uses allOf, anyOf, oneOf or not, all the leaf schemas
// reachable through those keywords must declare the same type. Branches are
// followed through $ref and through nested compositions, so only the leaves
// (schemas without a composition keyword of their own) are compared. Leaves
// that declare no type are ignored. A type list such as [string, "null"] is
// compared as a whole, so it differs from plain string.
//
// A composition that mixes, for example, an object and a string describes a
// value that no consumer can model with one static type, and most code
// generators fail or fall back to an untyped value.
//
// # Where it comes from
//
// The Oxide OpenAPI lint, which requires every leaf of a oneOf, allOf, anyOf or
// not to have the same type.
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Id:
//	      oneOf:
//	        - type: string
//	        - type: integer
//
// # Good
//
//	components:
//	  schemas:
//	    Id:
//	      oneOf:
//	        - type: string
//	          format: uuid
//	        - type: string
//	          format: ulid
//
// # Known false positives
//
// A deliberate union of different primitive types is reported; this is the
// pattern the rule exists to discourage. Nullable unions written as
// oneOf [T, type null] are reported too. Compositions that are only resolved
// through an external $ref are not followed.
func CompositionBranchesSameType() Rule {
	return New(Spec{
		Name:        "composition-branches-same-type",
		Description: "All leaf schemas of a oneOf, allOf, anyOf or not must have the same type.",
		HowToFix:    "Make every branch of the composition declare the same type, or restructure the schema.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkCompositionBranchesSameType),
	})
}

func checkCompositionBranchesSameType(call *Call, schema oas.Schema) []Finding {
	if !compositionBranchesSameTypeHasComposition(schema.Node) {
		return nil
	}

	unique := slices.Compact(
		slices.Sorted(slices.Values(compositionBranchesSameTypeCollect(call.Root, schema.Node, 0))),
	)

	return When(len(unique) >= compositionBranchesSameTypeMinDistinct, func() Finding {
		return Finding{
			Node:    schema.Node,
			Path:    schema.Path,
			Message: "all branches of the composition must have the same type, found: " + strings.Join(unique, ", "),
		}
	})
}

func compositionBranchesSameTypeKeywords() []string {
	return []string{compositionBranchesSameTypeAllOf, keywordAnyOf, literalOneof, keywordNot}
}

func compositionBranchesSameTypeHasComposition(schema *yaml.Node) bool {
	return slices.ContainsFunc(compositionBranchesSameTypeKeywords(), func(keyword string) bool {
		return oas.Has(schema, keyword)
	})
}

func compositionBranchesSameTypeCollect(root, schema *yaml.Node, depth int) []string {
	if depth > compositionBranchesSameTypeMaxDepth {
		return nil
	}

	return seq.FlatMapSlice(compositionBranchesSameTypeKeywords(), func(keyword string) []string {
		value := oas.Get(schema, keyword)
		return seq.FlatMapSlice(compositionBranchesSameTypeBranches(value), func(branch *yaml.Node) []string {
			return compositionBranchesSameTypeLeaf(root, oas.Resolve(root, branch), depth+1)
		})
	})
}

func compositionBranchesSameTypeBranches(value *yaml.Node) []*yaml.Node {
	if oas.IsSeq(value) {
		return value.Content
	}

	return []*yaml.Node{value}
}

func compositionBranchesSameTypeLeaf(root, branch *yaml.Node, depth int) []string {
	if !oas.IsMap(branch) {
		return nil
	}

	if compositionBranchesSameTypeHasComposition(branch) {
		return compositionBranchesSameTypeCollect(root, branch, depth)
	}

	declared := oas.Types(branch)
	if len(declared) == 0 {
		return nil
	}

	return []string{strings.Join(slices.Sorted(slices.Values(declared)), "|")}
}

const (
	compositionBranchesSameTypeAllOf = "allOf"
)
