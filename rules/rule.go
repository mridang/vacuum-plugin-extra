package rules

import (
	"cmp"
	"context"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// IDPrefix is prepended to the name of every rule so that the identifiers never
// clash with the rules that vacuum ships.
const IDPrefix = "extra-"

// categoryName is the category that every function of this package reports.
const categoryName = "extra"

// Option describes one entry of the functionOptions a rule accepts.
type Option struct {
	// Name is the functionOptions key.
	Name string
	// Description explains what the option controls and its accepted values.
	Description string
	// Default is the value used when the option is not supplied. A nil Default
	// means the option has no default and the rule does nothing for it.
	Default any
}

// Finding is one violation reported by a check.
type Finding struct {
	// Node is the YAML node the violation points at.
	Node *yaml.Node
	// Path is the JSONPath of the violation. An empty Path means the root.
	Path string
	// Message says what is wrong.
	Message string
}

// Check inspects the node a rule was given and returns its violations.
type Check func(call *Call) []Finding

// Spec is the declarative description of a rule.
type Spec struct {
	// Name is the kebab-case rule name without the IDPrefix.
	Name string
	// Description is the one-line statement of what the rule checks.
	Description string
	// HowToFix tells the user how to resolve a violation.
	HowToFix string
	// Severity is one of the model.Severity constants.
	Severity string
	// Given is the JSONPath whose matches are handed to Check.
	Given string
	// Type is the vacuum rule type, "style" or "validation".
	Type string
	// Formats restricts the rule to the listed document formats. It is empty
	// for rules that apply to every OpenAPI version.
	Formats []string
	// Options lists the functionOptions the rule understands.
	Options []Option
	// Check is the implementation of the rule.
	Check Check
}

// Rule is a vacuum custom function with the metadata of its ruleset entry.
type Rule struct {
	spec Spec
}

// New builds a Rule from its specification.
func New(spec Spec) Rule {
	return Rule{spec: spec}
}

// Spec returns the declarative description of the rule.
func (r Rule) Spec() Spec {
	return r.spec
}

// ID returns the rule identifier, which is also the custom function name.
func (r Rule) ID() string {
	return IDPrefix + r.spec.Name
}

// GetSchema implements model.RuleFunction.
func (r Rule) GetSchema() model.RuleFunctionSchema {
	properties := slices.AppendSeq(
		make([]model.RuleFunctionProperty, 0, len(r.spec.Options)),
		slices.Values(seq.Map(slices.Values(r.spec.Options), func(option Option) model.RuleFunctionProperty {
			return model.RuleFunctionProperty{
				Name:        option.Name,
				Description: option.Description,
			}
		})),
	)

	return model.RuleFunctionSchema{
		Name:       r.ID(),
		Properties: properties,
	}
}

// GetCategory implements model.RuleFunction.
func (r Rule) GetCategory() string {
	return categoryName
}

// RunRule implements model.RuleFunction. It hands every node matched by the
// given JSONPath to the check and converts the findings into vacuum results.
func (r Rule) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	log := slog.Default().With("rule", r.ID())
	enabled := log.Enabled(context.Background(), slog.LevelDebug)

	if enabled {
		log.Debug("run", "given", r.spec.Given, "options", ctx.Options, "nodes", len(nodes))
	}

	results := seq.FlatMapSlice(nodes, func(node *yaml.Node) []model.RuleFunctionResult {
		return seq.Map(
			slices.Values(r.spec.Check(newCall(node, ctx, r.spec.Options, log))),
			func(finding Finding) model.RuleFunctionResult {
				return r.result(finding, node, ctx)
			},
		)
	})

	if enabled {
		log.Debug("done", "findings", len(results))
	}

	return results
}

func (r Rule) result(finding Finding, matched *yaml.Node, context model.RuleFunctionContext) model.RuleFunctionResult {
	target := cmp.Or(finding.Node, oas.Unwrap(matched))
	path := cmp.Or(finding.Path, "$")

	return model.RuleFunctionResult{
		Message:   finding.Message,
		StartNode: target,
		EndNode:   target,
		Path:      path,
		Rule:      context.Rule,
	}
}

// RuleEntry returns the model.Rule that wires the function into a ruleset,
// carrying the given JSONPath, the severity and the default options.
func (r Rule) RuleEntry() *model.Rule {
	return r.RuleEntryWith(nil)
}

// RuleEntryWith is RuleEntry with the given functionOptions layered over the
// defaults of the rule.
func (r Rule) RuleEntryWith(overrides map[string]any) *model.Rule {
	options := maps.Collect(func(yield func(string, any) bool) {
		for _, option := range r.spec.Options {
			if option.Default != nil && !yield(option.Name, option.Default) {
				return
			}
		}

		for name, value := range overrides {
			if !yield(name, value) {
				return
			}
		}
	})

	return &model.Rule{
		Id:          r.ID(),
		Name:        r.ID(),
		Description: r.spec.Description,
		Message:     "{{error}}",
		HowToFix:    r.spec.HowToFix,
		Given:       r.spec.Given,
		Formats:     r.spec.Formats,
		Severity:    r.spec.Severity,
		Type:        r.spec.Type,
		Then: model.RuleAction{
			Function:        r.ID(),
			FunctionOptions: options,
		},
	}
}

// ShortName returns the rule name without the IDPrefix.
func ShortName(id string) string {
	return strings.TrimPrefix(id, IDPrefix)
}

// Rule types accepted by Spec.Type.
const (
	// TypeStyle marks a rule that enforces a convention.
	TypeStyle = "style"
	// TypeValidation marks a rule that enforces correctness.
	TypeValidation = "validation"
)
