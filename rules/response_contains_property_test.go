package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const responseContainsPropertyDoc = `
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
        "404":
          description: missing
          content:
            application/json:
              schema:
                type: object
                properties:
                  code: {type: integer}
        "400":
          description: bad
          content:
            application/json:
              schema:
                type: object
                properties:
                  message: {type: string}
components:
  schemas:
    Pet:
      allOf:
        - type: object
          properties:
            name: {type: string}
`

func TestResponseContainsProperty(t *testing.T) {
	t.Parallel()

	rule := rules.ResponseContainsProperty()

	t.Run("flags responses lacking configured properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(responseContainsPropertyDoc),
			ruletest.Options{"names": map[string]any{"4XX": []any{"message"}, "200": []any{"id"}}},
			"$.paths['/pets'].get.responses['404'].content['application/json'].schema.properties",
			"$.paths['/pets'].get.responses['200'].content['application/json'].schema.properties")
	})

	t.Run("accepts responses with the properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(responseContainsPropertyDoc),
			ruletest.Options{"names": map[string]any{"200": []any{"name"}}})
	})

	t.Run("does nothing without configuration", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(responseContainsPropertyDoc), nil)
	})

	t.Run("uses the first matching key and skips 204", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(responseContainsPropertyDoc), ruletest.Options{
			"names": map[string]any{"400": []any{"code"}, "4XX": []any{"message"}, "204": []any{"x"}},
		},
			"$.paths['/pets'].get.responses['400'].content['application/json'].schema.properties",
			"$.paths['/pets'].get.responses['404'].content['application/json'].schema.properties")
	})

	t.Run("does not inspect 204 responses", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    delete:
      responses:
        "204":
          description: gone
          content:
            application/json:
              schema: {type: object, properties: {}}
`), ruletest.Options{"names": map[string]any{"204": []any{"x"}}})
	})

	t.Run("supports lower-case ranges and default", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(responseContainsPropertyDoc), ruletest.Options{
			"names": map[string]any{"4xx": []any{"nope"}},
		},
			"$.paths['/pets'].get.responses['404'].content['application/json'].schema.properties",
			"$.paths['/pets'].get.responses['400'].content['application/json'].schema.properties")
	})

	t.Run("requires properties in every oneOf branch", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                oneOf:
                  - type: object
                    properties: {id: {type: string}, a: {type: string}}
                  - type: object
                    properties: {id: {type: string}}
        "201":
          description: ok
          content:
            application/json:
              schema:
                anyOf:
                  - type: object
                    properties: {id: {type: string}, a: {type: string}}
                  - type: object
                    properties: {id: {type: string}}
`)

		ruletest.RequireClean(t, rule, doc, ruletest.Options{"names": map[string]any{"2XX": []any{"id"}}})
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"names": map[string]any{"200": []any{"a"}}},
			"$.paths['/pets'].get.responses['200'].content['application/json'].schema.properties")
	})

	t.Run("skips array schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: array
                items: {type: string}
`), ruletest.Options{"names": map[string]any{"200": []any{"id"}}})
	})
}
