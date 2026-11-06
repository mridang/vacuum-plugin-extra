package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const actionVerbsInPathVerbsOption = "verbs"

// ActionVerbsInPath returns the extra-action-verbs-in-path rule.
//
// # What it checks
//
// No literal (non-parameter) segment of a path template may be an action verb
// such as get, create, update, delete, list, fetch, remove, add, set, find,
// search, make, do, run, execute, process, handle, post or put. The segment is
// lower-cased before it is compared with the verb list, and only whole
// segments are compared, so /settings or /lists are never flagged. One finding
// is reported per operation and offending segment.
//
// # Why
//
// In a resource-oriented API the HTTP method is the verb and the path names the
// noun. Paths such as /createUser or /pets/get repeat the method, make the API
// harder to predict and push clients towards RPC-style naming.
//
// # Where it comes from
//
// ai-api-lint (AI050, path segments should not contain verbs).
//
// # Options
//
//   - verbs (list of strings, default: get, create, update, delete, fetch,
//     remove, add, set, find, search, make, do, run, execute, process, handle,
//     list, post, put): the segments that are treated as verbs. Values are
//     compared case-insensitively.
//
// # Bad
//
//	paths:
//	  /pets/create:
//	    post:
//	      responses:
//	        "201":
//	          description: created
//
// # Good
//
//	paths:
//	  /pets:
//	    post:
//	      responses:
//	        "201":
//	          description: created
//
// # Known false positives
//
// A segment that is a verb but also a legitimate noun, for example /set in a
// trading-card API or /run in a CI API, is flagged. Remove the entry from the
// verbs option to allow it. Action-style endpoints defined by design (for
// example /orders/{id}/cancel) are not flagged unless the verb is in the list.
func ActionVerbsInPath() Rule {
	return New(Spec{
		Name:        "action-verbs-in-path",
		Description: "Path segments must be nouns, not action verbs such as create or delete.",
		HowToFix:    "Rename the path segment to a noun and let the HTTP method express the action.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        actionVerbsInPathVerbsOption,
			Description: "List of segments treated as action verbs.",
			Default:     actionVerbsInPathDefaults(),
		}},
		Check: PerOperation(checkActionVerbsInPath),
	})
}

func checkActionVerbsInPath(call *Call, operation oas.Operation) []Finding {
	verbs := seq.Map(slices.Values(call.Strings(actionVerbsInPathVerbsOption)), Fold)

	return seq.FlatMapSlice(actionVerbsInPathSegments(operation.Route), func(segment string) []Finding {
		return When(slices.Contains(verbs, Fold(segment)), func() Finding {
			return Finding{
				Node:    oas.Get(operation.PathItem, operation.Method),
				Path:    operation.Path,
				Message: "path segment '" + segment + "' is an action verb; use a noun and express the action with the HTTP method",
			}
		})
	})
}

func actionVerbsInPathSegments(route string) []string {
	return seq.Filter(strings.SplitSeq(route, "/"), func(segment string) bool {
		return segment != "" && !strings.Contains(segment, "{")
	})
}

// actionVerbsInPathDefaults returns the default list of action verbs.
func actionVerbsInPathDefaults() []string {
	return strings.Fields(
		"get create update delete fetch remove add set find search make do run execute process handle list post put",
	)
}
