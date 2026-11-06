package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestEncodingKeysMatchProperties(t *testing.T) {
	t.Parallel()

	rule := rules.EncodingKeysMatchProperties()

	t.Run("flags an encoding key without a matching property", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /upload:
    post:
      requestBody:
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                avatar:
                  type: string
                  format: binary
            encoding:
              avatr:
                contentType: image/png
              avatar:
                contentType: image/png
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(
			t,
			"$.paths['/upload'].post.requestBody.content['multipart/form-data'].encoding.avatr",
			results[0].Path,
		)
		require.Contains(t, results[0].Message, "avatr")
	})

	t.Run("accepts matching keys through a ref and allOf", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /upload:
    post:
      requestBody:
        content:
          multipart/form-data:
            schema:
              $ref: "#/components/schemas/Upload"
            encoding:
              avatar:
                contentType: image/png
              name:
                contentType: text/plain
      responses:
        "200":
          description: ok
components:
  schemas:
    Upload:
      allOf:
        - type: object
          properties:
            avatar: {type: string, format: binary}
        - type: object
          properties:
            name: {type: string}
`), nil)
	})

	t.Run("skips media types without a schema or with oneOf", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /upload:
    post:
      requestBody:
        content:
          multipart/form-data:
            encoding:
              anything:
                contentType: image/png
          application/x-www-form-urlencoded:
            schema:
              oneOf:
                - type: object
                  properties:
                    a: {type: string}
            encoding:
              b:
                style: form
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("checks component request bodies", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  requestBodies:
    Upload:
      content:
        multipart/form-data:
          schema:
            type: object
            properties:
              file: {type: string}
          encoding:
            other:
              contentType: text/plain
`), nil, "$.components.requestBodies.Upload.content['multipart/form-data'].encoding.other")
	})
}
