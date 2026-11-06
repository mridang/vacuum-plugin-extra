package rules

import (
	"regexp"
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// wireParameterNoCamelCaseDefault matches a lower case letter followed by an
// upper case letter, the signature of camelCase.
const wireParameterNoCamelCaseDefault = `[a-z][A-Z]`

// WireParameterNoCamelCase returns the extra-wire-parameter-no-camel-case rule.
//
// # What it checks
//
// The name of a parameter must not contain a lower case letter directly
// followed by an upper case letter, which is what makes a name camelCase.
// Every parameter of a path item, of an operation and of components/parameters
// is checked. Names such as Idempotency-Key, Origin, page_size and
// _zflow_request are accepted because they have no such pair.
//
// Query strings, paths and header names are case-insensitive or conventionally
// lower case on the wire, so camelCase names are an easy source of bugs in
// proxies and clients. Kebab or snake case names do not have that problem.
//
// # Where it comes from
//
// Redocly CLI, the community rule that rejects camelCase parameter names
// (the pattern [a-z][A-Z]).
//
// # Options
//
//   - pattern (string, default [a-z][A-Z]): regular expression a parameter name
//     must not match. An empty string disables the rule, and a pattern that is
//     not a valid regular expression disables it too.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: pageSize
//	          in: query
//	          schema:
//	            type: integer
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: page_size
//	          in: query
//	          schema:
//	            type: integer
//
// # Known false positives
//
// Established headers written in camel case by a third party (for example
// X-Amz-Meta headers sent as xAmzMeta) are reported. Parameters that are $ref
// entries are checked where they are defined, in components/parameters.
func WireParameterNoCamelCase() Rule {
	return New(Spec{
		Name:        "wire-parameter-no-camel-case",
		Description: "Parameter names must not be camelCase.",
		HowToFix:    "Rename the parameter to kebab case or snake case.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        literalPattern,
				Description: "Regular expression a parameter name must not match.",
				Default:     wireParameterNoCamelCaseDefault,
			},
		},
		Check: OnDocument(checkWireParameterNoCamelCase),
	})
}

func checkWireParameterNoCamelCase(call *Call) []Finding {
	pattern := call.String("pattern")
	if pattern == "" {
		return nil
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	return slices.Concat(
		seq.FlatMap(oas.PathItems(call.Root), func(item oas.PathItem) []Finding {
			return wireParameterNoCamelCaseList(
				compiled,
				oas.Get(item.Node, "parameters"),
				oas.Join(item.Path, "parameters"),
			)
		}),
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
			return wireParameterNoCamelCaseList(
				compiled,
				oas.Get(operation.Node, "parameters"),
				oas.Join(operation.Path, "parameters"),
			)
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(call.Root, "components", "parameters")),
			func(key, parameter *yaml.Node) []Finding {
				return wireParameterNoCamelCaseCheck(
					compiled,
					parameter,
					oas.JoinAll("$.components.parameters", key.Value),
				)
			},
		),
	)
}

func wireParameterNoCamelCaseList(pattern *regexp.Regexp, list *yaml.Node, path string) []Finding {
	return seq.FlatMapPairs(oas.Items(list), func(i int, parameter *yaml.Node) []Finding {
		return wireParameterNoCamelCaseCheck(pattern, parameter, oas.Index(path, i))
	})
}

func wireParameterNoCamelCaseCheck(pattern *regexp.Regexp, parameter *yaml.Node, path string) []Finding {
	name := oas.Get(parameter, "name")

	return When(oas.IsScalar(name) && pattern.MatchString(name.Value), func() Finding {
		return Finding{
			Node:    name,
			Path:    oas.Join(path, "name"),
			Message: "parameter name " + name.Value + " must not be camelCase",
		}
	})
}
