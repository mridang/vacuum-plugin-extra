package rules

import (
	"strings"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
)

// CasingOption is the name of the functionOption that every casing rule uses
// to select its naming convention. Its value is one of the names returned by
// [casing.Names], or a list of them when a name may follow any of several.
const CasingOption = "casing"

// casingOption declares the shared casing option with the default convention
// of one rule.
func casingOption(subject string, defaults ...casing.Casing) Option {
	return Option{
		Name: CasingOption,
		Description: subject + ": one value or a list of " + strings.Join(casing.Names(), ", ") +
			". An empty list disables the casing check.",
		Default: casingDefaults(defaults),
	}
}

// casingDefaults renders conventions as the plain strings of a ruleset.
func casingDefaults(defaults []casing.Casing) []string {
	names := make([]string, 0, len(defaults))

	for _, c := range defaults {
		names = append(names, string(c))
	}

	return names
}

// WithCasing parses the shared casing option and hands the resulting set to
// next. An unknown value is reported as a single finding on the document that
// names the allowed values, and next is not run.
func WithCasing(next func(call *Call, allowed casing.Set) []Finding) Check {
	return func(call *Call) []Finding {
		allowed, err := casing.ParseSet(call.Strings(CasingOption))
		if err != nil {
			call.log.Debug("invalid casing option", "error", err)

			return One(Finding{Path: "$", Message: "option " + CasingOption + ": " + err.Error()})
		}

		call.log.Debug("casing", "allowed", allowed.String())

		return next(call, allowed)
	}
}
