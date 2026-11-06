package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// ArrayParameterSerialization returns the extra-array-parameter-serialization rule.
//
// # What it checks
//
// Every parameter whose schema is an array must declare both style and
// explode. Parameters are read from operations, from path items and from
// components.parameters. The check can be limited to some parameter locations
// with the in option.
//
// # Why
//
// The default serialization of an array depends on the parameter location and
// differs between tools and OpenAPI versions (form with explode true for query,
// simple for path and header). Spelling it out makes the wire format explicit,
// so clients, servers and documentation agree.
//
// # Where it comes from
//
// Redocly CLI (array-parameter-serialization).
//
// # Options
//
//   - in: list of parameter locations to check. Default: query, header, path,
//     cookie.
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
//	          explode: true
//	          schema:
//	            type: array
//	            items:
//	              type: string
//
// # Known false positives / limitations
//
// Parameters described with content instead of schema carry no style and are
// ignored. A parameter shared through a $ref is reported once, where it is first
// used.
func ArrayParameterSerialization() Rule {
	return New(Spec{
		Name:        "array-parameter-serialization",
		Description: "Array parameters must declare both style and explode.",
		HowToFix:    "Add style and explode to the array parameter so its serialization is explicit.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Formats:     []string{formatOAS3},
		Options: []Option{{
			Name:        "in",
			Description: "Parameter locations to check: query, header, path and cookie.",
			Default:     []string{literalQuery, keywordHeader, literalPath, "cookie"},
		}},
		Check: OnDocument(checkArrayParameterSerialization),
	})
}

func checkArrayParameterSerialization(call *Call) []Finding {
	locations := call.Strings("in")
	parameters := documentParameters(call.Root)

	return seq.FlatMapIndexed(parameters, func(i int, parameter oas.Parameter) []Finding {
		first := slices.IndexFunc(parameters, func(other oas.Parameter) bool {
			return other.Node == parameter.Node
		})
		if first != i || !slices.Contains(locations, parameter.In()) {
			return nil
		}

		if !oas.HasType(oas.Resolve(call.Root, oas.Get(parameter.Node, "schema")), "array") {
			return nil
		}

		return seq.FlatMapSlice([]string{"style", "explode"}, func(keyword string) []Finding {
			return When(!oas.Has(parameter.Node, keyword), func() Finding {
				return Finding{
					Node:    parameter.Node,
					Path:    parameter.Path,
					Message: "array parameter '" + parameter.Name() + "' must declare " + keyword,
				}
			})
		})
	})
}
