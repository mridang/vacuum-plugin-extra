package rules

import (
	"slices"
	"strings"
	"unicode"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// booleanParameterPrefixOption is the name of the option listing the prefixes.
const booleanParameterPrefixOption = "prefixes"

// BooleanParameterPrefix returns the extra-boolean-parameter-prefix rule.
//
// # What it checks
//
// The name of a parameter whose schema has type boolean must start with one of
// the configured prefixes, followed by an upper-case letter, an underscore or
// a hyphen, or be exactly the prefix. This keeps flags readable as questions:
// isActive, has_children and is-visible read well, while active and enabled do
// not say what the flag answers. The schema of a parameter is followed through
// a local $ref, and parameters declared in components and path items are
// checked once.
//
// # Where it comes from
//
// Redocly CLI's boolean-parameter-prefixes rule.
//
// # Options
//
//   - prefixes: list of allowed prefixes. Default: is, has. An empty list
//     disables the rule.
//
// # Bad
//
//	parameters:
//	  - name: active
//	    in: query
//	    schema:
//	      type: boolean
//
// # Good
//
//	parameters:
//	  - name: isActive
//	    in: query
//	    schema:
//	      type: boolean
//
// # Known false positives
//
// Names that are established by an external protocol (for example a "dryRun"
// query flag) are flagged; add a suppression or extend the prefixes. Matching
// is case sensitive, so "IsActive" is rejected.
func BooleanParameterPrefix() Rule {
	return New(Spec{
		Name:        "boolean-parameter-prefix",
		Description: "Boolean parameter names must start with one of the configured prefixes.",
		HowToFix:    "Rename the parameter so that it starts with an allowed prefix such as is or has.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        booleanParameterPrefixOption,
			Description: "Allowed name prefixes for boolean parameters.",
			Default:     []string{"is", "has"},
		}},
		Check: OnDocument(checkBooleanParameterPrefix),
	})
}

func checkBooleanParameterPrefix(call *Call) []Finding {
	prefixes := call.Strings(booleanParameterPrefixOption)
	if len(prefixes) == 0 {
		return nil
	}

	return seq.FlatMapSlice(booleanParameterPrefixAll(call.Root), func(parameter oas.Parameter) []Finding {
		name := parameter.Name()

		return When(
			name != "" &&
				booleanParameterPrefixIsBoolean(call.Root, parameter.Node) &&
				!booleanParameterPrefixMatches(name, prefixes),
			func() Finding {
				return Finding{
					Node:    oas.Get(parameter.Node, "name"),
					Path:    oas.Join(parameter.Path, "name"),
					Message: "boolean parameter " + name + " must start with one of: " + strings.Join(prefixes, ", "),
				}
			},
		)
	})
}

func booleanParameterPrefixIsBoolean(root, parameter *yaml.Node) bool {
	return oas.HasType(oas.Resolve(root, oas.Get(parameter, "schema")), "boolean")
}

func booleanParameterPrefixMatches(name string, prefixes []string) bool {
	return slices.ContainsFunc(prefixes, func(prefix string) bool {
		rest, ok := strings.CutPrefix(name, prefix)
		if !ok {
			return false
		}

		if rest == "" {
			return true
		}

		first := []rune(rest)[0]

		return unicode.IsUpper(first) || first == '_' || first == '-'
	})
}

func booleanParameterPrefixAll(root *yaml.Node) []oas.Parameter {
	return slices.Concat(
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "parameters")),
			func(key, value *yaml.Node) []oas.Parameter {
				return []oas.Parameter{{
					Node: oas.Resolve(root, value),
					Path: oas.JoinAll("$.components.parameters", key.Value),
				}}
			},
		),
		seq.FlatMap(oas.PathItems(root), func(item oas.PathItem) []oas.Parameter {
			return seq.FlatMapPairs(
				oas.Items(oas.Get(item.Node, "parameters")),
				func(i int, value *yaml.Node) []oas.Parameter {
					return []oas.Parameter{{
						Node: oas.Resolve(root, value),
						Path: oas.Index(oas.Join(item.Path, "parameters"), i),
					}}
				},
			)
		}),
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []oas.Parameter {
			return oas.OwnParameters(root, operation)
		}),
	)
}
