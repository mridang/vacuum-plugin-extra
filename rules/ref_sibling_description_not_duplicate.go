package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// RefSiblingDescriptionNotDuplicate returns the extra-ref-sibling-description-not-duplicate rule.
//
// # What it checks
//
// A description written next to a reference is meant to override the
// description of the referenced schema for that one use. When it is exactly
// the same text (compared after collapsing white space) it overrides nothing
// and only repeats what the target already says. Two shapes are recognised:
// the OpenAPI 3.1 sibling form, a schema with $ref and description side by
// side, and the allOf form used for OpenAPI 3.0, a schema whose allOf list
// holds a single $ref member and a description of its own. The reference must
// be local and resolvable, and the target must have a description.
//
// # Where it comes from
//
// IBM's openapi-validator, whose ref-sibling rules reject descriptions that
// duplicate the one of the referenced schema.
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Name:
//	      type: string
//	      description: The name of the pet.
//	    Pet:
//	      type: object
//	      properties:
//	        name:
//	          description: The name of the pet.
//	          allOf:
//	            - $ref: "#/components/schemas/Name"
//
// # Good
//
//	name:
//	  description: The name that the owner chose for the pet.
//	  allOf:
//	    - $ref: "#/components/schemas/Name"
//
// # Known false positives
//
// External references cannot be followed and are not checked. A duplicated
// description is occasionally left on purpose so that it survives later edits
// of the target; the rule still reports it.
func RefSiblingDescriptionNotDuplicate() Rule {
	return New(Spec{
		Name:        "ref-sibling-description-not-duplicate",
		Description: "A description beside a $ref must not duplicate the description of the referenced schema.",
		HowToFix:    "Remove the duplicated description or give it a text that adds information.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerSchema(checkRefSiblingDescriptionNotDuplicate),
	})
}

// checkRefSiblingDescriptionNotDuplicate reports a description that repeats
// the description of the schema it references.
func checkRefSiblingDescriptionNotDuplicate(call *Call, schema oas.Schema) []Finding {
	own := oas.Get(schema.Node, "description")
	target := refSiblingDescriptionNotDuplicateTarget(call.Root, schema.Node)
	text := oas.Get(target, "description")

	return When(
		oas.IsScalar(own) && target != nil && oas.IsScalar(text) &&
			refSiblingDescriptionNotDuplicateNormalise(
				text.Value,
			) == refSiblingDescriptionNotDuplicateNormalise(
				own.Value,
			),
		func() Finding {
			return Finding{
				Node:    own,
				Path:    oas.Join(schema.Path, "description"),
				Message: "description duplicates the description of the referenced schema",
			}
		},
	)
}

// refSiblingDescriptionNotDuplicateTarget returns the referenced schema, either
// through a $ref or through an allOf with a single $ref member.
func refSiblingDescriptionNotDuplicateTarget(root, schema *yaml.Node) *yaml.Node {
	if ref := oas.Ref(schema); ref != "" {
		return oas.Pointer(root, ref)
	}

	allOf := oas.Get(schema, "allOf")
	if !oas.IsSeq(allOf) || len(allOf.Content) != 1 {
		return nil
	}

	if oas.Ref(allOf.Content[0]) == "" {
		return nil
	}

	return oas.Resolve(root, allOf.Content[0])
}

// refSiblingDescriptionNotDuplicateNormalise collapses whitespace.
func refSiblingDescriptionNotDuplicateNormalise(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
