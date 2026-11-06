package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestValidMediaTypeSyntax(t *testing.T) {
	t.Parallel()

	rule := rules.ValidMediaTypeSyntax()

	t.Run("flags malformed request body media types", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          json:
            schema: {type: object}
          appliation/json:
            schema: {type: object}
          application/:
            schema: {type: object}
          application/json; charset=utf-8:
            schema: {type: object}
      responses:
        "200":
          description: ok
          content:
            nonsense:
              schema: {type: object}
components:
  requestBodies:
    Body:
      content:
        text:
          schema: {type: string}
`), nil,
			"$.paths['/pets'].post.requestBody.content.json",
			"$.paths['/pets'].post.requestBody.content['appliation/json']",
			"$.paths['/pets'].post.requestBody.content['application/']",
			"$.components.requestBodies.Body.content.text")
	})

	t.Run("accepts valid types and wildcards", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/vnd.api+json:
            schema: {type: object}
          multipart/form-data:
            schema: {type: object}
          image/*:
            schema: {type: string}
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the types option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          x-world/x-vrml:
            schema: {type: string}
      responses:
        "200":
          description: ok
`), ruletest.Options{"types": []any{"x-world"}})
	})
}
