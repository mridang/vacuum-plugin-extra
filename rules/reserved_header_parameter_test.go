package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestReservedHeaderParameter(t *testing.T) {
	t.Parallel()

	rule := rules.ReservedHeaderParameter()

	t.Run("flags reserved header parameters at every level", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: accept
        in: header
        schema: {type: string}
    post:
      parameters:
        - name: Content-Type
          in: header
          schema: {type: string}
        - name: X-Trace
          in: header
          schema: {type: string}
      responses:
        "200":
          description: ok
          headers:
            content-type:
              schema: {type: string}
components:
  parameters:
    Auth:
      name: Authorization
      in: header
      schema: {type: string}
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].parameters[0].name",
			"$.paths['/pets'].post.parameters[0].name",
			"$.paths['/pets'].post.responses['200'].headers['content-type']",
			"$.components.parameters.Auth.name",
		}, ruletest.Paths(results))
	})

	t.Run("accepts unreserved headers and non header parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: X-Trace
          in: header
          schema: {type: string}
        - name: Accept
          in: query
          schema: {type: string}
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the names option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      parameters:
        - name: Authorization
          in: header
          schema: {type: string}
        - name: X-Trace
          in: header
          schema: {type: string}
      responses:
        "200":
          description: ok
`), ruletest.Options{"names": []any{"x-trace"}}, "$.paths['/pets'].get.parameters[1].name")
	})

	t.Run("flags media types overridden by a Content-Type header", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      parameters:
        - name: Content-Type
          in: header
          schema:
            type: string
            enum: [application/xml]
      requestBody:
        content:
          application/json:
            schema: {type: object}
          application/xml:
            schema: {type: object}
      responses:
        "200":
          description: ok
          headers:
            Content-Type:
              schema:
                type: string
                enum: [text/plain]
          content:
            application/json:
              schema: {type: string}
`), ruletest.Options{"names": []any{"x-none"}})

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].post.requestBody.content['application/json']",
			"$.paths['/pets'].post.responses['200'].content['application/json']",
		}, ruletest.Paths(results))

		for _, result := range results {
			require.Contains(t, result.Message, "does not match the respective Content-Type header")
		}
	})

	t.Run("flags response types that the Accept header cannot allow", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: accept
        in: header
        schema:
          type: string
          enum: ["application/xml;q=0.5, text/*", "image/png;q=0"]
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: string}
            text/csv:
              schema: {type: string}
            application/xml:
              schema: {type: string}
            image/png:
              schema: {type: string}
        "404":
          $ref: '#/components/responses/Missing'
components:
  responses:
    Missing:
      description: missing
      content:
        application/problem+json:
          schema: {type: string}
`)

		results := ruletest.Run(t, rule, doc, ruletest.Options{"names": []any{"x-none"}})
		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['200'].content['application/json']",
			"$.paths['/pets'].get.responses['200'].content['image/png']",
			"$.components.responses.Missing.content['application/problem+json']",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "do not allow response Content-Type")
	})

	t.Run("accepts wildcards and unknown Accept values", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: Accept
          in: header
          schema: {type: string, enum: ["*/*"]}
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: string}
  /owners:
    get:
      parameters:
        - name: X-Accept
          in: header
          schema: {type: string}
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: string}
`), ruletest.Options{"names": []any{"x-none"}})
	})
}
