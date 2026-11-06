package rules

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// OAuth2RefreshURL returns the extra-oauth2-refresh-url rule.
//
// # What it checks
//
// Every flow of an oauth2 security scheme except clientCredentials must define
// a refreshUrl. The flows examined are implicit, password and authorizationCode,
// found under components.securitySchemes.
//
// # Why
//
// Access tokens are short lived. Without a refresh endpoint clients cannot
// renew them and have to send the user through the authorization again.
// Client credentials flows have no user to bother, so a new token is simply
// requested again.
//
// # Where it comes from
//
// The Spectral OWASP ruleset, rule owasp:api2:2023-short-lived-access-tokens,
// which runs given
// $.components.securitySchemes[?(@ && @.type=="oauth2")].flows[?(@property != "clientCredentials")]
// and requires a truthy refreshUrl field, with error severity.
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  securitySchemes:
//	    oauth:
//	      type: oauth2
//	      flows:
//	        authorizationCode:
//	          authorizationUrl: https://auth.example.com/authorize
//	          tokenUrl: https://auth.example.com/token
//	          scopes: {}
//
// # Good
//
//	components:
//	  securitySchemes:
//	    oauth:
//	      type: oauth2
//	      flows:
//	        authorizationCode:
//	          authorizationUrl: https://auth.example.com/authorize
//	          tokenUrl: https://auth.example.com/token
//	          refreshUrl: https://auth.example.com/refresh
//	          scopes: {}
//
// # Known false positives / limitations
//
// A refresh endpoint that is the same as the token endpoint is still reported
// unless refreshUrl is set explicitly. An empty refreshUrl counts as missing
// (truthy semantics). Only schemes declared inline under components are read
// ($ref schemes are followed, as the original sees the dereferenced document).
func OAuth2RefreshURL() Rule {
	return New(Spec{
		Name:        "oauth2-refresh-url",
		Description: "OAuth2 flows other than clientCredentials must define a refreshUrl.",
		HowToFix:    "Add a refreshUrl to the OAuth2 flow.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeStyle,
		Formats:     []string{formatOAS3},
		Check:       OnDocument(checkOAuth2RefreshURL),
	})
}

// checkOAuth2RefreshURL reports flows without a refreshUrl.
func checkOAuth2RefreshURL(call *Call) []Finding {
	return seq.FlatMapPairs(
		oas.Pairs(oas.Dig(call.Root, "components", "securitySchemes")),
		func(name, entry *yaml.Node) []Finding {
			scheme := oas.Resolve(call.Root, entry)
			if oas.StringAt(scheme, "type") != keywordOAuth2 {
				return nil
			}

			base := oas.JoinAll("$.components.securitySchemes", name.Value, "flows")

			return seq.FlatMapPairs(oas.Pairs(oas.Get(scheme, "flows")), func(flow, definition *yaml.Node) []Finding {
				return When(
					flow.Value != "clientCredentials" && oas.StringAt(definition, "refreshUrl") == "",
					func() Finding {
						return Finding{
							Node:    flow,
							Path:    oas.Join(base, flow.Value),
							Message: "OAuth2 flow '" + flow.Value + "' of scheme '" + name.Value + "' must define a refreshUrl",
						}
					},
				)
			})
		},
	)
}
