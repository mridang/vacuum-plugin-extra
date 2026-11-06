package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestReadonlyAndWriteonlyExclusive(t *testing.T) {
	t.Parallel()

	rule := rules.ReadonlyAndWriteonlyExclusive()

	t.Run("flags a property that is both", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    User:
      type: object
      properties:
        password:
          type: string
          readOnly: true
          writeOnly: true
`), nil)

		require.Equal(t, []string{"$.components.schemas.User.properties.password"}, ruletest.Paths(results))
	})

	t.Run("flags a schema in a request body for 3.1", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /users:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: string
              readOnly: true
              writeOnly: true
      responses:
        "201":
          description: created
`), nil, "$.paths['/users'].post.requestBody.content['application/json'].schema")
	})

	t.Run("accepts a single flag or false values", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          readOnly: true
        password:
          type: string
          writeOnly: true
        name:
          type: string
          readOnly: true
          writeOnly: false
`), nil)
	})
}
