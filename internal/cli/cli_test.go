package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"

	extra "github.com/mridang/vacuum-plugin-extra"
	"github.com/mridang/vacuum-plugin-extra/internal/cli"
)

const sampleSpec = `openapi: 3.0.3
info: {title: Sample, version: "1"}
paths:
  /pets:
    get:
      summary: List all pets.
      responses:
        "200": {description: ok}
`

func writeSpec(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "api.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleSpec), 0o600))

	return path
}

func TestRunReportsFindingsAsJSON(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer
	code := cli.Run(
		[]string{"-format", "json", "-only", "extra-summary-no-trailing-period", "-fail-on", "warn", writeSpec(t)},
		&out,
		&errOut,
	)

	require.Equal(t, cli.ExitFindings, code, errOut.String())

	var findings []cli.Finding
	require.NoError(t, json.Unmarshal(out.Bytes(), &findings))
	require.Len(t, findings, 1)
	require.Equal(t, "extra-summary-no-trailing-period", findings[0].Rule)
	require.Equal(t, "$.paths['/pets'].get.summary", findings[0].Path)
}

func TestRunPassesBelowTheFailureSeverity(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer
	code := cli.Run([]string{"-only", "extra-summary-no-trailing-period", writeSpec(t)}, &out, &errOut)

	require.Equal(t, cli.ExitOK, code, errOut.String())
	require.Contains(t, out.String(), "extra-summary-no-trailing-period")
}

func TestRunListsRules(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer

	require.Equal(t, cli.ExitOK, cli.Run([]string{"-list"}, &out, &errOut))
	require.Contains(t, out.String(), "extra-unique-tag-names")
}

func TestRunRejectsBadUsage(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer

	require.Equal(t, cli.ExitUsage, cli.Run(nil, &out, &errOut))
	require.Equal(t, cli.ExitUsage, cli.Run([]string{"-format", "xml", "a.yaml"}, &out, &errOut))
	require.Equal(t, cli.ExitUsage, cli.Run([]string{"/does/not/exist.yaml"}, &out, &errOut))
}

//nolint:paralleltest // the debug flag replaces the process wide default logger
func TestRunDebugLogsEveryRuleStage(t *testing.T) {
	var out, errOut bytes.Buffer

	code := cli.Run([]string{"-debug", "-only", "extra-summary-no-trailing-period,extra-no-eval-in-title",
		writeSpec(t)}, &out, &errOut)

	require.Equal(t, cli.ExitOK, code)

	logs := errOut.String()
	require.Contains(t, logs, "ruleset loaded")
	require.Contains(t, logs, "msg=\"rule skipped\" rule=extra-api-max-operations reason=\"not selected by -only\"")
	require.Contains(t, logs, "msg=run rule=extra-summary-no-trailing-period")
	require.Contains(t, logs, "nodes=")
	require.Contains(t, logs, "msg=done rule=extra-summary-no-trailing-period findings=1")
}

//nolint:paralleltest // the environment variable is read process wide
func TestRunDebugCanBeEnabledByEnvironment(t *testing.T) {
	t.Setenv("VACUUM_EXTRA_DEBUG", "1")

	var out, errOut bytes.Buffer

	cli.Run([]string{"-only", "extra-summary-no-trailing-period", writeSpec(t)}, &out, &errOut)

	require.Contains(t, errOut.String(), "ruleset loaded")
}

//nolint:paralleltest // the environment variable is read process wide
func TestRunIsSilentWithoutDebug(t *testing.T) {
	t.Setenv("VACUUM_EXTRA_DEBUG", "")

	var out, errOut bytes.Buffer

	cli.Run([]string{"-only", "extra-summary-no-trailing-period", writeSpec(t)}, &out, &errOut)

	require.Empty(t, errOut.String())
}

