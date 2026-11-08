package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestComponentSchemaNameCase(t *testing.T) {
	t.Parallel()

	rule := rules.ComponentSchemaNameCase()

	t.Run("flags names that are not PascalCase", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    pet_store:
      type: object
    petStore:
      type: object
    Pet-Owner:
      type: object
    PetStore:
      type: object
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.pet_store",
			"$.components.schemas.petStore",
			"$.components.schemas['Pet-Owner']",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "PascalCase")
	})

	t.Run("accepts PascalCase names including acronyms", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
    HTTPError:
      type: object
    ID:
      type: string
`), nil)
	})

	t.Run("kebab-case style accepts kebab names and flags the rest", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    pet-owner:
      type: object
    pet:
      type: object
    PetOwner:
      type: object
    pet_owner:
      type: object
`), ruletest.Options{"casing": "kebab-case"},
			"$.components.schemas.PetOwner",
			"$.components.schemas.pet_owner",
		)
	})

	t.Run("ignore skips names that match a pattern", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
paths: {}
components:
  schemas:
    pet-owner:
      type: object
    auth-csrf_invalid:
      type: object
    enc_key-not_found:
      type: object
`)

		ruletest.RequirePaths(t, rule, document, ruletest.Options{"casing": "kebab-case"},
			"$.components.schemas['auth-csrf_invalid']",
			"$.components.schemas['enc_key-not_found']",
		)
		ruletest.RequirePaths(t, rule, document,
			ruletest.Options{"casing": "kebab-case", "ignore": "^auth-.*_"},
			"$.components.schemas['enc_key-not_found']",
		)
		ruletest.RequireClean(t, rule, document,
			ruletest.Options{"casing": "kebab-case", "ignore": []string{"^auth-.*_", "^enc_key-"}})
	})

	t.Run("an ignore pattern that does not compile is reported", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
`), ruletest.Options{"ignore": "("})

		require.Equal(t, []string{"$"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, `option ignore: invalid pattern "("`)
	})

	t.Run("does nothing without component schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
`), nil)
	})
}
