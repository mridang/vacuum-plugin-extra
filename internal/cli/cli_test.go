package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

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
