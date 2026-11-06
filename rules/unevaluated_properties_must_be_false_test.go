package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestUnevaluatedPropertiesMustBeFalse(t *testing.T) {
	t.Parallel()

	rule := rules.UnevaluatedPropertiesMustBeFalse()

	t.Run("flags true and schema values", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    A:
      type: object
      unevaluatedProperties: true
    B:
      type: object
      unevaluatedProperties:
        type: string
    C:
      type: object
      unevaluatedProperties: false
    D:
      type: object
`), nil,
			"$.components.schemas.A.unevaluatedProperties",
			"$.components.schemas.B.unevaluatedProperties")
	})

	t.Run("accepts false", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    C:
      type: object
      unevaluatedProperties: false
`), nil)
	})

	t.Run("does not apply to 3.0 documents", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    A:
      type: object
      unevaluatedProperties: true
`), nil)
	})
}
