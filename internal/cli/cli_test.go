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
