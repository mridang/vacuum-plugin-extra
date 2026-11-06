package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"

	extra "github.com/mridang/vacuum-plugin-extra"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// Exit codes returned by Run.
const (
	// ExitOK means the document was linted and nothing at or above the failure
	// severity was found.
	ExitOK = 0
	// ExitFindings means at least one finding reached the failure severity.
	ExitFindings = 1
	// ExitUsage means the command line or the inputs were invalid.
	ExitUsage = 2
)

const (
	formatText   = "text"
	formatJSON   = "json"
	severityNone = "none"
)

// ErrUsage is returned for invalid command lines.
var ErrUsage = errors.New("usage")

// options are the parsed command line options.
type options struct {
	ruleset string
	format  string
	failOn  string
	only    string
	list    bool
	spec    string
}

// Finding is one violation in the machine readable output.
type Finding struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
}

// Run executes the command with the given arguments and returns the exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	opts, err := parse(args, stderr)
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stderr, "vacuum-extra:", err)
		}

		return ExitUsage
	}

	if opts.list {
		listRules(stdout)

		return ExitOK
	}

	findings, err := lint(opts)
	if err != nil {
		fmt.Fprintln(stderr, "vacuum-extra:", err)

		return ExitUsage
	}

	if err = render(stdout, opts.format, findings); err != nil {
		fmt.Fprintln(stderr, "vacuum-extra:", err)

		return ExitUsage
	}

	if reaches(findings, opts.failOn) {
		return ExitFindings
	}

	return ExitOK
}

func parse(args []string, stderr io.Writer) (options, error) {
	set := flag.NewFlagSet("vacuum-extra", flag.ContinueOnError)
	set.SetOutput(stderr)

	ruleset := set.String("r", "", "ruleset file to use instead of the embedded one")
	format := set.String("format", formatText, "output format: text or json")
	failOn := set.String(
		"fail-on",
		model.SeverityError,
		"lowest severity that fails the run: error, warn, info, hint or none",
	)
	only := set.String("only", "", "comma separated rule ids to run; all rules run when empty")
	list := set.Bool("list", false, "list the rule ids and exit")

	if err := set.Parse(args); err != nil {
		return options{}, fmt.Errorf("parse flags: %w", err)
	}

	opts := options{
		ruleset: *ruleset,
		format:  *format,
		failOn:  *failOn,
		only:    *only,
		list:    *list,
		spec:    set.Arg(0),
	}

	if opts.list {
		return opts, nil
	}

	if set.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: vacuum-extra [flags] <openapi-file>")
		set.PrintDefaults()

		return opts, fmt.Errorf("%w: expected exactly one OpenAPI file", ErrUsage)
	}

	if opts.format != formatText && opts.format != formatJSON {
		return opts, fmt.Errorf("%w: unknown format %q", ErrUsage, opts.format)
	}

	return opts, nil
}

func listRules(out io.Writer) {
	for _, rule := range extra.Rules() {
		fmt.Fprintf(out, "%s\t%s\t%s\n", rule.ID(), rule.Spec().Severity, rule.Spec().Description)
	}
}

func rulesetBytes(path string) ([]byte, error) {
	if path == "" {
		return extra.Ruleset(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ruleset: %w", err)
	}

	return data, nil
}

func loadRuleset(path string) (*rulesets.RuleSet, error) {
	data, err := rulesetBytes(path)
	if err != nil {
		return nil, err
	}

	set, err := rulesets.CreateRuleSetFromData(data)
	if err != nil {
		return nil, fmt.Errorf("parse ruleset: %w", err)
	}

	return set, nil
}

func lint(opts options) ([]Finding, error) {
	spec, err := os.ReadFile(opts.spec)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}

	loaded, err := loadRuleset(opts.ruleset)
	if err != nil {
		return nil, err
	}

	set := onlyRules(loaded, opts.only)

	result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:           set,
		Spec:              spec,
		SpecFileName:      opts.spec,
		CustomFunctions:   extra.Functions(),
		SilenceLogs:       true,
		SkipDocumentCheck: true,
		Logger:            slog.New(slog.DiscardHandler),
	})
	defer result.ReleaseOwnedResources()

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("lint: %w", errors.Join(result.Errors...))
	}

	return convert(result.Results), nil
}

// onlyRules returns set restricted to the comma separated rule ids in only, or
// set itself when only is empty.
func onlyRules(set *rulesets.RuleSet, only string) *rulesets.RuleSet {
	if only == "" {
		return set
	}

	ids := strings.Split(only, ",")

	return &rulesets.RuleSet{
		Executable:       set.Executable,
		FormatMatches:    set.FormatMatches,
		Description:      set.Description,
		DocumentationURI: set.DocumentationURI,
		Formats:          set.Formats,
		RuleDefinitions:  set.RuleDefinitions,
		Rules: maps.Collect(func(yield func(string, *model.Rule) bool) {
			for id, rule := range set.Rules {
				if slices.Contains(ids, id) && !yield(id, rule) {
					return
				}
			}
		}),
		Extends:       set.Extends,
		Aliases:       set.Aliases,
		ParsedAliases: set.ParsedAliases,
	}
}

func convert(results []model.RuleFunctionResult) []Finding {
	findings := slices.AppendSeq(
		make([]Finding, 0, len(results)),
		slices.Values(seq.Map(slices.Values(results), func(result model.RuleFunctionResult) Finding {
			line, column := position(result)

			return Finding{
				Rule:     result.RuleId,
				Severity: result.RuleSeverity,
				Path:     result.Path,
				Line:     line,
				Column:   column,
				Message:  result.Message,
			}
		})),
	)

	slices.SortStableFunc(findings, func(a, b Finding) int {
		if a.Line != b.Line {
			return a.Line - b.Line
		}

		return strings.Compare(a.Rule, b.Rule)
	})

	return findings
}

// position returns the line and column of a result, falling back to the start
// node when the engine did not fill in the range.
func position(result model.RuleFunctionResult) (int, int) {
	if result.Range.Start.Line > 0 {
		return result.Range.Start.Line, result.Range.Start.Char
	}

	if result.StartNode != nil {
		return result.StartNode.Line, result.StartNode.Column
	}

	return 0, 0
}

func render(out io.Writer, format string, findings []Finding) error {
	if format == formatJSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(findings); err != nil {
			return fmt.Errorf("encode findings: %w", err)
		}

		return nil
	}

	for _, finding := range findings {
		fmt.Fprintf(out, "%d:%d\t%s\t%s\t%s\t%s\n",
			finding.Line, finding.Column, finding.Severity, finding.Rule, finding.Message, finding.Path)
	}

	return nil
}

// severities lists the severities from the most to the least severe.
func severities() []string {
	return []string{model.SeverityError, model.SeverityWarn, model.SeverityInfo, model.SeverityHint}
}

func rank(severity string) int {
	index := slices.Index(severities(), severity)
	if index < 0 {
		return len(severities())
	}

	return index
}

func reaches(findings []Finding, failOn string) bool {
	if failOn == severityNone {
		return false
	}

	limit := rank(failOn)
	for _, finding := range findings {
		if rank(finding.Severity) <= limit {
			return true
		}
	}

	return false
}
