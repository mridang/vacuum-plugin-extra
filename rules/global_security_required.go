package rules

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// GlobalSecurityRequired returns the extra-global-security-required rule.
//
// # What it checks
//
// The document root must declare a security property that is a non-empty list
// of security requirement objects. Declaring security globally makes every
// operation secure by default; operations that are public then opt out with an
// explicit empty list. A missing security property, an empty list, a value that
// is not a list, or a list entry that is not an object is reported.
//
// # Where it comes from
//
// rate-my-openapi, whose security checks require a root security array with
// at least one item.
//
// # Options
//
// None.
//
// # Bad
//
//	paths: {}
//
// # Good
//
//	security:
//	  - apiKey: []
//	paths: {}
//
// # Known false positives
//
// Documents in which every operation sets its own security are flagged even
// though no operation is unprotected. A root list holding only an empty
// requirement object ({}) is accepted because it is syntactically a valid
// non-empty list, although it means anonymous access.
func GlobalSecurityRequired() Rule {
	return New(Spec{
		Name:        "global-security-required",
		Description: "The root security property must exist and hold at least one security requirement.",
		HowToFix:    "Add a non-empty security list at the document root.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkGlobalSecurityRequired),
	})
}

func checkGlobalSecurityRequired(call *Call) []Finding {
	security := oas.Get(call.Root, "security")

	switch {
	case security == nil:
		return []Finding{{Node: call.Root, Path: "$", Message: "the document must define a global security property"}}
	case !oas.IsSeq(security):
		return []Finding{
			{Node: security, Path: "$.security", Message: "security must be a list of security requirements"},
		}
	case len(security.Content) == 0:
		return []Finding{
			{Node: security, Path: "$.security", Message: "security must contain at least one security requirement"},
		}
	}

	return seq.FlatMapPairs(oas.Items(security), func(i int, item *yaml.Node) []Finding {
		return When(!oas.IsMap(item), func() Finding {
			return Finding{
				Node:    item,
				Path:    oas.Index("$.security", i),
				Message: "a security requirement must be an object",
			}
		})
	})
}