func TestRunSeesRefsAsWritten(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "api.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`openapi: 3.0.3
info: {title: Sample, version: "1"}
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                oneOf:
                  - $ref: "#/components/schemas/Cat"
                  - $ref: "#/components/schemas/Dog"
                discriminator: {propertyName: kind}
components:
  schemas:
    Cat: {type: object, required: [kind], properties: {kind: {type: string}, lives: {type: integer}}}
    Dog: {type: object, required: [kind], properties: {kind: {type: string}, bark: {type: boolean}}}
`), 0o600))

	var out, errOut bytes.Buffer

	code := cli.Run([]string{"-only", "extra-illogical-composition-keywords", path}, &out, &errOut)

	require.Equal(t, cli.ExitOK, code, errOut.String())
	require.Empty(t, out.String())
}

func TestRunFollowsReferencesToOtherFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pets.yaml"), []byte(`get:
  summary: List all pets.
  responses:
    "200": {description: ok}
`), 0o600))

	root := filepath.Join(dir, "api.yaml")
	require.NoError(t, os.WriteFile(root, []byte(`openapi: 3.0.3
info: {title: Sample, version: "1"}
paths:
  /pets:
    $ref: ./pets.yaml
`), 0o600))

	var out, errOut bytes.Buffer

	code := cli.Run([]string{"-format", "json", "-only", "extra-summary-no-trailing-period", root}, &out, &errOut)

	require.Equal(t, cli.ExitOK, code, errOut.String())

	var findings []cli.Finding
	require.NoError(t, json.Unmarshal(out.Bytes(), &findings))
	require.Len(t, findings, 1)
	require.Equal(t, "$.paths['/pets'].get.summary", findings[0].Path)
}

// rulesetFor writes a ruleset that holds only the given rule of the embedded
// ruleset, with the severity and, when not nil, the functionOptions replaced.
func rulesetFor(t *testing.T, id, severity string, options map[string]any) string {
	t.Helper()

	var doc struct {
		Rules map[string]map[string]any `yaml:"rules"`
	}

	require.NoError(t, yaml.Unmarshal(extra.Ruleset(), &doc))

	entry := doc.Rules[id]
	require.NotNil(t, entry, id)

	entry["severity"] = severity

	if options != nil {
		then, ok := entry["then"].(map[string]any)
		require.True(t, ok, id)

		then["functionOptions"] = options
	}

	data, err := yaml.Marshal(map[string]any{"rules": map[string]any{id: entry}})
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "ruleset.yaml")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	return path
}

func TestRunSkipsEveryRuleSwitchedOff(t *testing.T) {
	t.Parallel()

	spec := writeSpec(t)

	for _, rule := range extra.Rules() {
		t.Run(rule.ID(), func(t *testing.T) {
			t.Parallel()

			var out, errOut bytes.Buffer

			path := rulesetFor(t, rule.ID(), "off", nil)
			code := cli.Run([]string{"-r", path, "-format", "json", spec}, &out, &errOut)

			require.Equal(t, cli.ExitOK, code, errOut.String())
			require.JSONEq(t, "[]", out.String())
		})
	}
}

//nolint:paralleltest // the debug flag replaces the process wide default logger
func TestRunLogsEachRuleSwitchedOffOnce(t *testing.T) {
	for _, rule := range extra.Rules() {
		var out, errOut bytes.Buffer

		code := cli.Run([]string{"-debug", "-r", rulesetFor(t, rule.ID(), "off", nil), writeSpec(t)}, &out, &errOut)

		require.Equal(t, cli.ExitOK, code, errOut.String())
		line := "msg=\"rule skipped\" rule=" + rule.ID() + " reason=\"severity off\""
		require.Equal(t, 1, strings.Count(errOut.String(), line), rule.ID())
		require.NotContains(t, errOut.String(), "msg=run rule="+rule.ID(), rule.ID())
	}
}

func TestRunSkipsARuleSwitchedOffWithInvalidOptions(t *testing.T) {
	t.Parallel()

	path := rulesetFor(t, "extra-banned-words-in-docs", "off", map[string]any{"words": 42})

	var out, errOut bytes.Buffer

	code := cli.Run([]string{"-r", path, "-format", "json", writeSpec(t)}, &out, &errOut)

	require.Equal(t, cli.ExitOK, code, errOut.String())
	require.JSONEq(t, "[]", out.String())
}

func TestRunReportsTheConfiguredSeverity(t *testing.T) {
	t.Parallel()

	for _, severity := range []string{"error", "warn", "info", "hint"} {
		t.Run(severity, func(t *testing.T) {
			t.Parallel()

			path := rulesetFor(t, "extra-summary-no-trailing-period", severity, nil)

			var out, errOut bytes.Buffer

			cli.Run([]string{"-r", path, "-format", "json", "-fail-on", "none", writeSpec(t)}, &out, &errOut)

			var findings []cli.Finding
			require.NoError(t, json.Unmarshal(out.Bytes(), &findings), errOut.String())
			require.Len(t, findings, 1)
			require.Equal(t, severity, findings[0].Severity)
		})
	}
}

func TestRunAcceptsEverySeverityForEveryRule(t *testing.T) {
	t.Parallel()

	spec := writeSpec(t)

	for _, rule := range extra.Rules() {
		t.Run(rule.ID(), func(t *testing.T) {
			t.Parallel()

			for _, severity := range []string{"error", "warn", "info", "hint", "off"} {
				var out, errOut bytes.Buffer

				path := rulesetFor(t, rule.ID(), severity, nil)
				code := cli.Run([]string{"-r", path, "-fail-on", "none", spec}, &out, &errOut)

				require.Equal(t, cli.ExitOK, code, severity+": "+errOut.String())
			}
		})
	}
}

func TestRunRejectsAnUnknownSeverityNamingTheAllowedOnes(t *testing.T) {
	t.Parallel()

	for _, severity := range []string{"none", "fatal"} {
		var out, errOut bytes.Buffer

		path := rulesetFor(t, "extra-summary-no-trailing-period", severity, nil)
		code := cli.Run([]string{"-r", path, writeSpec(t)}, &out, &errOut)

		require.Equal(t, cli.ExitUsage, code, severity)
		require.Contains(t, errOut.String(), "'error', 'warn', 'info', 'hint', 'off'", severity)
	}
}

func TestRunDoesNotRunARuleMissingFromTheRuleset(t *testing.T) {
	t.Parallel()

	path := rulesetFor(t, "extra-no-eval-in-title", "warn", nil)

	var out, errOut bytes.Buffer

	code := cli.Run([]string{"-r", path, "-format", "json", writeSpec(t)}, &out, &errOut)

	require.Equal(t, cli.ExitOK, code, errOut.String())
	require.JSONEq(t, "[]", out.String())
}
