package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestEtagWithConditionalHeaders(t *testing.T) {
	t.Parallel()

	rule := rules.EtagWithConditionalHeaders()

	t.Run("flags a GET without ETag when another operation is conditional", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      responses:
        "200":
          description: ok
        "404":
          description: missing
    put:
      parameters:
        - name: If-Match
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets/{id}'].get.responses['200']", results[0].Path)
	})

	t.Run("honours path item level parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    parameters:
      - name: if-none-match
        in: header
        schema:
          type: string
    get:
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets/{id}'].get.responses['200']")
	})

	t.Run("accepts a GET with an ETag header", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      parameters:
        - name: If-None-Match
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
          headers:
            etag:
              schema:
                type: string
`), nil)
	})

	t.Run("ignores path items without conditional headers", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})
}
