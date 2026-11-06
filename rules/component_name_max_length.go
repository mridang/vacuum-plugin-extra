package rules

import (
	"strconv"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// componentNameMaxLengthExamples is the examples section of components.
const componentNameMaxLengthExamples = "examples"

// componentNameMaxLengthDefault is the longest component key accepted by default.
const componentNameMaxLengthDefault = 50

// ComponentNameMaxLength returns the extra-component-name-max-length rule.
//
// # What it checks
//
// The keys of components.schemas, requestBodies, responses, parameters,
// examples, headers and securitySchemes must not be longer than a maximum
// number of characters.
//
// # Why
//
// Code generators turn component names into class, type and file names. Very
// long names produce unreadable identifiers and can exceed file name or
// identifier limits of the target language or file system.
//
// # Where it comes from
//
// Rate My OpenAPI (apimatic-component-name-max-length).
//
// # Options
//
//   - maxLength: the longest accepted key, counted in characters. Default: 50.
//
// # Bad
//
//	components:
//	  schemas:
//	    TheVeryLongNameOfASchemaThatKeepsGoingAndGoingAndGoingForever:
//	      type: object
//
// # Good
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//
// # Known false positives / limitations
//
// Components of other sections (links, callbacks, pathItems) are not checked.
// Names that contain multi-byte characters are counted by character, not byte.
func ComponentNameMaxLength() Rule {
	return New(Spec{
		Name:        "component-name-max-length",
		Description: "Component names must not be longer than a maximum length.",
		HowToFix:    "Rename the component to a shorter name and update the references to it.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        keywordMaxLength,
			Description: "The longest accepted component key, in characters.",
			Default:     componentNameMaxLengthDefault,
		}},
		Check: OnDocument(checkComponentNameMaxLength),
	})
}

// componentNameMaxLengthSections lists the components sections that are checked.
func componentNameMaxLengthSections() []string {
	return []string{
		literalSchemas,
		literalRequestbodies,
		literalResponses,
		literalParameters,
		componentNameMaxLengthExamples,
		keywordHeaders,
		"securitySchemes",
	}
}

func checkComponentNameMaxLength(call *Call) []Finding {
	limit := call.Int("maxLength")
	if limit <= 0 {
		return nil
	}

	return seq.FlatMapSlice(componentNameMaxLengthSections(), func(section string) []Finding {
		base := oas.JoinAll("$.components", section)

		return seq.FlatMapPairs(
			oas.Pairs(oas.Dig(call.Root, "components", section)),
			func(key, _ *yaml.Node) []Finding {
				length := utf8.RuneCountInString(key.Value)

				return When(length > limit, func() Finding {
					return Finding{
						Node: key,
						Path: oas.Join(base, key.Value),
						Message: "component name '" + key.Value + "' is " + strconv.Itoa(
							length,
						) + " characters long, the maximum is " + strconv.Itoa(
							limit,
						),
					}
				})
			},
		)
	})
}
