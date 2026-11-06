package rules

import (
	"regexp"
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// ServerVariablesDefined returns the extra-server-variables-defined rule.
//
// # What it checks
//
// Server URL templates and their variables must agree in both directions.
// Every {variable} in a server url must have a matching entry in the variables
// map of the same server, and every entry of that map must be used by the URL.
// Servers are checked at the document root, on path items and on operations.
// An undefined variable leaves the URL unresolvable; an unused one is dead
// configuration that readers will assume matters.
//
// # Where it comes from
//
// Redocly CLI and rate-my-openapi, which both validate server variables
// against the URL template.
//
// # Options
//
// None.
//
// # Bad
//
//	servers:
//	  - url: https://{region}.example.com
//	    variables:
//	      env:
//	        default: prod
//
// # Good
//
//	servers:
//	  - url: https://{region}.example.com
//	    variables:
//	      region:
//	        default: eu
//
// # Known false positives
//
// None known. Server objects reached through $ref are skipped because the
// OpenAPI specification does not allow references there.
func ServerVariablesDefined() Rule {
	return New(Spec{
		Name:        "server-variables-defined",
		Description: "Server URL variables must be defined, and defined variables must be used.",
		HowToFix:    "Add the missing variable to server.variables or remove the unused one.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check: Concat(
			OnDocument(func(call *Call) []Finding {
				return serverVariablesDefinedList(oas.Get(call.Root, "servers"), "$.servers")
			}),
			PerPathItem(func(_ *Call, item oas.PathItem) []Finding {
				return serverVariablesDefinedList(oas.Get(item.Node, "servers"), oas.Join(item.Path, "servers"))
			}),
			PerOperation(func(_ *Call, operation oas.Operation) []Finding {
				return serverVariablesDefinedList(
					oas.Get(operation.Node, "servers"),
					oas.Join(operation.Path, "servers"),
				)
			}),
		),
	})
}

// serverVariablesDefinedList checks every server of a servers list.
func serverVariablesDefinedList(list *yaml.Node, path string) []Finding {
	return seq.FlatMapPairs(oas.Items(list), func(i int, server *yaml.Node) []Finding {
		return serverVariablesDefinedServer(server, oas.Index(path, i))
	})
}

// serverVariablesDefinedServer reports the undefined and the unused variables
// of one server.
func serverVariablesDefinedServer(server *yaml.Node, path string) []Finding {
	url := oas.Get(server, "url")
	if !oas.IsScalar(url) {
		return nil
	}

	matches := regexp.MustCompile(`\{([^{}]+)\}`).FindAllStringSubmatch(url.Value, -1)
	used := seq.Map(slices.Values(matches), func(match []string) string {
		return match[1]
	})
	variables := oas.Get(server, "variables")
	defined := oas.Keys(variables)

	undefined := seq.FlatMapSlice(used, func(name string) []Finding {
		return When(!slices.Contains(defined, name), func() Finding {
			return Finding{
				Node:    url,
				Path:    oas.Join(path, "url"),
				Message: "server variable {" + name + "} is not defined in variables",
			}
		})
	})
	unused := seq.FlatMapPairs(oas.Pairs(variables), func(key, _ *yaml.Node) []Finding {
		return When(!slices.Contains(used, key.Value), func() Finding {
			return Finding{
				Node:    key,
				Path:    oas.JoinAll(path, "variables", key.Value),
				Message: "server variable " + key.Value + " is defined but not used in the url",
			}
		})
	})

	return slices.Concat(undefined, unused)
}
