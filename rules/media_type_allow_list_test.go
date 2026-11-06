package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const mediaTypeAllowListDocument = `
paths:
  /pets:
    post:
      requestBody:
        $ref: "#/components/requestBodies/Pet"
      responses:
        "200":
          description: ok
          content:
            text/csv:
              schema:
                type: string
  /owners:
    post:
      requestBody:
        content:
          application/json; charset=utf-8:
            schema:
              type: object
          application/xml:
            schema:
              type: object
      responses:
        "200":
          description: ok
components:
  requestBodies:
    Pet:
      content:
        multipart/form-data:
          schema:
            type: object
`

func TestMediaTypeAllowList(t *testing.T) {
	t.Parallel()

	rule := rules.MediaTypeAllowList()
	options := ruletest.Options{"allowed": []any{"application/json"}}

	t.Run("flags request media types outside the list", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(mediaTypeAllowListDocument), options)

		require.ElementsMatch(t, []string{
			"$.paths['/owners'].post.requestBody.content['application/xml']",
			"$.components.requestBodies.Pet.content['multipart/form-data']",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "allow-list")
	})

	t.Run("does nothing when the list is empty", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(mediaTypeAllowListDocument), nil)
	})

	t.Run("supports type wildcards", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(mediaTypeAllowListDocument),
			ruletest.Options{"allowed": []any{"application/*"}},
			"$.components.requestBodies.Pet.content['multipart/form-data']",
		)
	})

	t.Run("checks responses when enabled", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(
			t,
			rule,
			ruletest.Doc30(mediaTypeAllowListDocument),
			ruletest.Options{
				"allowed":          []any{"application/json", "application/xml", "multipart/form-data"},
				"includeResponses": true,
			},
			"$.paths['/pets'].post.responses['200'].content['text/csv']",
		)
	})
}
