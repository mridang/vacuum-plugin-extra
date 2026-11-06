package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestBannedWordsInDocs(t *testing.T) {
	t.Parallel()

	rule := rules.BannedWordsInDocs()
	options := ruletest.Options{"words": []any{"simply", "obviously"}}

	t.Run("flags banned words case-insensitively", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: Simply list pets
      description: This is OBVIOUSLY fine, obviously.
      responses:
        "200":
          description: ok
`), options, "$.paths['/pets'].get.summary", "$.paths['/pets'].get.description")
	})

	t.Run("respects word boundaries", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: Simplyfied listing
      responses:
        "200":
          description: ok
`), options)
	})

	t.Run("does nothing without configured words", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: Simply list pets
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("checks info title", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, `
openapi: 3.0.3
info:
  title: Obviously great
  version: "1"
paths: {}
`, options, "$.info.title")
	})
}
