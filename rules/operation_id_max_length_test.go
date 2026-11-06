package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOperationIDMaxLength(t *testing.T) {
	t.Parallel()

	rule := rules.OperationIDMaxLength()

	const longID = "aVeryLongOperationIdentifierThatGoesOnAndOnAndOnAndOnAndOnAndOnAndOnAndOnForever"

	t.Run("flags an operationId above the default limit", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      operationId: `+longID+`
      responses:
        "200":
          description: ok
`), nil)

		require.Equal(t, []string{"$.paths['/pets'].get.operationId"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "64")
	})

	t.Run("accepts an operationId at the limit", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: ok
`), ruletest.Options{"maxLength": 8})
	})

	t.Run("honours a lower limit", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: ok
`), ruletest.Options{"maxLength": 7}, "$.paths['/pets'].get.operationId")
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
}
