package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

const (
	// schemaDescriptionRequiredAllOf is the allOf keyword.
	schemaDescriptionRequiredAllOf = "allOf"
	// schemaDescriptionRequiredAnyOf is the anyOf keyword.
	schemaDescriptionRequiredAnyOf = "anyOf"
	// schemaDescriptionRequiredOneOf is the oneOf keyword.
	schemaDescriptionRequiredOneOf = "oneOf"
)

const (
	// schemaDescriptionRequiredScope is the option that selects the schemas checked.
	schemaDescriptionRequiredScope = "scope"
	// schemaDescriptionRequiredComponents limits the check to component schemas.
	schemaDescriptionRequiredComponents = "components"
	// schemaDescriptionRequiredMaxDepth bounds the composition search.
	schemaDescriptionRequiredMaxDepth = 8
)

// SchemaDescriptionRequired returns the extra-schema-description-required rule.
//
// # What it checks
//
// Every schema must have a non-empty description. A schema that is a bare $ref
// is not checked (its target is), and a schema counts as described when any
// member of its allOf, anyOf or oneOf lists (followed through local $ref) has a
// description. With scope set to "components" only the top-level schemas under
// components.schemas are checked.
//
// # Why
//
// Descriptions are the documentation of the data model. Generated reference
// pages and client libraries show them, and a schema without one leaves its
// meaning to be guessed.
//
// # Where it comes from
//
// The IBM API Handbook ruleset (ibm-schema-description-exists).
//
// # Options
//
//   - scope (string, default "all"): "all" checks every schema including
//     properties and items, "components" checks component schemas only.
//
// # Bad
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//
// # Good
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      description: A pet in the store.
//
// # Known false positives / limitations
//
// With scope "all" every property schema needs a description, which is noisy
// for self-explanatory fields. Schemas that only wrap another schema with
// allOf and carry no description of their own are accepted only when a member
// has one.
func SchemaDescriptionRequired() Rule {
	return New(Spec{
		Name:        "schema-description-required",
		Description: "Schemas must have a description.",
		HowToFix:    "Add a description to the schema.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        schemaDescriptionRequiredScope,
				Description: "Schemas to check: all or components.",
				Default:     "all",
			},
		},
		Check: OnDocument(checkSchemaDescriptionRequired),
	})
}

// checkSchemaDescriptionRequired reports schemas without a description.
func checkSchemaDescriptionRequired(call *Call) []Finding {
	componentsOnly := call.String(schemaDescriptionRequiredScope) == schemaDescriptionRequiredComponents

	return PerSchema(func(call *Call, schema oas.Schema) []Finding {
		skipped := componentsOnly && schema.Component == ""

		return When(
			!skipped && !oas.Has(schema.Node, "$ref") && !schemaDescriptionRequiredHas(call.Root, schema.Node, 0),
			func() Finding {
				return Finding{
					Node:    schema.Node,
					Path:    schema.Path,
					Message: "schema must have a description",
				}
			},
		)
	})(call)
}

// schemaDescriptionRequiredHas reports a description on the schema or a composed member.
func schemaDescriptionRequiredHas(root, schema *yaml.Node, depth int) bool {
	resolved := oas.Resolve(root, schema)
	if strings.TrimSpace(oas.StringAt(resolved, "description")) != "" {
		return true
	}

	if depth >= schemaDescriptionRequiredMaxDepth {
		return false
	}

	keywords := []string{
		schemaDescriptionRequiredAllOf,
		schemaDescriptionRequiredAnyOf,
		schemaDescriptionRequiredOneOf,
	}

	return slices.ContainsFunc(keywords, func(keyword string) bool {
		members := oas.Get(resolved, keyword)

		return oas.IsSeq(members) && slices.ContainsFunc(members.Content, func(member *yaml.Node) bool {
			return schemaDescriptionRequiredHas(root, member, depth+1)
		})
	})
}
