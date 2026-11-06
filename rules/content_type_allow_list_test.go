package rules_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestContentTypeAllowList(t *testing.T) {
	t.Parallel()

	rule := rules.ContentTypeAllowList()

	t.Run("flags media types outside the default list", func(t *testing.T) {
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
          content:
            text/plain:
              schema:
                type: string
            application/json:
              schema:
                type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].post.requestBody.content['application/xml']",
			"$.paths['/pets'].post.responses['200'].content['text/plain']",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "application/json")
	})

	t.Run("ignores parameters and letter case", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            Application/JSON; charset=utf-8:
              schema:
                type: string
`), nil)
	})

	t.Run("checks component request bodies and responses", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  requestBodies:
    Upload:
      content:
        multipart/form-data:
          schema:
            type: object
  responses:
    Page:
      description: page
      content:
        text/html:
          schema:
            type: string
`), nil,
			"$.components.requestBodies.Upload.content['multipart/form-data']",
			"$.components.responses.Page.content['text/html']",
		)
	})

	t.Run("honours the mimetypes option including wildcards", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            text/plain:
              schema:
                type: string
            application/xml:
              schema:
                type: string
            image/png:
              schema:
                type: string
`), ruletest.Options{"mimetypes": []any{"text/*", "application/json"}},
			"$.paths['/pets'].get.responses['200'].content['application/xml']",
			"$.paths['/pets'].get.responses['200'].content['image/png']",
		)
	})

	t.Run("validates example bodies against the schema", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
            example:
              name: 5
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
              examples:
                missing:
                  value:
                    age: 3
                good:
                  value:
                    name: rex
                    age: 3
                badEnum:
                  value:
                    name: rex
                    kind: fish
                nested:
                  value:
                    name: rex
                    owner:
                      email: not-an-email
                      tags: [a, a]
                shared:
                  $ref: '#/components/examples/Bad'
components:
  examples:
    Bad:
      value:
        name: rex
        age: -1
  schemas:
    Pet:
      type: object
      required: [name]
      properties:
        name:
          type: string
        age:
          type: integer
          minimum: 0
        kind:
          type: string
          enum: [dog, cat]
        owner:
          type: object
          properties:
            email:
              type: string
              format: email
            tags:
              type: array
              uniqueItems: true
              items:
                type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].post.requestBody.content['application/json'].example",
			"$.paths['/pets'].post.responses['200'].content['application/json'].examples.missing.value",
			"$.paths['/pets'].post.responses['200'].content['application/json'].examples.badEnum.value",
			"$.paths['/pets'].post.responses['200'].content['application/json'].examples.nested.value",
			"$.paths['/pets'].post.responses['200'].content['application/json'].examples.shared",
		}, ruletest.Paths(results))

		messages := make([]string, 0, len(results))
		for _, result := range results {
			messages = append(messages, result.Message)
		}

		joined := strings.Join(messages, "\n")
		require.Contains(t, joined, "'body.name' must be string")
		require.Contains(t, joined, "must have required property 'name'")
		require.Contains(t, joined, "'body.kind' must be equal to one of the allowed values")
		require.Contains(t, joined, "'body.owner.email' must match format \"email\"")
		require.Contains(t, joined, "'body.age' must be greater than or equal to 0")
	})

	t.Run("parses string example bodies as JSON and YAML text", func(t *testing.T) {
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
                type: object
                required: [id]
              examples:
                broken:
                  value: '{"id": '
                wrong:
                  value: '{"name": "x"}'
                fine:
                  value: '{"id": 1}'
            application/yaml:
              schema:
                type: object
                required: [id]
              example: "id: 1"
            text/plain:
              schema:
                type: string
              example: "{"
`)

		results := ruletest.Run(t, rule, doc, ruletest.Options{
			"mimetypes": []any{"application/json", "application/yaml", "text/plain"},
		})
		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['200'].content['application/json'].examples.broken.value",
			"$.paths['/pets'].get.responses['200'].content['application/json'].examples.wrong.value",
		}, ruletest.Paths(results))

		for _, result := range results {
			require.Condition(t, func() bool {
				return strings.Contains(result.Message, "JSON body does not parse correctly!") ||
					strings.Contains(result.Message, "required property 'id'")
			})
		}

		ruletest.RequireClean(t, rule, doc, ruletest.Options{
			"mimetypes":        []any{"application/json", "application/yaml", "text/plain"},
			"validateExamples": false,
		})
	})

	t.Run("does not parse string examples of string schemas and validates composition", func(t *testing.T) {
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
                type: string
                maxLength: 3
              example: '{"id": '
        "201":
          description: created
          content:
            application/json:
              schema:
                oneOf:
                  - type: integer
                  - type: string
                    format: uuid
              example: not-a-uuid
        "202":
          description: accepted
          content:
            application/json:
              schema:
                type: [number, "null"]
                exclusiveMinimum: 1
                multipleOf: 2
              example: 3
`), nil,
			"$.paths['/pets'].get.responses['200'].content['application/json'].example",
			"$.paths['/pets'].get.responses['201'].content['application/json'].example",
			"$.paths['/pets'].get.responses['202'].content['application/json'].example",
		)
	})
}
