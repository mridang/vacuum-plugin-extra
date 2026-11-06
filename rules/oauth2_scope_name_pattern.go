package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// oauth2ScopeNamePatternOAuth2 is the security scheme type that declares scopes.
const oauth2ScopeNamePatternOAuth2 = "oauth2"

const (
	// oauth2ScopeNamePatternPattern is the option with the scope name expression.
	oauth2ScopeNamePatternPattern = "pattern"
	// oauth2ScopeNamePatternAllowed is the option with exempt scope names.
	oauth2ScopeNamePatternAllowed = "allowed"
	// oauth2ScopeNamePatternWhitelist is the option with the paths exempt from the scope check.
	oauth2ScopeNamePatternWhitelist = "pathWhitelist"
	// oauth2ScopeNamePatternDefault is the scope_regex of the original rule. It
	// is matched against the whole scope name.
	oauth2ScopeNamePatternDefault = `^(uid)|(([a-z-]+\.){1,2}(read|write))$`
)

// OAuth2ScopeNamePattern returns the extra-oauth2-scope-name-pattern rule.
//
// # What it checks
//
// Two checks, both from the same Zally rule.
//
// First, every scope declared by an oauth2 security scheme (under
// components.securitySchemes.*.flows.*.scopes) must match, in full, the
// configured regular expression. The default accepts the scope uid and
// scopes of the form <app>.read, <app>.write and <area>.<app>.read or .write,
// where the segments are lower-case letters and hyphens. Scope names listed in
// the allowed option are exempt.
//
// Second, every operation must be secured by scopes. The security requirements
// of the operation apply, or those of the document when the operation has none
// (an empty security array counts as none). The operation is reported when:
//
//   - there is no security requirement at all (Endpoint is not secured by
//     scope(s)),
//   - a requirement names a scheme that is not in components.securitySchemes,
//   - a requirement names an oauth2 scheme and lists no scopes, or
//   - it lists a scope that no flow of that scheme declares.
//
// Schemes that are not oauth2 are not checked for scopes. Paths that match one
// of the pathWhitelist expressions anywhere in the route are skipped by the
// second check.
//
// # Why
//
// Consistent scope names make permissions predictable for API consumers and let
// gateways map scopes to resources mechanically, and an endpoint without scopes
// cannot be granted or denied individually.
//
// # Where it comes from
//
// Zally rule 105, "Secure All Endpoints With Scopes"
// (SecureAllEndpointsWithScopesRule), with its scope_regex and path_whitelist
// settings and its two checks checkDefinedScopeFormats and
// checkOperationsAreScoped.
//
// # Options
//
//   - pattern (string, default ^(uid)|(([a-z-]+\.){1,2}(read|write))$): the
//     expression every scope name must match in full. An invalid expression
//     disables the scope name check.
//   - allowed (list of strings, default empty): scope names that are always
//     accepted by the name check.
//   - pathWhitelist (list of strings, default empty): regular expressions; an
//     operation whose route contains a match is not checked for scopes. An
//     invalid expression is ignored.
//
// # Bad
//
//	components:
//	  securitySchemes:
//	    oauth:
//	      type: oauth2
//	      flows:
//	        implicit:
//	          authorizationUrl: https://example.com/auth
//	          scopes:
//	            admin: Administrator
//
// # Good
//
//	components:
//	  securitySchemes:
//	    oauth:
//	      type: oauth2
//	      flows:
//	        implicit:
//	          authorizationUrl: https://example.com/auth
//	          scopes:
//	            pets.read: Read pets
//
// # Known false positives / limitations
//
// Scope names used in requirements are also checked by
// extra-security-scopes-defined; the two rules overlap by design. Expressions
// use the Go regexp syntax (RE2). Only the four standard flows are read, as in
// the original.
func OAuth2ScopeNamePattern() Rule {
	return New(Spec{
		Name:        "oauth2-scope-name-pattern",
		Description: "OAuth2 scope names must match the configured pattern and every endpoint must be secured by scopes.",
		HowToFix: "Rename the scope so that it matches the configured pattern, and secure the endpoint with " +
			"defined OAuth2 scopes.",
		Severity: model.SeverityError,
		Given:    "$",
		Type:     TypeStyle,
		Options: []Option{
			{
				Name:        oauth2ScopeNamePatternPattern,
				Description: "Regular expression every scope name must match in full.",
				Default:     oauth2ScopeNamePatternDefault,
			},
			{
				Name:        oauth2ScopeNamePatternAllowed,
				Description: "Scope names that are always accepted.",
				Default:     []string{},
			},
			{
				Name:        oauth2ScopeNamePatternWhitelist,
				Description: "Regular expressions of routes that are not checked for scopes.",
				Default:     []string{},
			},
		},
		Check: Concat(OnDocument(oauth2ScopeNamePatternNames), OnDocument(oauth2ScopeNamePatternEndpoints)),
	})
}

