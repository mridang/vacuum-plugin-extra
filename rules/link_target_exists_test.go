package rules_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const linkTargetExistsBase = `
paths:
  /users:
    post:
      operationId: createUser
      responses:
        "201":
          description: created
          links:
            fetch:
              %s
  /users/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
    get:
      operationId: getUser
      responses:
        "200":
          description: ok
    delete:
      operationId: deleteUser
      responses:
        "204":
          description: gone
`

func linkTargetExistsDoc(link string) string {
	return ruletest.Doc30(strings.Replace(linkTargetExistsBase, "%s", link, 1))
}

func TestLinkTargetExists(t *testing.T) {
	t.Parallel()

	rule := rules.LinkTargetExists()

	t.Run("flags an unknown operationId", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, linkTargetExistsDoc("operationId: getUsr"), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/users'].post.responses['201'].links.fetch.operationId", results[0].Path)
		require.Contains(t, results[0].Message, "getUsr")
	})

	t.Run("accepts a known operationId", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, linkTargetExistsDoc("operationId: getUser"), nil)
	})

	t.Run("resolves a relative operationRef", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, linkTargetExistsDoc(`operationRef: "#/paths/~1users~1{id}/get"`), nil)
		ruletest.RequirePaths(t, rule, linkTargetExistsDoc(`operationRef: "#/paths/~1nope/get"`), nil,
			"$.paths['/users'].post.responses['201'].links.fetch.operationRef")
	})

	t.Run("restricts operationRef methods when configured", func(t *testing.T) {
		t.Parallel()

		doc := linkTargetExistsDoc(`operationRef: "#/paths/~1users~1{id}/delete"`)
		ruletest.RequireClean(t, rule, doc, nil)
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"operationRefMethods": []any{"get"}},
			"$.paths['/users'].post.responses['201'].links.fetch.operationRef")
	})

	t.Run("accepts external operationRef values", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(
			t,
			rule,
			linkTargetExistsDoc(`operationRef: "https://example.com/openapi.yaml#/paths/~1x/get"`),
			nil,
		)
	})

	t.Run("flags links with both or neither target", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, linkTargetExistsDoc(`description: nothing`), nil,
			"$.paths['/users'].post.responses['201'].links.fetch")
		ruletest.RequirePaths(
			t,
			rule,
			linkTargetExistsDoc("operationId: getUser\n              operationRef: \"#/paths/~1users/post\""),
			nil,
			"$.paths['/users'].post.responses['201'].links.fetch",
		)
	})

	t.Run("checks component links", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  links:
    Broken:
      operationId: missing
`), nil, "$.components.links.Broken.operationId")
	})
}
