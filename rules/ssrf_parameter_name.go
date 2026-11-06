package rules

import (
	"regexp"
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// ssrfParameterNameDefault is the notMatch pattern of the original rule. The
// trailing dollar applies to the whole group, so the name must be exactly
// callback or redirect, or end with _url, Url or -url. It is case-sensitive.
const ssrfParameterNameDefault = `(^(callback|redirect)|(_url|Url|-url))$`

// SSRFParameterName returns the extra-ssrf-parameter-name rule.
//
// # What it checks
//
// The name of every parameter declared directly on a path item or on an
// operation (not in components, and not through a $ref) must not match the
// pattern. By default the pattern matches the names callback and redirect and
// every name that ends with _url, -url or Url (return_url, callback-url,
// imageUrl). Matching is case-sensitive and, because the end anchor applies to
// the whole group, a name that merely starts with callback (callbackHandler)
// does not match. Matching parameters are reported.
//
// Parameters that make the server fetch or redirect to a caller-supplied URL
// are the entry point of server-side request forgery and open-redirect attacks.
// The rule points the reviewer at them so that the input is validated against an
// allow-list.
//
// # Where it comes from
//
// The Spectral OWASP ruleset, rule owasp:api7:2023-concerning-url-parameter,
// which runs given $.paths[*].parameters[*].name and
// $.paths[*][get,put,post,delete,options,head,patch,trace].parameters[*].name
// with the pattern function (notMatch) and information severity.
//
// # Options
//
//   - pattern (string, default (^(callback|redirect)|(_url|Url|-url))$):
//     regular expression a parameter name must not match. An empty string
//     disables the rule, and a pattern that is not a valid regular expression
//     disables it too.
//
// # Bad
//
//	paths:
//	  /fetch:
//	    get:
//	      parameters:
//	        - name: callback_url
//	          in: query
//	          schema:
//	            type: string
//
// # Good
//
//	paths:
//	  /fetch:
//	    get:
//	      parameters:
//	        - name: resourceId
//	          in: query
//	          schema:
//	            type: string
//
// # Known false positives
//
// Parameters under components.parameters and parameters given as a $ref are
// not inspected, as in the original. A parameter that is a URL by design (a
// webhook registration, for example) is reported even when the server validates
// it. The rule is a prompt to review
// such inputs, so suppress it for the parameters that are reviewed.
func SSRFParameterName() Rule {
	return New(Spec{
		Name:        "ssrf-parameter-name",
		Description: "Parameter names must not suggest a caller-supplied URL.",
		HowToFix:    "Review the parameter for server-side request forgery and rename or suppress it.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        literalPattern,
				Description: "Regular expression a parameter name must not match.",
				Default:     ssrfParameterNameDefault,
			},
		},
		Check: OnDocument(checkSSRFParameterName),
	})
}

// checkSSRFParameterName reports parameter names that match the pattern.
func checkSSRFParameterName(call *Call) []Finding {
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
			return ssrfParameterNameInList(compiled, item.Node, item.Path)
		}),
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
			return ssrfParameterNameInList(compiled, operation.Node, operation.Path)
		}),
	)
}

// ssrfParameterNameInList checks the parameters list of one path item or operation.
func ssrfParameterNameInList(pattern *regexp.Regexp, owner *yaml.Node, ownerPath string) []Finding {
	listPath := oas.Join(ownerPath, "parameters")

	return seq.FlatMapPairs(oas.Items(oas.Get(owner, "parameters")), func(i int, parameter *yaml.Node) []Finding {
		name := oas.Get(parameter, "name")

		return When(oas.IsScalar(name) && pattern.MatchString(name.Value), func() Finding {
			return Finding{
				Node: name,
				Path: oas.Join(oas.Index(listPath, i), "name"),
				Message: "parameter name " + name.Value +
					" suggests a caller-supplied URL and may allow server-side request forgery",
			}
		})
	})
}
