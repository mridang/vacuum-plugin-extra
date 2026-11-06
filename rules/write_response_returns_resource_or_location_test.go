package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestWriteResponseReturnsResourceOrLocation(t *testing.T) {
	t.Parallel()

	rule := rules.WriteResponseReturnsResourceOrLocation()

	t.Run("flags empty write responses", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      responses:
        "201":
          description: created
    put:
      responses:
        "200":
          description: ok
    get:
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].post.responses['201']",
			"$.paths['/pets'].put.responses['200']",
		}, ruletest.Paths(results))
	})

	t.Run("accepts content or a Location header", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      responses:
        "201":
          description: created
          headers:
            location:
              schema:
                type: string
    patch:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
        "400":
          description: bad
`), nil)
	})

	t.Run("requires Location only for 201 and 204", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    put:
      responses:
        "200":
          description: ok
          headers:
            Location:
              schema:
                type: string
`), nil, "$.paths['/pets'].put.responses['200']")
	})

	t.Run("honours ignoreCodes", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets:
    post:
      responses:
        "202":
          description: accepted
        "204":
          description: none
`)

		ruletest.RequirePaths(t, rule, doc, nil, "$.paths['/pets'].post.responses['204']")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"ignoreCodes": []any{"202", "204"}})
	})
}
