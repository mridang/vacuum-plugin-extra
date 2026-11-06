package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const validMediaTypeSyntaxTypesOption = "types"

// validMediaTypeSyntaxSubtype matches a subtype of RFC 6838 or the wildcard.
const validMediaTypeSyntaxSubtype = `^(\*|[A-Za-z0-9][A-Za-z0-9!#$&^_.+-]*)$`

// ValidMediaTypeSyntax returns the extra-valid-media-type-syntax rule.
//
// # What it checks
//
// Every key of the content map of a request body (inline or under
// components.requestBodies) must be a well formed media type: a top-level type
// from the configured list, a slash, and a subtype made of the characters
// RFC 6838 allows. Parameters after a semicolon, such as charset=utf-8, are
// ignored, and the wildcards */* and type/* are accepted. The finding is placed
// on the media type entry.
//
// # Why
//
// A misspelt media type such as appliation/json, json or application\json is
// not recognised by servers or client generators, so the body is silently
// treated as unsupported.
//
// # Where it comes from
//
// rate-my-openapi (request body content keys must have a valid mime prefix).
//
// # Options
//
//   - types (list of strings, default application, audio, font, image, message,
//     model, multipart, text, video): the accepted top-level types.
//
// # Bad
//
//	requestBody:
//	  content:
//	    json:
//	      schema: {type: object}
//
// # Good
//
//	requestBody:
//	  content:
//	    application/json:
//	      schema: {type: object}
//
// # Known false positives
//
// Vendor trees that use a non-standard top-level type, such as x-world/x-vrml,
// are flagged until the type is added to the types option. Only request bodies
// are checked; response content is not.
func ValidMediaTypeSyntax() Rule {
	return New(Spec{
		Name:        "valid-media-type-syntax",
		Description: "Request body content keys must be valid media types.",
		HowToFix:    "Use a media type of the form type/subtype, such as application/json.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{{
			Name:        validMediaTypeSyntaxTypesOption,
			Description: "Accepted top-level media types.",
			Default: []string{
				"application", "audio", "font", "image", "message", "model", "multipart", "text", "video",
			},
		}},
		Check: Concat(
			PerOperation(checkValidMediaTypeSyntaxOperation),
			OnDocument(checkValidMediaTypeSyntaxComponents),
		),
	})
}

// checkValidMediaTypeSyntaxOperation validates the content keys of the request
// body of one operation.
func checkValidMediaTypeSyntaxOperation(call *Call, operation oas.Operation) []Finding {
	body := oas.Resolve(call.Root, oas.Get(operation.Node, "requestBody"))
	base := oas.JoinAll(operation.Path, "requestBody", "content")

	return validMediaTypeSyntaxContent(oas.Get(body, "content"), base, validMediaTypeSyntaxTypes(call))
}

// checkValidMediaTypeSyntaxComponents validates the content keys of the
// reusable request bodies.
func checkValidMediaTypeSyntaxComponents(call *Call) []Finding {
	types := validMediaTypeSyntaxTypes(call)

	return seq.FlatMapPairs(
		oas.Pairs(oas.Dig(call.Root, "components", "requestBodies")),
		func(name, body *yaml.Node) []Finding {
			base := oas.JoinAll("$.components.requestBodies", name.Value, "content")

			return validMediaTypeSyntaxContent(oas.Get(oas.Resolve(call.Root, body), "content"), base, types)
		},
	)
}

// validMediaTypeSyntaxTypes returns the accepted top-level types, lower-cased.
func validMediaTypeSyntaxTypes(call *Call) []string {
	return seq.Map(slices.Values(call.Strings(validMediaTypeSyntaxTypesOption)), Fold)
}

// validMediaTypeSyntaxContent validates one content map.
func validMediaTypeSyntaxContent(content *yaml.Node, base string, types []string) []Finding {
	return seq.FlatMapPairs(oas.Pairs(content), func(key, media *yaml.Node) []Finding {
		return When(!validMediaTypeSyntaxValid(key.Value, types), func() Finding {
			return Finding{
				Node:    media,
				Path:    oas.Join(base, key.Value),
				Message: "'" + key.Value + "' is not a valid media type",
			}
		})
	})
}

// validMediaTypeSyntaxValid reports whether name is a well formed media type.
func validMediaTypeSyntaxValid(name string, types []string) bool {
	essence, _, _ := strings.Cut(name, ";")

	kind, subtype, found := strings.Cut(strings.TrimSpace(essence), "/")

	return found && regexp.MustCompile(validMediaTypeSyntaxSubtype).MatchString(subtype) &&
		(kind == "*" || slices.Contains(types, strings.ToLower(kind)))
}
