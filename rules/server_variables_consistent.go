package rules

import (
	"regexp"
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const serverVariablesConsistentEnumOption = "requireEnum"

// serverVariablesConsistentPlaceholder matches a {name} placeholder of a server URL.
const serverVariablesConsistentPlaceholder = `\{([^{}]+)\}`

// ServerVariablesConsistent returns the extra-server-variables-consistent rule.
//
// # What it checks
//
// Every server object (root servers, path item servers, operation servers and
// the server of a link) is checked, and for each of its variables:
//
//   - the variable must define a non-empty default;
//   - when it defines an enum, the enum must not be empty and must contain the
//     default;
//   - when requireEnum is true, the variable must define an enum;
//   - every {placeholder} in the server url must have a matching entry in
//     variables.
//
// Findings are reported on the variable, or on the server for a placeholder
// without a variable.
//
// # Why
//
// Tools build the real URL by substituting each variable with its default. An
// absent or empty default, or a default outside the enum, yields an invalid URL
// and generators fail or silently choose a wrong value.
//
// # Where it comes from
//
// Four rules of the source list are merged here: IBM (server variables have a
// non-empty default), Redocly (enum is not empty and contains the default),
// rate-my-openapi (each variable defines an enum, here the requireEnum option)
// and Spectral (server variables are consistent).
//
// # Options
//
//   - requireEnum (boolean, default false): require every server variable to
//     define an enum.
//
// # Bad
//
//	servers:
//	  - url: https://{region}.example.com
//	    variables:
//	      region:
//	        default: ""
//
// # Good
//
//	servers:
//	  - url: https://{region}.example.com
//	    variables:
//	      region:
//	        default: eu
//	        enum: [eu, us]
//
// # Known false positives
//
// None known. Placeholders inside a url that is itself a $ref are not
// inspected.
func ServerVariablesConsistent() Rule {
	return New(Spec{
		Name:        "server-variables-consistent",
		Description: "Server variables must have a default, which must be one of the enum values.",
		HowToFix:    "Give every server variable a non-empty default that is part of its enum, and define all URL placeholders.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{{
			Name:        serverVariablesConsistentEnumOption,
			Description: "Require every server variable to define an enum.",
			Default:     false,
		}},
		Check: checkServerVariablesConsistent,
	})
}

// checkServerVariablesConsistent inspects every server object of the document.
func checkServerVariablesConsistent(call *Call) []Finding {
	requireEnum := call.Bool(serverVariablesConsistentEnumOption)

	return Concat(
		OnDocument(func(call *Call) []Finding {
			return serverVariablesConsistentServers(oas.Get(call.Root, "servers"), "$.servers", requireEnum)
		}),
		PerPathItem(func(_ *Call, item oas.PathItem) []Finding {
			return serverVariablesConsistentServers(
				oas.Get(item.Node, "servers"),
				oas.Join(item.Path, "servers"),
				requireEnum,
			)
		}),
		PerOperation(func(call *Call, operation oas.Operation) []Finding {
			return slices.Concat(
				serverVariablesConsistentServers(
					oas.Get(operation.Node, "servers"), oas.Join(operation.Path, "servers"), requireEnum),
				serverVariablesConsistentLinks(call.Root, operation, requireEnum),
			)
		}),
	)(call)
}

func serverVariablesConsistentServers(list *yaml.Node, base string, requireEnum bool) []Finding {
	return seq.FlatMapPairs(oas.Items(list), func(i int, server *yaml.Node) []Finding {
		return serverVariablesConsistentServer(server, oas.Index(base, i), requireEnum)
	})
}

func serverVariablesConsistentLinks(root *yaml.Node, operation oas.Operation, requireEnum bool) []Finding {
	return seq.FlatMapSlice(oas.Responses(root, operation), func(response oas.Response) []Finding {
		return seq.FlatMapPairs(oas.Pairs(oas.Get(response.Node, "links")), func(name, link *yaml.Node) []Finding {
			path := oas.JoinAll(response.Path, "links", name.Value, "server")

			return serverVariablesConsistentServer(oas.Get(oas.Resolve(root, link), "server"), path, requireEnum)
		})
	})
}

func serverVariablesConsistentServer(server *yaml.Node, path string, requireEnum bool) []Finding {
	if !oas.IsMap(server) {
		return nil
	}

	variables := oas.Get(server, "variables")
	matches := regexp.MustCompile(serverVariablesConsistentPlaceholder).
		FindAllStringSubmatch(oas.StringAt(server, "url"), -1)

	return slices.Concat(
		seq.FlatMapSlice(matches, func(match []string) []Finding {
			return When(!oas.Has(variables, match[1]), func() Finding {
				return Finding{
					Node:    server,
					Path:    path,
					Message: "server url placeholder '{" + match[1] + "}' has no matching entry in variables",
				}
			})
		}),
		seq.FlatMapPairs(oas.Pairs(variables), func(name, variable *yaml.Node) []Finding {
			return serverVariablesConsistentVariable(
				name.Value,
				variable,
				oas.JoinAll(path, "variables", name.Value),
				requireEnum,
			)
		}),
	)
}

func serverVariablesConsistentVariable(name string, variable *yaml.Node, path string, requireEnum bool) []Finding {
	report := func(message string) []Finding {
		return []Finding{{Node: variable, Path: path, Message: message}}
	}

	defaultNode := oas.Get(variable, "default")
	if oas.Text(defaultNode) == "" {
		return report("server variable '" + name + "' must have a non-empty default")
	}

	enum := oas.Get(variable, "enum")

	switch {
	case enum == nil && requireEnum:
		return report("server variable '" + name + "' must define an enum")
	case enum == nil:
		return nil
	case oas.Empty(enum):
		return report("enum of server variable '" + name + "' must not be empty")
	case !slices.Contains(oas.Strings(enum), defaultNode.Value):
		return report("default of server variable '" + name + "' must be one of its enum values")
	default:
		return nil
	}
}
