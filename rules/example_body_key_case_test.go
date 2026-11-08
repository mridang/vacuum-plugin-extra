package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestExampleBodyKeyCase(t *testing.T) {
	t.Parallel()

	rule := rules.ExampleBodyKeyCase()

	t.Run("flags camel case keys at every depth in a snake case API", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              example:
                firstName: Ada
                owner_id: 1
                address:
                  zipCode: "123"
                tags:
                  - lastSeen: now
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['200'].content['application/json'].example.firstName",
			"$.paths['/pets'].get.responses['200'].content['application/json'].example.address.zipCode",
			"$.paths['/pets'].get.responses['200'].content['application/json'].example.tags[0].lastSeen",
		}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results),
			"example body property 'firstName' must be snake_case (for example 'first_name')",
		)
	})

	t.Run("accepts snake case examples", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            examples:
              basic:
                value:
                  first_name: Ada
                  id2: 1
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("checks named examples and honours the casing option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            examples:
              basic:
                value:
                  first_name: Ada
                  lastName: Lovelace
      responses:
        "200":
          description: ok
`), ruletest.Options{"casing": "camelCase"},
			"$.paths['/pets'].post.requestBody.content['application/json'].examples.basic.value.first_name")
	})

	t.Run("supports kebab case and acronyms", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  responses:
    Ok:
      description: ok
      content:
        application/json:
          example:
            user-id: 1
            userID: 2
`), ruletest.Options{"casing": "kebab-case"}, "$.components.responses.Ok.content['application/json'].example.userID")
	})

	t.Run("ignores non json media types", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/xml:
              example:
                firstName: Ada
`)
		ruletest.RequireClean(t, rule, doc, nil)
	})
}
