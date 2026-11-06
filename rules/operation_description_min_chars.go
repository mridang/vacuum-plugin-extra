package rules

import (
	"cmp"
	"slices"

	"strings"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// operationDescriptionMinCharsOption is the name of the minimum length option.
const operationDescriptionMinCharsOption = "minLength"

// operationDescriptionMinCharsDefault is the default minimum length.
const operationDescriptionMinCharsDefault = 30

// operationDescriptionMinCharsDescription is the description keyword.
const operationDescriptionMinCharsDescription = "description"

// OperationDescriptionMinChars returns the extra-operation-description-min-chars rule.
//
// # What it checks
//
// For every operation that has a description, a summary or both, the longer of
// the two (after trimming white space, counted in characters) must be at least
// the configured length. Operations with neither field are not reported here;
// other rules require the fields to exist. One-word summaries such as "Get"
// teach neither people nor language models how the operation behaves, so at
// least one of the two texts must say something substantial.
//
// # Where it comes from
//
// ai-api-lint, which checks that operation text is long enough to be useful.
//
// # Options
//
//   - minLength: minimum number of characters. Default: 30. A value of zero
//     or less disables the rule.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      summary: List pets
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      summary: List pets
//	      description: Returns every pet that the caller is allowed to see.
//
// # Known false positives
//
// Short but complete texts in languages with dense scripts (for example
// Chinese) may be flagged because the length is counted in characters.
func OperationDescriptionMinChars() Rule {
	return New(Spec{
		Name:        "operation-description-min-chars",
		Description: "The longer of an operation's description and summary must reach a minimum length.",
		HowToFix:    "Write a more descriptive summary or description for the operation.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        operationDescriptionMinCharsOption,
			Description: "Minimum number of characters of the longer of description and summary.",
			Default:     operationDescriptionMinCharsDefault,
		}},
		Check: PerOperation(checkOperationDescriptionMinChars),
	})
}

// checkOperationDescriptionMinChars reports operations whose longest text is
// shorter than the configured minimum. It does nothing for a minimum of zero or
// less.
func checkOperationDescriptionMinChars(call *Call, operation oas.Operation) []Finding {
	minimum := call.Int(operationDescriptionMinCharsOption)
	node, key := operationDescriptionMinCharsLongest(operation.Node)

	return When(minimum > 0 && node != nil && operationDescriptionMinCharsLength(node) < minimum, func() Finding {
		return Finding{
			Node:    node,
			Path:    oas.Join(operation.Path, key),
			Message: "the longest of summary and description is shorter than the minimum length",
		}
	})
}

// operationDescriptionMinCharsText is a scalar text of an operation and its key.
type operationDescriptionMinCharsText struct {
	node *yaml.Node
	key  string
}

// operationDescriptionMinCharsLength returns the trimmed length in characters.
func operationDescriptionMinCharsLength(node *yaml.Node) int {
	return utf8.RuneCountInString(strings.TrimSpace(node.Value))
}

// operationDescriptionMinCharsLongest returns the longer of the description and
// summary, preferring the description on a tie, or nil when neither is a scalar.
func operationDescriptionMinCharsLongest(operation *yaml.Node) (*yaml.Node, string) {
	texts := seq.FlatMapSlice(
		[]string{operationDescriptionMinCharsDescription, "summary"},
		func(key string) []operationDescriptionMinCharsText {
			node := oas.Get(operation, key)
			if !oas.IsScalar(node) {
				return nil
			}

			return []operationDescriptionMinCharsText{{node: node, key: key}}
		},
	)
	if len(texts) == 0 {
		return nil, ""
	}

	best := slices.MaxFunc(texts, func(a, b operationDescriptionMinCharsText) int {
		return cmp.Compare(operationDescriptionMinCharsLength(a.node), operationDescriptionMinCharsLength(b.node))
	})

	return best.node, best.key
}
