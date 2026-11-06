package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOperationIDVerbPrefix(t *testing.T) {
	t.Parallel()

	rule := rules.OperationIDVerbPrefix()

	t.Run("flags identifiers without a leading verb", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      operationId: pets
      responses:
        "200":
          description: ok
    post:
      operationId: getaway
      responses:
        "201":
          description: created
    put:
      operationId: list_pets
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.operationId",
			"$.paths['/pets'].post.operationId",
			"$.paths['/pets'].put.operationId",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "verb")
	})

	t.Run("accepts verbs followed by an upper-case letter", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: ok
    post:
      operationId: createPet
      responses:
        "201":
          description: created
`), nil)
	})

	t.Run("ignores operations without an operationId", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the prefixes option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc31(`
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: ok
    post:
      operationId: registerPet
      responses:
        "201":
          description: created
`)

		ruletest.RequirePaths(t, rule, document, ruletest.Options{"prefixes": []string{"register"}},
			"$.paths['/pets'].get.operationId")
	})
}
