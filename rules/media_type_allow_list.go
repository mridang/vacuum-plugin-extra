package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	// mediaTypeAllowListAllowed is the option with the allowed media types.
	mediaTypeAllowListAllowed = "allowed"
	// mediaTypeAllowListResponses is the option that extends the check to responses.
	mediaTypeAllowListResponses = "includeResponses"
)

// MediaTypeAllowList returns the extra-media-type-allow-list rule.
//
// # What it checks
//
// Every media type of a request body (operation request bodies and
// components.requestBodies) must appear in the configured allow-list. Entries
// are compared case-insensitively after removing media type parameters such as
// charset, and an entry of the form type/* allows every subtype of that type.
// With includeResponses the media types of responses are checked too.
//
// # Why
//
// Restricting the media types keeps an API predictable and prevents accidental
// support for formats that the server does not parse or that clients cannot
// generate.
//
// # Where it comes from
//
// Redocly CLI (request-mime-type, with the allowed list configured per rule).
//
// # Options
//
//   - allowed (list of strings, default empty): the permitted media types. The
//     rule does nothing when the list is empty.
//   - includeResponses (bool, default false): also check response content.
//
// # Bad
//
// With allowed: [application/json]:
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        content:
//	          application/xml:
//	            schema:
//	              type: object
//
// # Good
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        content:
//	          application/json:
//	            schema:
//	              type: object
//
// # Known false positives / limitations
//
// Wildcards in the document (such as */*) are compared literally, so they only
// pass when the allow-list contains them. A $ref request body is reported once, at the
// component it points to.
func MediaTypeAllowList() Rule {
	return New(Spec{
		Name:        "media-type-allow-list",
		Description: "Media types must be in the configured allow-list.",
		HowToFix:    "Use one of the allowed media types or extend the allowed option.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        mediaTypeAllowListAllowed,
				Description: "Permitted media types; the rule is inactive when empty.",
				Default:     []string{},
			},
			{
				Name:        mediaTypeAllowListResponses,
				Description: "Also check the media types of responses.",
				Default:     false,
			},
		},
		Check: Concat(PerOperation(checkMediaTypeAllowListOperation), OnDocument(checkMediaTypeAllowListComponents)),
	})
}

// checkMediaTypeAllowListOperation reports request and response media types
// outside the allow-list.
func checkMediaTypeAllowListOperation(call *Call, operation oas.Operation) []Finding {
	allowed := mediaTypeAllowListBases(call)
	body := oas.Get(operation.Node, "requestBody")
	request := mediaTypeAllowListRequest(body, oas.Join(operation.Path, "requestBody"), allowed)
	if !call.Bool(mediaTypeAllowListResponses) {
		return request
	}

	return slices.Concat(
		request,
		seq.FlatMapSlice(oas.Responses(call.Root, operation), func(response oas.Response) []Finding {
			return mediaTypeAllowListContent(response.Node, response.Path, allowed)
		}),
	)
}

// checkMediaTypeAllowListComponents reports the media types of the reusable
// request bodies that are outside the allow-list.
func checkMediaTypeAllowListComponents(call *Call) []Finding {
	allowed := mediaTypeAllowListBases(call)

	return seq.FlatMapPairs(
		oas.Pairs(oas.Dig(call.Root, "components", "requestBodies")),
		func(key, value *yaml.Node) []Finding {
			path := oas.JoinAll("$.components.requestBodies", key.Value)

			return mediaTypeAllowListContent(oas.Resolve(call.Root, value), path, allowed)
		},
	)
}

// mediaTypeAllowListBases returns the configured allow-list entries without
// parameters, lower-cased.
func mediaTypeAllowListBases(call *Call) []string {
	return seq.Map(slices.Values(call.Strings(mediaTypeAllowListAllowed)), mediaTypeAllowListBase)
}

// mediaTypeAllowListRequest checks an inline request body.
func mediaTypeAllowListRequest(body *yaml.Node, path string, allowed []string) []Finding {
	if oas.Ref(body) != "" {
		return nil
	}

	return mediaTypeAllowListContent(body, path, allowed)
}

// mediaTypeAllowListContent checks the content map of one owner. The rule is
// inactive when the allow-list is empty.
func mediaTypeAllowListContent(owner *yaml.Node, path string, allowed []string) []Finding {
	if len(allowed) == 0 {
		return nil
	}

	return seq.FlatMapSlice(
		oas.MediaTypes(oas.Get(owner, "content"), oas.Join(path, "content")),
		func(media oas.MediaType) []Finding {
			return When(!mediaTypeAllowListPermits(allowed, media.Name), func() Finding {
				return Finding{
					Path:    media.Path,
					Message: "media type " + media.Name + " is not in the allow-list",
				}
			})
		},
	)
}

// mediaTypeAllowListPermits reports whether name matches an allow-list entry.
func mediaTypeAllowListPermits(allowed []string, name string) bool {
	base := mediaTypeAllowListBase(name)
	kind, _, _ := strings.Cut(base, "/")

	return slices.Contains(allowed, base) || slices.Contains(allowed, kind+"/*")
}

// mediaTypeAllowListBase lower-cases a media type and strips its parameters.
func mediaTypeAllowListBase(name string) string {
	base, _, _ := strings.Cut(name, ";")

	return Fold(base)
}
