package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const pathSegmentPluralOp = `
    get:
      responses:
        "200":
          description: ok
`

func TestPathSegmentPlural(t *testing.T) {
	t.Parallel()

	rule := rules.PathSegmentPlural()

	t.Run("flags singular segments", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pet/{petId}:`+pathSegmentPluralOp+`
  /pets/{petId}/toy:`+pathSegmentPluralOp), nil)

		require.Len(t, results, 2)
		require.ElementsMatch(t, []string{
			"$.paths['/pet/{petId}']",
			"$.paths['/pets/{petId}/toy']",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "should be plural")
	})

	t.Run("accepts plural, irregular and uncountable segments", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{petId}/toys:`+pathSegmentPluralOp+`
  /people:`+pathSegmentPluralOp+`
  /user-groups:`+pathSegmentPluralOp+`
  /userGroups:`+pathSegmentPluralOp+`
  /metadata:`+pathSegmentPluralOp+`
  /data:`+pathSegmentPluralOp+`
  /children:`+pathSegmentPluralOp+`
  /schemata:`+pathSegmentPluralOp+`
  /categories:`+pathSegmentPluralOp+`
  /boxes:`+pathSegmentPluralOp), nil)
	})

	t.Run("reports uncountable words like Redocly unless excepted", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /sheep:` + pathSegmentPluralOp + `
  /analytics:` + pathSegmentPluralOp)

		ruletest.RequirePaths(t, rule, doc, nil, "$.paths['/sheep']", "$.paths['/analytics']")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"exceptions": []any{"sheep", "analytics"}})
	})

	t.Run("judges versions and dotted names as whole words like Redocly", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /v1/pets:` + pathSegmentPluralOp + `
  /files/report.pdf:` + pathSegmentPluralOp)

		ruletest.RequirePaths(t, rule, doc, nil, "$.paths['/v1/pets']", "$.paths['/files/report.pdf']")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"exceptions": []any{"v1", "report.pdf", "files"}})
	})

	t.Run("flags irregular and suffix singulars and suggests the plural", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /person:`+pathSegmentPluralOp+`
  /child:`+pathSegmentPluralOp+`
  /category:`+pathSegmentPluralOp+`
  /box:`+pathSegmentPluralOp+`
  /foot:`+pathSegmentPluralOp+`
  /schema:`+pathSegmentPluralOp), nil)

		require.Len(t, results, 6)
		require.Contains(t, results[0].Message, "`people`")
		require.Contains(t, results[1].Message, "`children`")
		require.Contains(t, results[2].Message, "`categories`")
		require.Contains(t, results[3].Message, "`boxes`")
		require.Contains(t, results[4].Message, "`feet`")
		require.Contains(t, results[5].Message, "`schemata`")
	})

	t.Run("judges only whole parameter segments as parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets/{petId}.json:`+pathSegmentPluralOp), nil, "$.paths['/pets/{petId}.json']")
	})

	t.Run("flags compound segments by their whole text", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /user_group:`+pathSegmentPluralOp), nil, "$.paths['/user_group']")
	})

	t.Run("ignores the last segment on request", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets/lookup:` + pathSegmentPluralOp)

		ruletest.RequirePaths(t, rule, doc, nil, "$.paths['/pets/lookup']")
		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/lookup:`+pathSegmentPluralOp), ruletest.Options{"ignoreLastPathSegment": true})
	})

	t.Run("honours exceptions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /status:`+pathSegmentPluralOp), ruletest.Options{"exceptions": []any{"status"}})
	})

	t.Run("checks only collection segments when asked", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pet/{petId}/toy:` + pathSegmentPluralOp)

		ruletest.RequirePaths(
			t,
			rule,
			doc,
			ruletest.Options{"onlyCollectionSegments": true},
			"$.paths['/pet/{petId}/toy']",
		)
		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{petId}/toy:`+pathSegmentPluralOp), ruletest.Options{"onlyCollectionSegments": true})
	})
}
