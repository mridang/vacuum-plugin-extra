package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// UnusedGlobalTags returns the extra-unused-global-tags rule.
//
// # What it checks
//
// Every tag declared in the top-level tags list must be referenced by the tags
// of at least one operation. Tags are matched by name. A document without a
// top-level tags list is not checked.
//
// # Why
//
// Declared but unused tags show up as empty groups in documentation, add noise
// to the navigation and usually mean that an operation was re-tagged or removed
// without tidying the tag list.
//
// # Where it comes from
//
// The IBM API Handbook ruleset (ibm-unused-tags).
//
// # Options
//
// None.
//
// # Bad
//
//	tags:
//	  - name: pets
//	  - name: orders
//	paths:
//	  /pets:
//	    get:
//	      tags: [pets]
//
// # Good
//
//	tags:
//	  - name: pets
//	paths:
//	  /pets:
//	    get:
//	      tags: [pets]
//
// # Known false positives / limitations
//
// A tag that is used only by operations reached through an external $ref path
// item, or only by webhooks and callbacks, is reported as unused.
func UnusedGlobalTags() Rule {
	return New(Spec{
		Name:        "unused-global-tags",
		Description: "Every global tag must be used by at least one operation.",
		HowToFix:    "Remove the unused tag or add it to an operation.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkUnusedGlobalTags),
	})
}

// checkUnusedGlobalTags reports global tags no operation references.
func checkUnusedGlobalTags(call *Call) []Finding {
	used := slices.Concat(
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []string {
			return oas.Strings(oas.Get(operation.Node, "tags"))
		}),
		unusedGlobalTagsWebhookTags(call.Root),
	)

	return seq.FlatMapPairs(oas.Items(oas.Get(call.Root, "tags")), func(i int, tag *yaml.Node) []Finding {
		name := oas.Get(tag, "name")

		return When(oas.IsScalar(name) && !slices.Contains(used, name.Value), func() Finding {
			return Finding{
				Node:    name,
				Path:    oas.Join(oas.Index("$.tags", i), "name"),
				Message: "tag " + name.Value + " is not used by any operation",
			}
		})
	})
}

// unusedGlobalTagsWebhookTags lists the tags of the operations under webhooks,
// which count as uses of a tag like the operations under paths.
func unusedGlobalTagsWebhookTags(root *yaml.Node) []string {
	return seq.FlatMapPairs(oas.Pairs(oas.Get(root, "webhooks")), func(_, item *yaml.Node) []string {
		return seq.FlatMapPairs(oas.Pairs(oas.Resolve(root, item)), func(method, operation *yaml.Node) []string {
			if !oas.IsMethod(method.Value) {
				return nil
			}

			return oas.Strings(oas.Get(operation, "tags"))
		})
	})
}
