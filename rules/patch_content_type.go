package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// PatchContentType returns the extra-patch-content-type rule.
//
// # What it checks
//
// The request body of a PATCH operation may offer only the patch media types:
// application/json-patch+json and application/merge-patch+json. Any other key
// of the request body content map, such as application/json, is reported.
// Parameters after a semicolon and letter case are ignored. $ref request
// bodies are resolved.
//
// # Why
//
// A PATCH body is a description of a change, not a representation of the
// resource. Declaring it as plain application/json hides which patch format the
// server understands, so clients cannot know whether to send a JSON Patch
// document or a merge patch.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (ibm-patch-request-content-type).
//
// # Options
//
//   - allowed (list of strings, default application/json-patch+json and
//     application/merge-patch+json): the accepted media types.
//
// # Bad
//
//	patch:
//	  requestBody:
//	    content:
//	      application/json:
//	        schema: {type: object}
//
// # Good
//
//	patch:
//	  requestBody:
//	    content:
//	      application/merge-patch+json:
//	        schema: {type: object}
//
// # Known false positives
//
// APIs that implement PATCH with a vendor media type are flagged unless the
// type is added to the allowed option.
func PatchContentType() Rule {
	return New(Spec{
		Name:        "patch-content-type",
		Description: "PATCH request bodies must use a JSON Patch or JSON Merge Patch media type.",
		HowToFix:    "Use application/json-patch+json or application/merge-patch+json for the PATCH request body.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        "allowed",
			Description: "Media types accepted for PATCH request bodies.",
			Default:     []string{"application/json-patch+json", "application/merge-patch+json"},
		}},
		Check: PerOperation(checkPatchContentType),
	})
}

// checkPatchContentType reports disallowed media types of PATCH bodies.
func checkPatchContentType(call *Call, operation oas.Operation) []Finding {
	if operation.Method != methodPatch {
		return nil
	}

	allowed := seq.Map(slices.Values(call.Strings("allowed")), Fold)
	body := oas.Resolve(call.Root, oas.Get(operation.Node, "requestBody"))
	base := oas.JoinAll(operation.Path, "requestBody", "content")

	return seq.FlatMapPairs(oas.Pairs(oas.Get(body, "content")), func(key, media *yaml.Node) []Finding {
		name, _, _ := strings.Cut(key.Value, ";")

		return When(!slices.Contains(allowed, Fold(name)), func() Finding {
			return Finding{
				Node:    media,
				Path:    oas.Join(base, key.Value),
				Message: "PATCH request body must not use media type " + key.Value,
			}
		})
	})
}
