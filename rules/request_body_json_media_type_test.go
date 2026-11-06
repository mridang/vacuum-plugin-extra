package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestRequestBodyJSONMediaType(t *testing.T) {
	t.Parallel()

	rule := rules.RequestBodyJSONMediaType()

	t.Run("flags a body without application/json", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/xml:
            schema:
              type: string
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].post.requestBody", results[0].Path)
		require.Contains(t, results[0].Message, "application/json")
	})

	t.Run("accepts a body offering application/json among others", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/xml:
            schema: {type: string}
          application/json:
            schema: {type: string}
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("requires an exact key", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json; charset=utf-8:
            schema: {type: string}
          application/vnd.api+json:
            schema: {type: string}
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets'].post.requestBody")
	})

	t.Run("resolves a referenced request body", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        $ref: "#/components/requestBodies/Pet"
      responses:
        "200":
          description: ok
components:
  requestBodies:
    Pet:
      content:
        text/plain:
          schema: {type: string}
`), nil, "$.paths['/pets'].post.requestBody")
	})

	t.Run("ignores missing or empty bodies and honours the option", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
    put:
      requestBody:
        content: {}
      responses:
        "200":
          description: ok
    post:
      requestBody:
        content:
          application/xml:
            schema: {type: string}
      responses:
        "200":
          description: ok
`)
		ruletest.RequirePaths(t, rule, doc, nil, "$.paths['/pets'].post.requestBody")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"mediaType": "application/xml"})
	})
}
