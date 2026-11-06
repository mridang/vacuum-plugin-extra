package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestUnsupported31Keywords(t *testing.T) {
	t.Parallel()

	rule := rules.Unsupported31Keywords()

	t.Run("flags jsonSchemaDialect and webhooks", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
jsonSchemaDialect: https://json-schema.org/draft/2020-12/schema
webhooks:
  newPet:
    post:
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{"$.jsonSchemaDialect", "$.webhooks"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "not supported")
	})

	t.Run("accepts a document without the keywords", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths: {}
`), nil)
	})

	t.Run("is not applied to OpenAPI 3.0 documents", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
webhooks: {}
paths: {}
`), nil)
	})

	t.Run("honours the keywords option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
jsonSchemaDialect: https://json-schema.org/draft/2020-12/schema
webhooks: {}
`), ruletest.Options{"keywords": []string{"webhooks"}}, "$.webhooks")
	})
}
