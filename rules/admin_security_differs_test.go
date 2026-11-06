package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestAdminSecurityDiffers(t *testing.T) {
	t.Parallel()

	rule := rules.AdminSecurityDiffers()

	t.Run("flags an admin operation that reuses a non-admin requirement", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /users:
    get:
      security:
        - oauth: [read, write]
      responses:
        "200":
          description: ok
  /admin/users:
    get:
      security:
        - oauth: [write, read]
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/admin/users'].get.security[0]", results[0].Path)
		require.Contains(t, results[0].Message, "/admin/users")
	})

	t.Run("accepts a distinct admin scope", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /users:
    get:
      security:
        - oauth: [read]
      responses:
        "200":
          description: ok
  /admin/users:
    get:
      security:
        - oauth: [admin]
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("ignores operations without operation level security", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /users:
    get:
      responses:
        "200":
          description: ok
  /admin/users:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours a custom admin fragment", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /users:
    get:
      security:
        - key: []
      responses:
        "200":
          description: ok
  /internal/users:
    get:
      security:
        - key: []
      responses:
        "200":
          description: ok
`), ruletest.Options{"adminUrl": "/internal"}, "$.paths['/internal/users'].get.security[0]")
	})
}
