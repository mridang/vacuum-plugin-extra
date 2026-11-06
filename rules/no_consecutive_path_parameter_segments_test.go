package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoConsecutivePathParameterSegments(t *testing.T) {
	t.Parallel()

	rule := rules.NoConsecutivePathParameterSegments()

	t.Run("flags consecutive parameter segments", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets/{petId}/{ownerId}:
    get:
      responses:
        "200":
          description: ok
  /a/{x}/{y}/{z}:
    get:
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets/{petId}/{ownerId}']",
			"$.paths['/a/{x}/{y}/{z}']",
		}, ruletest.Paths(results))
	})

	t.Run("accepts separated parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{petId}/owners/{ownerId}:
    get:
      responses:
        "200":
          description: ok
  /pets/{petId}:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("treats mixed literal and template segments as parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /files/{name}.{ext}/{version}:
    get:
      responses:
        "200":
          description: ok
`), nil, "$.paths['/files/{name}.{ext}/{version}']")
	})
}
