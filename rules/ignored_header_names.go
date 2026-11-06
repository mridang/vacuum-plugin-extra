package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// ignoredHeaderNamesHeader is the parameter location checked.
const ignoredHeaderNamesHeader = "header"

const (
	// ignoredHeaderNamesNames is the option that lists the ignored request headers.
	ignoredHeaderNamesNames = "names"
	// ignoredHeaderNamesContentType is the only ignored response header.
	ignoredHeaderNamesContentType = "content-type"
)

// IgnoredHeaderNames returns the extra-ignored-header-names rule.
//
// # What it checks
//
// The OpenAPI specification says that header parameters named Accept,
// Content-Type or Authorization are ignored, and that a response header named
// Content-Type is ignored. The rule flags every such declaration, because the
// definition has no effect. Names are compared case-insensitively.
//
// # Why
//
// The content type of a request is described by the requestBody content keys,
// the accepted type by the response content keys, and credentials by security
// schemes. Declaring them as headers produces documentation that tools do not
// honour and misleads readers.
//
// # Where it comes from
//
// The KaiZen OpenAPI parser (a header named Content-Type is declared and
// ignored).
//
// # Options
//
//   - names (list of strings, default [Accept, Content-Type, Authorization]):
//     the request header names that are reported.
//
// # Bad
//
//	paths:
//	  /pets:
//	    post:
//	      parameters:
//	        - name: Content-Type
//	          in: header
//	          schema:
//	            type: string
//
// # Good
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        content:
//	          application/json:
//	            schema:
//	              type: object
//
// # Known false positives / limitations
//
// None known; the names are fixed by the specification, and the option can be
// narrowed to Content-Type only.
func IgnoredHeaderNames() Rule {
	return New(Spec{
		Name:        "ignored-header-names",
		Description: "Header parameters named Accept, Content-Type or Authorization are ignored.",
		HowToFix:    "Remove the header and describe the media type with content and credentials with securitySchemes.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        ignoredHeaderNamesNames,
				Description: "Request header names that the specification says are ignored.",
				Default:     []string{"Accept", "Content-Type", "Authorization"},
			},
		},
		Check: checkIgnoredHeaderNames,
	})
}

// checkIgnoredHeaderNames reports header declarations that have no effect.
func checkIgnoredHeaderNames(call *Call) []Finding {
	names := seq.Map(slices.Values(call.Strings(ignoredHeaderNamesNames)), Fold)

	return ignoredHeaderNamesUnique(Concat(
		PerOperation(func(call *Call, operation oas.Operation) []Finding {
			return slices.Concat(
				seq.FlatMapSlice(oas.Parameters(call.Root, operation), func(parameter oas.Parameter) []Finding {
					return ignoredHeaderNamesParameter(parameter, names)
				}),
				seq.FlatMapSlice(oas.Responses(call.Root, operation), func(response oas.Response) []Finding {
					return ignoredHeaderNamesResponse(response.Node, response.Path)
				}),
			)
		}),
		PerPathItem(func(call *Call, item oas.PathItem) []Finding {
			return seq.FlatMapPairs(
				oas.Items(oas.Get(item.Node, "parameters")),
				func(i int, node *yaml.Node) []Finding {
					return ignoredHeaderNamesParameter(oas.Parameter{
						Node: oas.Resolve(call.Root, node),
						Path: oas.Index(oas.Join(item.Path, "parameters"), i),
					}, names)
				},
			)
		}),
	)(call))
}

func ignoredHeaderNamesParameter(parameter oas.Parameter, names []string) []Finding {
	if parameter.In() != ignoredHeaderNamesHeader || !slices.Contains(names, Fold(parameter.Name())) {
		return nil
	}

	return One(Finding{
		Node:    parameter.Node,
		Path:    parameter.Path,
		Message: "header parameter " + parameter.Name() + " is ignored by the specification",
	})
}

// ignoredHeaderNamesResponse checks the headers of one response.
func ignoredHeaderNamesResponse(response *yaml.Node, path string) []Finding {
	return seq.FlatMapPairs(oas.Pairs(oas.Get(response, "headers")), func(key, _ *yaml.Node) []Finding {
		return When(strings.ToLower(key.Value) == ignoredHeaderNamesContentType, func() Finding {
			return Finding{
				Node:    key,
				Path:    oas.JoinAll(path, "headers", key.Value),
				Message: "response header " + key.Value + " is ignored by the specification",
			}
		})
	})
}

func ignoredHeaderNamesUnique(findings []Finding) []Finding {
	return seq.FlatMapIndexed(findings, func(i int, finding Finding) []Finding {
		return When(slices.IndexFunc(findings, func(other Finding) bool { return other.Path == finding.Path }) == i,
			func() Finding { return finding })
	})
}
