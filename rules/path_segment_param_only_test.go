package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestPathSegmentParamOnly(t *testing.T) {
	t.Parallel()

	rule := rules.PathSegmentParamOnly()

	t.Run("flags segments mixing text and parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /files/{name}.json:
    get:
      responses:
        "200":
          description: ok
  /pets/pet_{id}:
    get:
      responses:
        "200":
          description: ok
  /a/{x}{y}:
    get:
      responses:
        "200":
          description: ok
`), nil,
			"$.paths['/files/{name}.json']", "$.paths['/pets/pet_{id}']", "$.paths['/a/{x}{y}']")
	})

	t.Run("accepts pure literal and pure parameter segments", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{petId}/toys/{toyId}:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})
}
