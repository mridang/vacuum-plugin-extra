package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const operationIDCasingDocument = `
paths:
  /pets:
    get:
      operationId: list_pets
      responses:
        "200":
          description: ok
    post:
      operationId: createPet
      responses:
        "201":
          description: ok
    put:
      operationId: replace-pet
      responses:
        "200":
          description: ok
    delete:
      responses:
        "204":
          description: ok
`

func TestOperationIDCasing(t *testing.T) {
	t.Parallel()

	rule := rules.OperationIDCasing()

	t.Run("flags identifiers that are not snake case", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(operationIDCasingDocument), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].post.operationId",
			"$.paths['/pets'].put.operationId",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "snake")
	})

	t.Run("accepts identifiers in the configured style", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(operationIDCasingDocument),
			ruletest.Options{"style": "camel"},
			"$.paths['/pets'].get.operationId",
			"$.paths['/pets'].put.operationId",
		)
	})

	t.Run("strictCharset rejects hyphens and dots", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      operationId: list.pets
      responses:
        "200":
          description: ok
    post:
      operationId: create-pet
      responses:
        "201":
          description: ok
`), ruletest.Options{"style": "", "strictCharset": true})

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.operationId",
			"$.paths['/pets'].post.operationId",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "letters, digits and underscores")
	})

	t.Run("accepts every operation of a conforming document", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      operationId: list_pets
      responses:
        "200":
          description: ok
`), nil)
	})
}
