package rules

import (
	"slices"
	"strconv"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// securityScopesDefinedOAuth2 is the security scheme type that declares scopes.
const securityScopesDefinedOAuth2 = "oauth2"

// securityScopesDefinedReportUnused is the option that reports unused scopes.
const securityScopesDefinedReportUnused = "reportUnused"

// SecurityScopesDefined returns the extra-security-scopes-defined rule.
//
// # What it checks
//
// Every security requirement, at the top level and on operations, must refer
// to a scheme defined in components.securitySchemes, and every scope that a
// requirement lists for an oauth2 scheme must be declared in at least one flow
// of that scheme. With reportUnused, scopes that an oauth2 scheme declares but
// no requirement uses are reported as well.
//
// # Why
//
// A requirement that names an unknown scheme or scope can never be satisfied,
// and tooling either rejects the document or silently drops the protection.
// Unused scope declarations are dead documentation.
//
// # Where it comes from
//
// The IBM API Handbook ruleset (security-scope consistency) and Redocly CLI
// (security-defined and its scope check). The two rows are merged.
//
// # Options
//
//   - reportUnused (bool, default false): also report oauth2 scopes that are
//     declared and never used by a requirement.
//
// # Bad
//
//	security:
//	  - oauth:
//	      - admin
//	components:
//	  securitySchemes:
//	    oauth:
//	      type: oauth2
//	      flows:
//	        implicit:
//	          authorizationUrl: https://example.com/auth
//	          scopes:
//	            read: Read access
//
// # Good
//
//	security:
//	  - oauth:
//	      - read
//	components:
//	  securitySchemes:
//	    oauth:
//	      type: oauth2
//	      flows:
//	        implicit:
//	          authorizationUrl: https://example.com/auth
//	          scopes:
//	            read: Read access
//
// # Known false positives / limitations
//
// Scopes listed for schemes that are not oauth2 (apiKey, http, openIdConnect)
// are not checked, because the document cannot declare them. Only the
// components.securitySchemes of the document itself are consulted.
func SecurityScopesDefined() Rule {
	return New(Spec{
		Name:        "security-scopes-defined",
		Description: "Security requirements must use defined schemes and declared scopes.",
		HowToFix:    "Define the scheme in components.securitySchemes or declare the scope in the oauth2 flows.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        securityScopesDefinedReportUnused,
				Description: "Also report declared scopes that no requirement uses.",
				Default:     false,
			},
		},
		Check: OnDocument(checkSecurityScopesDefined),
	})
}

// securityScopesDefinedUse is one scheme entry of a security requirement.
type securityScopesDefinedUse struct {
	name   *yaml.Node
	scopes *yaml.Node
	path   string
}

// checkSecurityScopesDefined validates every security requirement.
func checkSecurityScopesDefined(call *Call) []Finding {
	schemes := oas.Dig(call.Root, "components", "securitySchemes")
	uses := slices.Concat(
		securityScopesDefinedUses(oas.Get(call.Root, "security"), "$.security"),
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []securityScopesDefinedUse {
			return securityScopesDefinedUses(oas.Get(operation.Node, "security"), oas.Join(operation.Path, "security"))
		}),
	)

	return slices.Concat(
		seq.FlatMapSlice(uses, func(use securityScopesDefinedUse) []Finding {
			return securityScopesDefinedUndeclared(schemes, use)
		}),
		securityScopesDefinedUnused(call, schemes, uses),
	)
}

// securityScopesDefinedUses lists the scheme entries of one list of security requirement objects.
func securityScopesDefinedUses(list *yaml.Node, path string) []securityScopesDefinedUse {
	return seq.FlatMapPairs(oas.Items(list), func(i int, requirement *yaml.Node) []securityScopesDefinedUse {
		return seq.FlatMapPairs(oas.Pairs(requirement), func(name, scopes *yaml.Node) []securityScopesDefinedUse {
			return []securityScopesDefinedUse{{
				name:   name,
				scopes: scopes,
				path:   oas.Join(oas.Index(path, i), name.Value),
			}}
		})
	})
}

// securityScopesDefinedUndeclared checks one scheme entry of a requirement.
func securityScopesDefinedUndeclared(schemes *yaml.Node, use securityScopesDefinedUse) []Finding {
	scheme := oas.Get(schemes, use.name.Value)
	if scheme == nil {
		return One(Finding{
			Node:    use.name,
			Path:    use.path,
			Message: "security scheme " + use.name.Value + " is not defined in components.securitySchemes",
		})
	}

	if oas.StringAt(scheme, "type") != securityScopesDefinedOAuth2 {
		return nil
	}

	declared := securityScopesDefinedDeclared(scheme)

	return seq.FlatMapPairs(oas.Items(use.scopes), func(i int, scope *yaml.Node) []Finding {
		return When(!slices.Contains(declared, oas.Text(scope)), func() Finding {
			return Finding{
				Node: scope,
				Path: oas.Index(use.path, i),
				Message: "scope " + strconv.Quote(
					oas.Text(scope),
				) + " is not declared by security scheme " + use.name.Value,
			}
		})
	})
}

// securityScopesDefinedDeclared returns the scopes declared by the flows of a scheme.
func securityScopesDefinedDeclared(scheme *yaml.Node) []string {
	return seq.FlatMapPairs(oas.Pairs(oas.Get(scheme, "flows")), func(_, value *yaml.Node) []string {
		return oas.Keys(oas.Get(value, "scopes"))
	})
}

// securityScopesDefinedUnused reports declared scopes that no requirement uses.
func securityScopesDefinedUnused(call *Call, schemes *yaml.Node, uses []securityScopesDefinedUse) []Finding {
	if !call.Bool(securityScopesDefinedReportUnused) {
		return nil
	}

	return seq.FlatMapPairs(oas.Pairs(schemes), func(name, scheme *yaml.Node) []Finding {
		if oas.StringAt(scheme, "type") != securityScopesDefinedOAuth2 {
			return nil
		}

		return seq.FlatMapPairs(oas.Pairs(oas.Get(scheme, "flows")), func(flow, value *yaml.Node) []Finding {
			return seq.FlatMapPairs(oas.Pairs(oas.Get(value, "scopes")), func(scope, _ *yaml.Node) []Finding {
				used := slices.ContainsFunc(uses, func(use securityScopesDefinedUse) bool {
					return use.name.Value == name.Value && slices.ContainsFunc(
						seq.FlatMapPairs(oas.Items(use.scopes), func(_ int, item *yaml.Node) []string {
							return []string{oas.Text(item)}
						}),
						func(text string) bool { return text == scope.Value },
					)
				})

				return When(!used, func() Finding {
					return Finding{
						Node: scope,
						Path: oas.JoinAll(
							"$.components.securitySchemes",
							name.Value,
							"flows",
							flow.Value,
							"scopes",
							scope.Value,
						),
						Message: "scope " + strconv.Quote(
							scope.Value,
						) + " of security scheme " + name.Value + " is never used",
					}
				})
			})
		})
	})
}
