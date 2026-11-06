package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const webhookRestrictionsBad = `
webhooks:
  newPet:
    servers:
      - url: https://consumer.example.com
    post:
      callbacks: {}
      servers:
        - url: https://other.example.com
      responses:
        "200":
          description: ok
`

func TestWebhookRestrictions(t *testing.T) {
	t.Parallel()

	rule := rules.WebhookRestrictions()

	t.Run("flags callbacks and servers", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(webhookRestrictionsBad), nil)

		require.ElementsMatch(t, []string{
			"$.webhooks.newPet.servers",
			"$.webhooks.newPet.post.callbacks",
			"$.webhooks.newPet.post.servers",
		}, ruletest.Paths(results))
	})

	t.Run("accepts clean webhooks", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
webhooks:
  newPet:
    post:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("switches each half off by option", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc31(webhookRestrictionsBad)

		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"noCallbacks": false},
			"$.webhooks.newPet.servers", "$.webhooks.newPet.post.servers")
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"noServers": false},
			"$.webhooks.newPet.post.callbacks")
	})

	t.Run("does not apply to OpenAPI 3.0", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      callbacks: {}
      responses:
        "200":
          description: ok
`), nil)
	})
}
