package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// AllofSchemaTypeIntersection returns the extra-allof-schema-type-intersection rule.
//
// # What it checks
//
// A schema that declares an explicit type and also lists allOf branches that
// declare their own types must keep a non-empty intersection of those types.
// allOf means "valid against every branch", so a schema of type string whose
// allOf branch is of type integer can never be satisfied. Only the schema and
// its immediate allOf branches are compared, and only explicit type keywords
// count; branches without a type are ignored. A $ref branch is resolved first.
// Type lists of OpenAPI 3.1 are supported, and integer is treated as a subset
// of number.
//
// # Why
//
// An empty intersection makes the schema unsatisfiable, so no value can ever
// validate against it. It is almost always a copy and paste mistake, and
// generators produce a broken or empty model for it.
//
// # Where it comes from
//
// The Speakeasy OpenAPI linter (allof-schema-type-intersection, also known as
// semantic-allof-type-conflict). vacuum ships a related allof-conflicts rule
// that compares properties; this one compares the type keyword.
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      allOf:
//	        - type: string
//
// # Good
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      allOf:
//	        - properties:
//	            name:
//	              type: string
//
// # Known false positives / limitations
//
// Only the immediate branches are compared; nested allOf, oneOf and anyOf are
// not followed. Values of the type keyword that are not JSON Schema types are
// compared literally.
func AllofSchemaTypeIntersection() Rule {
	return New(Spec{
		Name:        "allof-schema-type-intersection",
		Description: "A schema and its immediate allOf branches must declare types with a non-empty intersection.",
		HowToFix:    "Make the types of the schema and of its allOf branches compatible, or remove the conflicting branch.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkAllofSchemaTypeIntersection),
	})
}

func checkAllofSchemaTypeIntersection(call *Call, schema oas.Schema) []Finding {
	branches := oas.Get(schema.Node, "allOf")

	return When(
		oas.IsSeq(branches) && allofSchemaTypeIntersectionConflicts(call.Root, schema.Node, branches),
		func() Finding {
			return Finding{
				Node:    branches,
				Path:    oas.Join(schema.Path, "allOf"),
				Message: "schema type and allOf branch types have no type in common",
			}
		},
	)
}

// allofSchemaTypeIntersectionConflicts reports whether the explicit types of
// the schema and its allOf branches have an empty intersection.
func allofSchemaTypeIntersectionConflicts(root, schema, branches *yaml.Node) bool {
	declared := slices.Concat(
		allofSchemaTypeIntersectionDeclared(oas.Types(schema)),
		seq.FlatMapPairs(oas.Items(branches), func(_ int, branch *yaml.Node) [][]string {
			return allofSchemaTypeIntersectionDeclared(oas.Types(oas.Resolve(root, branch)))
		}),
	)

	return len(declared) > 0 && len(allofSchemaTypeIntersectionAll(declared[0], declared[1:])) == 0
}

// allofSchemaTypeIntersectionDeclared wraps a type list so that an empty list
// (no declared type) contributes nothing.
func allofSchemaTypeIntersectionDeclared(types []string) [][]string {
	if len(types) == 0 {
		return nil
	}

	return [][]string{types}
}

// allofSchemaTypeIntersectionAll intersects the first type list with every later one in order.
func allofSchemaTypeIntersectionAll(first []string, rest [][]string) []string {
	if len(rest) == 0 {
		return first
	}

	return allofSchemaTypeIntersectionAll(allofSchemaTypeIntersectionOf(first, rest[0]), rest[1:])
}

// allofSchemaTypeIntersectionOf intersects two type lists, treating integer as
// a subset of number.
func allofSchemaTypeIntersectionOf(left, right []string) []string {
	return slices.Compact(seq.FlatMapSlice(left, func(candidate string) []string {
		switch {
		case slices.Contains(right, candidate):
			return []string{candidate}
		case candidate == keywordInteger && slices.Contains(right, keywordNumber):
			return []string{candidate}
		case candidate == keywordNumber && slices.Contains(right, keywordInteger):
			return []string{keywordInteger}
		}

		return nil
	}))
}
