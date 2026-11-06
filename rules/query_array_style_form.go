package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// QueryArrayStyleForm returns the extra-query-array-style-form rule.
//
// # What it checks
//
// In OpenAPI 3 documents, a query parameter whose schema is an array must set
// style: form explicitly. A parameter with a different style, or with no style
// at all, is reported. Parameters are read from operations, path items and
// components.parameters, and $ref schemas are followed.
//
// # Why
//
// form is the default style of query parameters, but relying on the default
// hides an important choice: together with explode it decides whether the query
// string reads tags=a&tags=b or tags=a,b. Writing the style down avoids tool
// disagreements and keeps the intent visible to reviewers.
//
// # Where it comes from
//
// Zally rule 154, "Use and Specify Explicitly the Form-Style Query Format for
// Collection Parameters" (QueryParameterCollectionFormatRule, severity SHOULD).
// Like the original it examines components.parameters, path item parameters and
// operation parameters of OpenAPI 3 documents, selects those with in: query and
// a schema of type array, and reports a missing style or any style other than
// form.
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: tags
//	          in: query
//	          schema:
//	            type: array
//	            items:
//	              type: string
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: tags
//	          in: query
//	          style: form
//	          schema:
//	            type: array
//	            items:
//	              type: string
//
// # Known false positives / limitations
//
// APIs that purposely use spaceDelimited, pipeDelimited or deepObject for
// arrays are reported. Parameters described with content instead of schema are
// ignored. OpenAPI 2.0 documents are not checked.
func QueryArrayStyleForm() Rule {
	return New(Spec{
		Name:        "query-array-style-form",
		Description: "Array query parameters must set style to form explicitly.",
		HowToFix:    "Add style: form to the array query parameter.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Formats:     []string{formatOAS3},
		Check:       OnDocument(checkQueryArrayStyleForm),
	})
}

// checkQueryArrayStyleForm reports array query parameters without style form.
func checkQueryArrayStyleForm(call *Call) []Finding {
	parameters := queryArrayStyleFormAll(call.Root)

	return seq.FlatMapIndexed(parameters, func(i int, parameter oas.Parameter) []Finding {
		firstSeen := slices.IndexFunc(parameters, func(other oas.Parameter) bool {
			return other.Node == parameter.Node
		}) == i
		schema := oas.Resolve(call.Root, oas.Get(parameter.Node, "schema"))

		return When(
			firstSeen && parameter.In() == literalQuery &&
				oas.HasType(schema, "array") && oas.StringAt(parameter.Node, "style") != keywordForm,
			func() Finding {
				return Finding{
					Node:    parameter.Node,
					Path:    parameter.Path,
					Message: "array query parameter '" + parameter.Name() + "' must set style to form",
				}
			},
		)
	})
}

// queryArrayStyleFormAll lists the parameters of the whole document.
func queryArrayStyleFormAll(root *yaml.Node) []oas.Parameter {
	return slices.Concat(
		seq.FlatMap(oas.PathItems(root), func(item oas.PathItem) []oas.Parameter {
			return seq.FlatMapPairs(
				oas.Items(oas.Get(item.Node, "parameters")),
				func(i int, entry *yaml.Node) []oas.Parameter {
					return queryArrayStyleFormEntry(root, entry, oas.Index(oas.Join(item.Path, "parameters"), i))
				},
			)
		}),
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []oas.Parameter {
			return oas.OwnParameters(root, operation)
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "parameters")),
			func(key, value *yaml.Node) []oas.Parameter {
				return queryArrayStyleFormEntry(root, value, oas.JoinAll("$.components.parameters", key.Value))
			},
		),
	)
}

// queryArrayStyleFormEntry returns the resolved parameter found at the path.
func queryArrayStyleFormEntry(root, entry *yaml.Node, path string) []oas.Parameter {
	return []oas.Parameter{{Node: oas.Resolve(root, entry), Path: path}}
}
