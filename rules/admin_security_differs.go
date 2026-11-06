package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	adminSecurityDiffersOption  = "adminUrl"
	adminSecurityDiffersDefault = "/admin"
)

// AdminSecurityDiffers returns the extra-admin-security-differs rule.
//
// # What it checks
//
// An operation-level security requirement on an administrative path must not be
// identical to a security requirement used by any non-administrative
// operation. A path is administrative when its route contains the configured
// admin URL fragment. Two requirements are identical when they name the same
// security schemes with the same scopes, regardless of order. Operations
// without an operation-level security list are ignored.
//
// # Why
//
// When administrators and ordinary users authenticate with the same scheme and
// scopes, any ordinary credential can reach the administrative surface, which
// makes privilege escalation a matter of a path change. Admin endpoints should
// require a distinct scheme or a distinct, dedicated scope.
//
// # Where it comes from
//
// The Spectral OWASP ruleset (the differentSecuritySchemes function behind the
// owasp:api5:2023-admin-security-unique rule).
//
// # Options
//
//   - adminUrl (string, default "/admin"): a route containing this text is
//     treated as administrative.
//
// # Bad
//
//	paths:
//	  /users:
//	    get:
//	      security:
//	        - oauth: [read]
//	  /admin/users:
//	    get:
//	      security:
//	        - oauth: [read]
//
// # Good
//
//	paths:
//	  /users:
//	    get:
//	      security:
//	        - oauth: [read]
//	  /admin/users:
//	    get:
//	      security:
//	        - oauth: [admin]
//
// # Known false positives and limitations
//
// Only operation-level security is compared; a global security list inherited
// by an operation is not considered. Matching the admin URL is a plain
// substring test, so a route such as /administrators is also administrative.
func AdminSecurityDiffers() Rule {
	return New(Spec{
		Name:        "admin-security-differs",
		Description: "Admin endpoints must not share a security requirement with non-admin endpoints.",
		HowToFix:    "Give admin operations a dedicated security scheme or scope that no non-admin operation uses.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        adminSecurityDiffersOption,
				Description: "Text that marks a route as administrative.",
				Default:     adminSecurityDiffersDefault,
			},
		},
		Check: OnDocument(checkAdminSecurityDiffers),
	})
}

type adminSecurityDiffersEntry struct {
	admin bool
	route string
	hash  string
	path  string
	node  *yaml.Node
}

// checkAdminSecurityDiffers reports admin requirements that a non-admin operation also uses.
func checkAdminSecurityDiffers(call *Call) []Finding {
	adminURL := call.String(adminSecurityDiffersOption)
	if adminURL == "" {
		return nil
	}

	entries := seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []adminSecurityDiffersEntry {
		return seq.FlatMapPairs(
			oas.Items(oas.Get(operation.Node, "security")),
			func(i int, requirement *yaml.Node) []adminSecurityDiffersEntry {
				if !oas.IsMap(requirement) {
					return nil
				}

				return []adminSecurityDiffersEntry{{
					admin: strings.Contains(operation.Route, adminURL),
					route: operation.Route,
					hash:  adminSecurityDiffersHash(requirement),
					path:  oas.Index(oas.Join(operation.Path, "security"), i),
					node:  requirement,
				}}
			},
		)
	})
	shared := seq.Map(slices.Values(seq.Filter(slices.Values(entries), func(entry adminSecurityDiffersEntry) bool {
		return !entry.admin
	})), func(entry adminSecurityDiffersEntry) string {
		return entry.hash
	})

	return seq.FlatMapSlice(entries, func(entry adminSecurityDiffersEntry) []Finding {
		return When(entry.admin && slices.Contains(shared, entry.hash), func() Finding {
			return Finding{
				Node:    entry.node,
				Path:    entry.path,
				Message: "admin endpoint " + entry.route + " has the same security requirement as a non-admin endpoint",
			}
		})
	})
}

// adminSecurityDiffersHash builds an order independent identity of a security requirement.
func adminSecurityDiffersHash(requirement *yaml.Node) string {
	parts := seq.FlatMapPairs(oas.Pairs(requirement), func(name, scopes *yaml.Node) []string {
		return []string{name.Value + "[" + strings.Join(slices.Sorted(slices.Values(oas.Strings(scopes))), ",") + "]"}
	})

	return strings.Join(slices.Sorted(slices.Values(parts)), ";")
}
