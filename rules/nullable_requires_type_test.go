package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNullableRequiresType(t *testing.T) {
	t.Parallel()

	rule := rules.NullableRequiresType()

	t.Run("flags nullable without a type", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Owner:
      nullable: true
      allOf:
        - $ref: "#/components/schemas/User"
    User:
      type: object
      properties:
        nick:
          nullable: true
          enum: [a]
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Owner.nullable",
			"$.components.schemas.User.properties.nick.nullable",
		}, ruletest.Paths(results))
	})

	t.Run("accepts nullable with a type", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Owner:
      type: object
      nullable: true
`), nil)
	})

	t.Run("checks inline operation schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                nullable: false
`), nil, "$.paths['/pets'].get.responses['200'].content['application/json'].schema.nullable")
	})

	t.Run("ignores schemas without nullable", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Any:
      description: anything
`), nil)
	})
}
