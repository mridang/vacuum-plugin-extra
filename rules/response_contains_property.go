package rules

import (
	"maps"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	responseContainsPropertyOption   = "names"
	responseContainsPropertyMaxDepth = 8
)

// ResponseContainsProperty returns the extra-response-contains-property rule.
//
// # What it checks
//
// A port of Redocly CLI's response-contains-property. For every response
// except 204, the schema of every media type must define the properties that
// the names option lists for the status code. The lookup is the one Redocly
// uses: the exact code (200), else its range (2XX), else the lower-case range
// (2xx); the first key present wins, and default is looked up as itself.
//
// Redocly only inspects a schema whose type is object. This port does the
// same, and additionally inspects a schema that declares no type at all but
// composes objects (properties, allOf, oneOf or anyOf), which is how most
// response envelopes are written. Properties are collected from the schema
// itself, from every allOf member (any member may supply the property) and
// from oneOf and anyOf (the property must be present in every branch, because
// a response matching a branch without it would lack the property). Local
// $refs are followed. A schema that declares a type other than object (an
// array, a string) is skipped, as in Redocly.
//
// The finding is placed on the properties keyword of the media type schema
// (JSONPath .schema.properties), like Redocly's location.child('properties'),
// and says that the response must contain a top-level property.
//
// # Where it comes from
//
// Redocly CLI, rule response-contains-property (packages/core/src/rules/oas3
// and oas2, option names).
//
// # Options
//
//   - names (map of status code or range to a list of property names, no
//     default): for example {"4XX": ["message"], "200": ["id"]}. When the map
//     is empty the rule does nothing.
//
// # Bad
//
//	# names: {"4XX": ["message"]}
//	responses:
//	  "404":
//	    description: Not found
//	    content:
//	      application/json:
//	        schema:
//	          type: object
//	          properties:
//	            code: {type: integer}
//
// # Good
//
//	responses:
//	  "404":
//	    description: Not found
//	    content:
//	      application/json:
//	        schema:
//	          type: object
//	          properties:
//	            message: {type: string}
//
// # Known false positives and limitations
//
// When the schema is a $ref the finding is reported at the referencing media
// type rather than inside the component, because no resolved-document location
// is available. Properties that only exist in a oneOf or anyOf branch are not
// counted (by design, see above). Only top-level properties are checked, as in
// Redocly.
func ResponseContainsProperty() Rule {
	return New(Spec{
		Name:        "response-contains-property",
		Description: "Configured response codes must define the configured top-level properties in their schema.",
		HowToFix:    "Add the required property to the response schema.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        responseContainsPropertyOption,
				Description: "Map of status code (or range) to the list of required property names. Empty disables the rule.",
			},
		},
		Check: PerResponse(checkResponseContainsProperty),
	})
}

// checkResponseContainsProperty applies the configured requirements.
func checkResponseContainsProperty(call *Call, _ oas.Operation, response oas.Response) []Finding {
	config := responseContainsPropertyConfig(call.raw(responseContainsPropertyOption))
	if len(config) == 0 || response.Code == "204" {
		return nil
	}

	return responseContainsPropertyCheck(call.Root, response, responseContainsPropertyRequired(config, response.Code))
}

func responseContainsPropertyConfig(raw any) map[string][]string {
	entries, _ := raw.(map[string]any)

	return maps.Collect(func(yield func(string, []string) bool) {
		for code, value := range entries {
			names, ok := responseContainsPropertyNames(value)
			if ok && !yield(code, names) {
				return
			}
		}
	})
}

func responseContainsPropertyNames(value any) ([]string, bool) {
	switch names := value.(type) {
	case []string:
		return names, true
	case []any:
		return stringsFromAny(names), true
	case string:
		return []string{names}, true
	}

	return nil, false
}

func responseContainsPropertyRange(code string) string {
	if len(code) == len("200") && code[1] >= '0' && code[1] <= '9' && code[2] >= '0' && code[2] <= '9' &&
		code[0] >= '0' &&
		code[0] <= '9' {
		return code[:1] + "XX"
	}

	return code
}

// responseContainsPropertyRequired returns the names configured for a status
// code: the exact code, else the range, else the lower-case range.
func responseContainsPropertyRequired(config map[string][]string, code string) []string {
	rng := responseContainsPropertyRange(code)

	for _, key := range []string{code, rng, strings.ToLower(rng)} {
		if names, ok := config[key]; ok && len(names) > 0 {
			return names
		}
	}

	return nil
}

// responseContainsPropertyCheck reports the required properties that are missing.
func responseContainsPropertyCheck(root *yaml.Node, response oas.Response, required []string) []Finding {
	return seq.FlatMapPairs(oas.Pairs(oas.Get(response.Node, "content")), func(name, media *yaml.Node) []Finding {
		schema := oas.Resolve(root, oas.Get(media, "schema"))
		if !responseContainsPropertyApplies(schema) {
			return nil
		}

		schemaPath := oas.JoinAll(response.Path, "content", name.Value, "schema")

		return seq.FlatMapSlice(required, func(want string) []Finding {
			return When(!responseContainsPropertyHas(root, schema, want, 0), func() Finding {
				return Finding{
					Node:    oas.Get(media, "schema"),
					Path:    oas.Join(schemaPath, "properties"),
					Message: "Response object must contain a top-level \"" + want + "\" property.",
				}
			})
		})
	})
}

func responseContainsPropertyApplies(schema *yaml.Node) bool {
	if !oas.IsMap(schema) {
		return false
	}

	if oas.Has(schema, "type") {
		return oas.HasType(schema, "object")
	}

	return slices.ContainsFunc(
		[]string{"properties", responseContainsPropertyAllOf, literalOneof, responseContainsPropertyAnyOf},
		func(key string) bool { return oas.Has(schema, key) },
	)
}

func responseContainsPropertyHas(root, schema *yaml.Node, name string, depth int) bool {
	resolved := oas.Resolve(root, schema)
	if !oas.IsMap(resolved) || depth > responseContainsPropertyMaxDepth {
		return false
	}

	if oas.Get(oas.Get(resolved, "properties"), name) != nil {
		return true
	}

	for _, part := range oas.Items(oas.Get(resolved, responseContainsPropertyAllOf)) {
		if responseContainsPropertyHas(root, part, name, depth+1) {
			return true
		}
	}

	return slices.ContainsFunc([]string{literalOneof, responseContainsPropertyAnyOf}, func(key string) bool {
		return responseContainsPropertyBranches(root, oas.Get(resolved, key), name, depth)
	})
}

func responseContainsPropertyBranches(root, branches *yaml.Node, name string, depth int) bool {
	if !oas.IsSeq(branches) || len(branches.Content) == 0 {
		return false
	}

	return !slices.ContainsFunc(branches.Content, func(branch *yaml.Node) bool {
		return !responseContainsPropertyHas(root, branch, name, depth+1)
	})
}

const (
	responseContainsPropertyAllOf = "allOf"
	responseContainsPropertyAnyOf = "anyOf"
)
