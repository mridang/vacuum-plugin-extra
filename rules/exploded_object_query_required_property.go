package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	// explodedObjectQueryRequiredPropertyQuery is the parameter location checked.
	explodedObjectQueryRequiredPropertyQuery = "query"
	// explodedObjectQueryRequiredPropertyFalse is the value that turns explode off.
	explodedObjectQueryRequiredPropertyFalse = "false"
)

// ExplodedObjectQueryRequiredProperty returns the
// extra-exploded-object-query-required-property rule.
//
// # What it checks
//
// A required query parameter whose schema is an object and that is serialised
// with style form and explode true (both are the defaults for query
// parameters) must have a schema that lists at least one required property.
// The check looks at the required list of the schema, of its $ref target and of
// its allOf members.
//
// # Why
//
// An exploded object is serialised as its individual properties: color=red
// &size=L rather than a single key. If no property is required, a client may
// send an object with no properties at all, which produces an empty query
// string, so the parameter that was declared required is never actually
// present on the wire and the server cannot tell it was omitted.
//
// # Where it comes from
//
// Specmatic (exploded object query parameter validation).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /search:
//	    get:
//	      parameters:
//	        - name: filter
//	          in: query
//	          required: true
//	          schema:
//	            type: object
//	            properties:
//	              color:
//	                type: string
//
// # Good
//
//	paths:
//	  /search:
//	    get:
//	      parameters:
//	        - name: filter
//	          in: query
//	          required: true
//	          schema:
//	            type: object
//	            required: [color]
//	            properties:
//	              color:
//	                type: string
//
// # Known false positives / limitations
//
// Required lists contributed through oneOf or anyOf are not considered, and
// parameters described with content instead of schema are skipped.
func ExplodedObjectQueryRequiredProperty() Rule {
	return New(Spec{
		Name:        "exploded-object-query-required-property",
		Description: "A required exploded object query parameter needs at least one required property.",
		HowToFix:    "Add a required property to the object schema, set explode to false, or make the parameter optional.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkExplodedObjectQueryRequiredProperty),
	})
}

// checkExplodedObjectQueryRequiredProperty reports required exploded objects with no required property.
func checkExplodedObjectQueryRequiredProperty(call *Call) []Finding {
	findings := PerParameter(func(call *Call, _ oas.Operation, parameter oas.Parameter) []Finding {
		return When(explodedObjectQueryRequiredPropertyApplies(call.Root, parameter), func() Finding {
			return Finding{
				Node:    parameter.Node,
				Path:    parameter.Path,
				Message: "required exploded object query parameter " + parameter.Name() + " must have at least one required property",
			}
		})
	})(call)

	return seq.FlatMapIndexed(findings, func(i int, finding Finding) []Finding {
		return When(slices.IndexFunc(findings, func(other Finding) bool { return other.Path == finding.Path }) == i,
			func() Finding { return finding })
	})
}

func explodedObjectQueryRequiredPropertyApplies(root *yaml.Node, parameter oas.Parameter) bool {
	if parameter.In() != explodedObjectQueryRequiredPropertyQuery || !parameter.Required() {
		return false
	}

	style := oas.StringAt(parameter.Node, "style")
	explode := oas.StringAt(parameter.Node, "explode")
	if (style != "" && style != keywordForm) || explode == explodedObjectQueryRequiredPropertyFalse {
		return false
	}

	schema := oas.Resolve(root, oas.Get(parameter.Node, "schema"))
	if !oas.IsMap(schema) || !oas.HasType(schema, "object") {
		return false
	}

	return !explodedObjectQueryRequiredPropertyHasRequired(root, schema, 0)
}

// explodedObjectQueryRequiredPropertyMaxDepth bounds allOf recursion.
const explodedObjectQueryRequiredPropertyMaxDepth = 16

// explodedObjectQueryRequiredPropertyHasRequired reports a non-empty required list.
func explodedObjectQueryRequiredPropertyHasRequired(root, schema *yaml.Node, depth int) bool {
	if len(oas.Strings(oas.Get(schema, "required"))) > 0 {
		return true
	}

	if depth > explodedObjectQueryRequiredPropertyMaxDepth {
		return false
	}

	for _, member := range oas.Items(oas.Get(schema, "allOf")) {
		resolved := oas.Resolve(root, member)
		if explodedObjectQueryRequiredPropertyHasRequired(root, resolved, depth+1) {
			return true
		}
	}

	return false
}
