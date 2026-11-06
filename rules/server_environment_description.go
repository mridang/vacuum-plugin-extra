package rules

import (
	"regexp"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// serverEnvironmentDescriptionDefault is the pattern of the original rule: an
// unanchored, case-insensitive search for an environment name anywhere in the
// description, so staging, preproduction and nextgen-test all match.
const serverEnvironmentDescriptionDefault = `(?i)(local|sandbox|alpha|beta|test|testing|stag|staging|prod|production|next|preprod|preproduction)`

// serverEnvironmentDescriptionRequire is the name of the option that also
// reports servers without a description.
const serverEnvironmentDescriptionRequire = "requireDescription"

// ServerEnvironmentDescription returns the extra-server-environment-description
// rule.
//
// # What it checks
//
// The description of every entry of the root servers array must match the
// pattern, which by default looks for one of the names local, sandbox, alpha,
// beta, test, testing, stag, staging, prod, production, next, preprod or
// preproduction. The search is case-insensitive and not anchored to word
// boundaries, exactly as in the original, so a description such as "Contested
// device server" matches test. The rule only applies to OpenAPI 3 documents.
//
// A server that has no description is not reported by default: the original
// rule applies its pattern to the description field only when it is present.
// Set requireDescription to also report a server without one.
//
// When an API lists several servers, the description is the only thing that
// tells a reader which one is production and which is a sandbox. Naming the
// environment avoids calls against the wrong one.
//
// # Where it comes from
//
// The Spectral OWASP ruleset, rule owasp:api9:2023-inventory-environment, which
// runs given $.servers.* with the pattern function on the description field
// and error severity.
//
// # Options
//
//   - pattern (string, default (?i)(local|sandbox|alpha|beta|test|testing|stag|
//     staging|prod|production|next|preprod|preproduction)): regular expression
//     searched in the description. An empty string disables the rule, and a
//     pattern that is not a valid regular expression disables it too.
//   - requireDescription (bool, default false): also report servers without a
//     description.
//
// # Bad
//
//	servers:
//	  - url: https://api.example.com
//	    description: The main server
//
// # Good
//
//	servers:
//	  - url: https://api.example.com
//	    description: Production server
//
// # Known false positives / limitations
//
// Only the root servers array is inspected, not servers declared on a path item
// or an operation, as in the original. Descriptions in another language than
// English are reported unless the pattern is changed. The pattern uses the Go
// regexp syntax (RE2).
func ServerEnvironmentDescription() Rule {
	return New(Spec{
		Name:        "server-environment-description",
		Description: "Every server description must name the environment of the server.",
		HowToFix:    "Add a description to the server that names its environment, such as Production or Staging.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeStyle,
		Formats:     []string{formatOAS3},
		Options: []Option{
			{
				Name:        literalPattern,
				Description: "Regular expression the server description must match.",
				Default:     serverEnvironmentDescriptionDefault,
			},
			{
				Name:        serverEnvironmentDescriptionRequire,
				Description: "Also report servers without a description.",
				Default:     false,
			},
		},
		Check: OnDocument(checkServerEnvironmentDescription),
	})
}

// checkServerEnvironmentDescription reports server descriptions without an environment name.
func checkServerEnvironmentDescription(call *Call) []Finding {
	pattern := call.String("pattern")
	if pattern == "" {
		return nil
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	return seq.FlatMapPairs(oas.Items(oas.Get(call.Root, "servers")), func(i int, server *yaml.Node) []Finding {
		key, description := oas.Entry(server, "description")
		skipped := key == nil && !call.Bool(serverEnvironmentDescriptionRequire)
		named := oas.IsScalar(description) && compiled.MatchString(description.Value)

		return When(!skipped && !named, func() Finding {
			path := oas.Index("$.servers", i)
			if key == nil {
				return Finding{
					Node:    server,
					Path:    path,
					Message: "the server description must name the environment of the server",
				}
			}

			return Finding{
				Node:    description,
				Path:    oas.Join(path, key.Value),
				Message: "the server description must name the environment of the server",
			}
		})
	})
}
