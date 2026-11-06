package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoBinaryInJSONAPI(t *testing.T) {
	t.Parallel()

	rule := rules.NoBinaryInJSONAPI()

	t.Run("flags binary schemas in parameters and JSON bodies", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /files:
    parameters:
      - name: blob
        in: query
        schema:
          type: string
          format: binary
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: string
              format: binary
      responses:
        "200":
          description: ok
          content:
            application/json; charset=utf-8:
              schema:
                type: array
                items:
                  type: string
                  format: binary
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/files'].parameters[0].schema",
			"$.paths['/files'].post.requestBody.content['application/json'].schema",
			"$.paths['/files'].post.responses['200'].content['application/json; charset=utf-8'].schema",
		}, ruletest.Paths(results))
		require.ElementsMatch(t, []string{
			"Parameters should not contain binary values (type: string, format: binary)",
			"Request bodies with JSON content should not contain binary values (type: string, format: binary)",
			"Responses with JSON content should not contain binary values (type: string, format: binary)",
		}, ruletest.Messages(results))
	})

	t.Run("accepts binary data in non JSON media types", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /files:
    post:
      requestBody:
        content:
          application/octet-stream:
            schema:
              type: string
              format: binary
      responses:
        "200":
          description: ok
          content:
            application/pdf:
              schema:
                type: string
                format: binary
`), nil)
	})

	t.Run("resolves a schema reference", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /files:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Blob"
components:
  schemas:
    Blob:
      type: string
      format: binary
`), nil, "$.paths['/files'].get.responses['200'].content['application/json'].schema")
	})

	t.Run("follows referenced parameters, bodies and composition", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /files:
    post:
      parameters:
        - $ref: "#/components/parameters/Blob"
      requestBody:
        $ref: "#/components/requestBodies/Upload"
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                oneOf:
                  - type: string
                    format: binary
                  - type: [string]
                    format: binary
        "201":
          description: ok
          content:
            application/json:
              schema:
                oneOf:
                  - type: string
                    format: binary
                  - type: string
components:
  parameters:
    Blob:
      name: blob
      in: query
      content:
        application/json:
          schema:
            type: [string]
            format: binary
  requestBodies:
    Upload:
      content:
        application/json:
          schema:
            allOf:
              - type: string
                format: binary
`), nil,
			"$.paths['/files'].post.parameters[0].content['application/json'].schema",
			"$.paths['/files'].post.requestBody.content['application/json'].schema",
			"$.paths['/files'].post.responses['200'].content['application/json'].schema",
		)
	})

	t.Run("does not treat other json suffix types or components as json bodies", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /files:
    get:
      responses:
        "200":
          description: ok
          content:
            application/vnd.api+json:
              schema:
                type: string
                format: binary
components:
  requestBodies:
    Unused:
      content:
        application/json:
          schema:
            type: string
            format: binary
`), nil)
	})

	t.Run("ignores plain strings", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /files:
    get:
      parameters:
        - name: q
          in: query
          schema:
            type: string
            format: byte
      responses:
        "200":
          description: ok
`), nil)
	})
}