// oauth2ScopeNamePatternNames reports scope names that do not match the pattern.
func oauth2ScopeNamePatternNames(call *Call) []Finding {
	expression, err := regexp.Compile(`\A(?:` + call.String(oauth2ScopeNamePatternPattern) + `)\z`)
	if err != nil {
		return nil
	}

	allowed := call.Strings(oauth2ScopeNamePatternAllowed)

	return seq.FlatMapPairs(
		oas.Pairs(oas.Dig(call.Root, "components", "securitySchemes")),
		func(key, scheme *yaml.Node) []Finding {
			base := oas.JoinAll("$.components.securitySchemes", key.Value)

			return seq.FlatMapSlice(
				oauth2ScopeNamePatternScopeMaps(oas.Resolve(call.Root, scheme), base),
				func(scopes oauth2ScopeNamePatternScopes) []Finding {
					return oauth2ScopeNamePatternCheckScopes(scopes, expression, allowed)
				},
			)
		},
	)
}

// oauth2ScopeNamePatternScopes is one scopes map with its location.
type oauth2ScopeNamePatternScopes struct {
	node *yaml.Node
	path string
}

// oauth2ScopeNamePatternScopeMaps lists the scopes maps of an oauth2 scheme.
func oauth2ScopeNamePatternScopeMaps(scheme *yaml.Node, base string) []oauth2ScopeNamePatternScopes {
	if oas.StringAt(scheme, "type") != oauth2ScopeNamePatternOAuth2 {
		return nil
	}

	return seq.FlatMapPairs(
		oas.Pairs(oas.Get(scheme, "flows")),
		func(flow, value *yaml.Node) []oauth2ScopeNamePatternScopes {
			path := oas.JoinAll(base, "flows", flow.Value, "scopes")

			return []oauth2ScopeNamePatternScopes{{node: oas.Get(value, "scopes"), path: path}}
		},
	)
}

// oauth2ScopeNamePatternCheckScopes checks every name of one scopes map.
func oauth2ScopeNamePatternCheckScopes(
	scopes oauth2ScopeNamePatternScopes,
	expression *regexp.Regexp,
	allowed []string,
) []Finding {
	return seq.FlatMapPairs(oas.Pairs(scopes.node), func(name, _ *yaml.Node) []Finding {
		invalid := !expression.MatchString(name.Value) && !slices.Contains(allowed, name.Value)

		return When(invalid, func() Finding {
			return Finding{
				Node: name,
				Path: oas.Join(scopes.path, name.Value),
				Message: "scope '" + name.Value + "' does not match regex '" + oauth2ScopeNamePatternSource(
					expression,
				) + "'",
			}
		})
	})
}

// oauth2ScopeNamePatternSource returns the configured text of a full-match expression.
func oauth2ScopeNamePatternSource(expression *regexp.Regexp) string {
	source := expression.String()

	return source[len(`\A(?:`) : len(source)-len(`)\z`)]
}

