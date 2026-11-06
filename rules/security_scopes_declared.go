package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// securityScopesDeclaredBearer names HTTP bearer authentication.
const securityScopesDeclaredBearer = "bearer"

// SecurityScopesDeclared returns the extra-security-scopes-declared rule.
//
// # What it checks
//
// Three checks on the security of the API, for OpenAPI 3 documents:
//
//   - components.securitySchemes must contain at least one allowed scheme.
//   - Every scheme must be of an allowed kind. By default these are oauth2 and
//     HTTP bearer authentication; any other type (apiKey, basic, openIdConnect,
//     mutualTLS) is reported.
//   - Every scope listed in a security requirement, at the document level or in
//     an operation, must be declared in a flow of the referenced OAuth2 scheme.
//
// # Why
//
// Scopes are what lets a token be limited to the operations it needs. A scope
// that is used but never declared cannot be requested by clients, so the
// operation is effectively unusable or is protected by nothing. Restricting the
// scheme types keeps the API on token based authentication.
//
// # Where it comes from
//
// Zally (SecureAllEndpointsRule: secure endpoints with OAuth2 or Bearer
// authentication, and used scopes must be specified in the security definition).
//
// # Options
//
//   - allowedTypes: the allowed scheme kinds. Use oauth2 for OAuth2 and bearer
//     for HTTP bearer authentication; any other value is matched against the
//     type of the scheme. Default: oauth2, bearer.
//
// # Bad
//
//	components:
//	  securitySchemes:
//	    oauth:
//	      type: oauth2
//	      flows:
//	        clientCredentials:
//	          tokenUrl: https://auth.example.com/token
//	          scopes:
//	            pets.read: read pets
//	paths:
//	  /pets:
//	    get:
//	      security:
//	        - oauth: [pets.write]
//
// # Good
//
//	components:
//	  securitySchemes:
//	    oauth:
//	      type: oauth2
//	      flows:
//	        clientCredentials:
//	          tokenUrl: https://auth.example.com/token
//	          scopes:
//	            pets.read: read pets
//	paths:
//	  /pets:
//	    get:
//	      security:
//	        - oauth: [pets.read]
//
// # Known false positives / limitations
//
// OpenAPI 2.0 documents (securityDefinitions) are not checked. Scopes of
// non-OAuth2 schemes are not validated, since they are free form. Security
// schemes used in requirements but missing from the components are reported by
// other rules.
func SecurityScopesDeclared() Rule {
	return New(Spec{
		Name:        "security-scopes-declared",
		Description: "Use OAuth2 or bearer security schemes and declare every scope that is used.",
		HowToFix:    "Define an OAuth2 or bearer scheme, remove other scheme types and declare each used scope in the OAuth2 flows.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Formats:     []string{formatOAS3},
		Options: []Option{{
			Name:        "allowedTypes",
			Description: "Allowed scheme kinds: oauth2, bearer or any other security scheme type.",
			Default:     []string{keywordOAuth2, securityScopesDeclaredBearer},
		}},
		Check: OnDocument(checkSecurityScopesDeclared),
	})
}

// securityScopesDeclaredScheme is a named security scheme with its resolved definition.
type securityScopesDeclaredScheme struct {
	name   *yaml.Node
	scheme *yaml.Node
}

// securityScopesDeclaredOAuth is an OAuth2 scheme with the scopes its flows declare.
type securityScopesDeclaredOAuth struct {
	name   string
	scopes []string
}

