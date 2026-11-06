package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	webhookRestrictionsCallbacksOption = "noCallbacks"
	webhookRestrictionsServersOption   = "noServers"
)

// WebhookRestrictions returns the extra-webhook-restrictions rule.
//
// # What it checks
//
// Webhooks, the OpenAPI 3.1 top-level webhooks map, describe requests that the
// API provider sends to the consumer, so two features of the specification make
// no sense there. With noCallbacks, an operation of a webhook must not define
// callbacks (a webhook is already a callback). With noServers, neither a
// webhook path item nor any of its operations may define servers, because the
// receiving URL is chosen by the consumer and not by the provider. Every
// violation is reported at the offending callbacks or servers keyword.
//
// # Where it comes from
//
// Spectral's OpenAPI ruleset, which has webhook-no-callbacks and
// webhook-no-servers, merged into this rule; each half is switched by an
// option.
//
// # Options
//
//   - noCallbacks: forbid callbacks in webhook operations. Default: true.
//   - noServers: forbid servers in webhook path items and operations.
//     Default: true.
//
// # Bad
//
//	webhooks:
//	  newPet:
//	    servers:
//	      - url: https://consumer.example.com
//	    post:
//	      callbacks: {}
//	      responses: {}
//
// # Good
//
//	webhooks:
//	  newPet:
//	    post:
//	      responses: {}
//
// # Known false positives
//
// None known. The rule only applies to OpenAPI 3.1 documents, where webhooks
// exist; callbacks under ordinary paths are unaffected.
func WebhookRestrictions() Rule {
	return New(Spec{
		Name:        "webhook-restrictions",
		Description: "Webhooks must not define callbacks, and webhook path items and operations must not define servers.",
		HowToFix:    "Remove the callbacks or servers keyword from the webhook.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Formats:     []string{formatOAS31},
		Options: []Option{
			{
				Name:        webhookRestrictionsCallbacksOption,
				Description: "Forbid callbacks in webhook operations.",
				Default:     true,
			},
			{
				Name:        webhookRestrictionsServersOption,
				Description: "Forbid servers in webhook path items and operations.",
				Default:     true,
			},
		},
		Check: checkWebhookRestrictions,
	})
}

func checkWebhookRestrictions(call *Call) []Finding {
	noCallbacks := call.Bool(webhookRestrictionsCallbacksOption)
	noServers := call.Bool(webhookRestrictionsServersOption)

	return seq.FlatMapPairs(oas.Pairs(oas.Get(call.Root, "webhooks")), func(name, hook *yaml.Node) []Finding {
		item := oas.Resolve(call.Root, hook)
		path := oas.Join("$.webhooks", name.Value)

		return slices.Concat(
			webhookRestrictionsFlag(noServers, item, path, "servers", "webhook path item must not define servers"),
			seq.FlatMapPairs(oas.Pairs(item), func(method, operation *yaml.Node) []Finding {
				if !oas.IsMethod(method.Value) {
					return nil
				}

				opPath := oas.Join(path, method.Value)

				return slices.Concat(
					webhookRestrictionsFlag(
						noCallbacks, operation, opPath, "callbacks", "webhook operation must not define callbacks"),
					webhookRestrictionsFlag(
						noServers, operation, opPath, "servers", "webhook operation must not define servers"),
				)
			}),
		)
	})
}

func webhookRestrictionsFlag(enabled bool, node *yaml.Node, path, keyword, message string) []Finding {
	if !enabled {
		return nil
	}

	key, value := oas.Entry(node, keyword)

	return When(value != nil, func() Finding {
		return Finding{Node: key, Path: oas.Join(path, keyword), Message: message}
	})
}
