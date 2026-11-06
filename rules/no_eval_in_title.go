package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// noEvalInTitleNeedle is the text that must not appear in titles and
// descriptions.
const noEvalInTitleNeedle = "eval("

// noEvalInTitleDescription is the description keyword.
const noEvalInTitleDescription = "description"

// NoEvalInTitle returns the extra-no-eval-in-title rule.
//
// # What it checks
//
// No title or description string anywhere in the document may contain the text
// eval( , compared case-insensitively. Documentation text is rendered by many
// tools and some of them evaluate or template strings; a description containing
// a call to eval is a known vector for script injection in such renderers, and
// it is never legitimate prose. Only scalar values of keys named title or
// description are examined, so a property that is itself called description
// (a mapping) is not mistaken for text.
//
// # Where it comes from
//
// Speccy and the OpenAPI Kit (oas-kit) linters, which both reject eval( in
// descriptions and titles.
//
// # Options
//
// None.
//
// # Bad
//
//	info:
//	  title: Pets
//	  description: Call eval(x) to start
//
// # Good
//
//	info:
//	  title: Pets
//	  description: Evaluate the request before sending it
//
// # Known false positives
//
// A genuine discussion of the JavaScript eval function in documentation is
// flagged. An example payload inside a description is flagged too.
func NoEvalInTitle() Rule {
	return New(Spec{
		Name:        "no-eval-in-title",
		Description: "Titles and descriptions must not contain eval(.",
		HowToFix:    "Remove the eval( text from the title or description.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkNoEvalInTitle),
	})
}

func checkNoEvalInTitle(call *Call) []Finding {
	return seq.FlatMap(oas.Walk(call.Root, "$"), func(node oas.Node) []Finding {
		return When(
			(node.Key == literalTitle || node.Key == noEvalInTitleDescription) &&
				oas.IsScalar(node.Node) &&
				strings.Contains(strings.ToLower(node.Node.Value), noEvalInTitleNeedle),
			func() Finding {
				return Finding{
					Node:    node.Node,
					Path:    node.Path,
					Message: node.Key + " must not contain " + noEvalInTitleNeedle,
				}
			},
		)
	})
}