// oauth2ScopeNamePatternEndpoints reports operations that are not secured by defined scopes.
func oauth2ScopeNamePatternEndpoints(call *Call) []Finding {
	whitelist := oauth2ScopeNamePatternCompileAll(call.Strings(oauth2ScopeNamePatternWhitelist))
	schemes := oas.Dig(call.Root, "components", "securitySchemes")

	return seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
		if slices.ContainsFunc(whitelist, func(e *regexp.Regexp) bool { return e.MatchString(operation.Route) }) {
			return nil
		}

		return oauth2ScopeNamePatternOperation(call.Root, schemes, operation)
	})
}

// oauth2ScopeNamePatternCompileAll compiles the valid expressions of a list.
func oauth2ScopeNamePatternCompileAll(sources []string) []*regexp.Regexp {
	return seq.FlatMapSlice(sources, func(source string) []*regexp.Regexp {
		expression, err := regexp.Compile(source)
		if err != nil {
			return nil
		}

		return []*regexp.Regexp{expression}
	})
}

// oauth2ScopeNamePatternOperation checks the effective security of one operation.
func oauth2ScopeNamePatternOperation(root, schemes *yaml.Node, operation oas.Operation) []Finding {
	node, path := oauth2ScopeNamePatternLocation(operation)
	requirements := oauth2ScopeNamePatternRequirements(root, operation)

	if oas.Empty(requirements) {
		return One(Finding{Node: node, Path: path, Message: "Endpoint is not secured by scope(s)"})
	}

	return seq.FlatMapPairs(oas.Items(requirements), func(_ int, requirement *yaml.Node) []Finding {
		return seq.FlatMapPairs(oas.Pairs(requirement), func(name, scopes *yaml.Node) []Finding {
			message := oauth2ScopeNamePatternProblem(root, schemes, name.Value, scopes)

			return When(message != "", func() Finding {
				return Finding{Node: node, Path: path, Message: message}
			})
		})
	})
}

// oauth2ScopeNamePatternLocation returns where the security of an operation is
// reported: its security entry when present, and the operation otherwise.
func oauth2ScopeNamePatternLocation(operation oas.Operation) (*yaml.Node, string) {
	key, value := oas.Entry(operation.Node, "security")
	if key == nil {
		return operation.Node, operation.Path
	}

	return value, oas.Join(operation.Path, "security")
}

// oauth2ScopeNamePatternRequirements returns the security requirements of an
// operation, falling back to the document level ones.
func oauth2ScopeNamePatternRequirements(root *yaml.Node, operation oas.Operation) *yaml.Node {
	own := oas.Get(operation.Node, "security")
	if !oas.Empty(own) {
		return own
	}

	return oas.Get(root, "security")
}

// oauth2ScopeNamePatternProblem describes what is wrong with one requirement entry, or returns an empty string.
func oauth2ScopeNamePatternProblem(root, schemes *yaml.Node, name string, scopes *yaml.Node) string {
	scheme := oas.Resolve(root, oas.Get(schemes, name))
	if scheme == nil {
		return "Security scheme " + name + " not found"
	}

	if oas.StringAt(scheme, "type") != oauth2ScopeNamePatternOAuth2 {
		return ""
	}

	requested := oas.Strings(scopes)
	if len(requested) == 0 {
		return "Endpoint is not secured by OAuth2 scope(s)"
	}

	defined := oauth2ScopeNamePatternDefined(scheme)
	undefined := seq.Filter(slices.Values(requested), func(scope string) bool {
		return !slices.Contains(defined, scope)
	})

	if len(undefined) == 0 {
		return ""
	}

	return "Endpoint is secured by undefined OAuth2 scope(s): " + name + ":" + strings.Join(undefined, ", ")
}

// oauth2ScopeNamePatternDefined lists the scopes of the four standard flows.
func oauth2ScopeNamePatternDefined(scheme *yaml.Node) []string {
	flows := []string{"implicit", "password", "clientCredentials", "authorizationCode"}

	return seq.FlatMapSlice(flows, func(flow string) []string {
		return seq.FlatMapPairs(
			oas.Pairs(oas.Dig(scheme, "flows", flow, "scopes")),
			func(scope, _ *yaml.Node) []string {
				return []string{scope.Value}
			},
		)
	})
}