func checkSecurityScopesDeclared(call *Call) []Finding {
	allowed := call.Strings("allowedTypes")
	schemes := oas.Dig(call.Root, "components", "securitySchemes")
	wanted := strings.Join(allowed, " or ")
	entries := seq.FlatMapPairs(oas.Pairs(schemes), func(name, entry *yaml.Node) []securityScopesDeclaredScheme {
		return []securityScopesDeclaredScheme{{name: name, scheme: oas.Resolve(call.Root, entry)}}
	})
	valid := slices.ContainsFunc(entries, func(entry securityScopesDeclaredScheme) bool {
		return securityScopesDeclaredKind(entry.scheme, allowed)
	})

	return slices.Concat(
		seq.FlatMapSlice(entries, func(entry securityScopesDeclaredScheme) []Finding {
			return When(!securityScopesDeclaredKind(entry.scheme, allowed), func() Finding {
				return Finding{
					Node: entry.name,
					Path: oas.JoinAll("$.components.securitySchemes", entry.name.Value),
					Message: "security scheme '" + entry.name.Value + "' of type " + oas.StringAt(
						entry.scheme,
						"type",
					) + " is not allowed, use " + wanted,
				}
			})
		}),
		When(!valid && len(allowed) > 0, func() Finding {
			return Finding{
				Node:    schemes,
				Path:    "$.components.securitySchemes",
				Message: "the API must define a security scheme of type " + wanted,
			}
		}),
		securityScopesDeclaredUndeclared(call, entries),
	)
}

// securityScopesDeclaredKind reports whether the scheme is of an allowed kind.
func securityScopesDeclaredKind(scheme *yaml.Node, allowed []string) bool {
	kind := oas.StringAt(scheme, "type")
	if kind == "http" && Fold(oas.StringAt(scheme, "scheme")) == "bearer" {
		return slices.Contains(allowed, securityScopesDeclaredBearer)
	}

	return slices.Contains(allowed, kind)
}

// securityScopesDeclaredDeclared returns the scopes declared by the flows of an
// OAuth2 scheme.
func securityScopesDeclaredDeclared(scheme *yaml.Node) []string {
	return seq.FlatMapPairs(oas.Pairs(oas.Get(scheme, "flows")), func(_, flow *yaml.Node) []string {
		return seq.FlatMapPairs(oas.Pairs(oas.Get(flow, "scopes")), func(scope, _ *yaml.Node) []string {
			return []string{scope.Value}
		})
	})
}

// securityScopesDeclaredUndeclared reports used scopes missing from OAuth2
// schemes, in the document level and operation level requirements.
func securityScopesDeclaredUndeclared(call *Call, entries []securityScopesDeclaredScheme) []Finding {
	declared := seq.FlatMapSlice(entries, func(entry securityScopesDeclaredScheme) []securityScopesDeclaredOAuth {
		if oas.StringAt(entry.scheme, "type") != "oauth2" {
			return nil
		}

		return []securityScopesDeclaredOAuth{{
			name:   entry.name.Value,
			scopes: securityScopesDeclaredDeclared(entry.scheme),
		}}
	})

	return slices.Concat(
		securityScopesDeclaredRequirements(oas.Get(call.Root, "security"), "$.security", declared),
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
			return securityScopesDeclaredRequirements(
				oas.Get(operation.Node, "security"), oas.Join(operation.Path, "security"), declared)
		}),
	)
}

// securityScopesDeclaredRequirements checks one list of security requirements.
func securityScopesDeclaredRequirements(
	requirements *yaml.Node,
	base string,
	declared []securityScopesDeclaredOAuth,
) []Finding {
	return seq.FlatMapPairs(oas.Items(requirements), func(i int, requirement *yaml.Node) []Finding {
		return seq.FlatMapPairs(oas.Pairs(requirement), func(name, scopes *yaml.Node) []Finding {
			index := slices.IndexFunc(declared, func(scheme securityScopesDeclaredOAuth) bool {
				return scheme.name == name.Value
			})
			if index < 0 {
				return nil
			}

			known := declared[index].scopes

			return seq.FlatMapPairs(oas.Items(scopes), func(j int, scope *yaml.Node) []Finding {
				return When(oas.IsScalar(scope) && !slices.Contains(known, scope.Value), func() Finding {
					return Finding{
						Node:    scope,
						Path:    oas.Index(oas.Join(oas.Index(base, i), name.Value), j),
						Message: "scope '" + scope.Value + "' is not declared in the flows of security scheme '" + name.Value + "'",
					}
				})
			})
		})
	})
}
