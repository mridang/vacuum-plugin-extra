package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const serverXInternalFalse = "false"

const serverXInternalStringTag = "!!str"

const serverXInternalExtension = "extension"

// ServerXInternal returns the extra-server-x-internal rule.
//
// # What it checks
//
// Every server object, in the root servers list, on path items and on
// operations, must define the x-internal extension with a boolean value (true
// or false). The name of the extension can be changed with an option.
//
// # Why
//
// Documents that describe several environments often mix internal servers
// (staging, intranet gateways) with public ones. A mandatory marker makes the
// visibility an explicit decision for every server instead of an implicit
// default, so documentation tooling can reliably filter out internal servers
// before publishing.
//
// # Where it comes from
//
// A Spectral ruleset convention (an x-internal property on every server).
//
// # Options
//
//   - extension (string, default "x-internal"): the name of the extension that
//     each server must define as a boolean.
//
// # Bad
//
//	servers:
//	  - url: https://api.example.com
//
// # Good
//
//	servers:
//	  - url: https://api.example.com
//	    x-internal: false
//	  - url: https://staging.example.com
//	    x-internal: true
//
// # Known false positives and limitations
//
// The extension must be a literal YAML boolean; the string "true" is rejected.
// An empty extension option disables the rule. Servers declared only through
// $ref'd path items are checked where the path item is resolved.
func ServerXInternal() Rule {
	return New(Spec{
		Name:        "server-x-internal",
		Description: "Every server object must define x-internal as true or false.",
		HowToFix:    "Add x-internal: true or x-internal: false to the server object.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        serverXInternalExtension,
				Description: "Name of the boolean extension each server must define.",
				Default:     "x-internal",
			},
		},
		Check: OnDocument(checkServerXInternal),
	})
}

// checkServerXInternal inspects the servers of the document, path items and operations.
func checkServerXInternal(call *Call) []Finding {
	extension := call.String(serverXInternalExtension)
	if extension == "" {
		return nil
	}

	servers := func(owner *yaml.Node, path string) []Finding {
		return seq.FlatMapPairs(oas.Items(oas.Get(owner, "servers")), func(i int, server *yaml.Node) []Finding {
			return serverXInternalCheck(server, oas.Index(oas.Join(path, "servers"), i), extension)
		})
	}

	return slices.Concat(
		servers(call.Root, "$"),
		seq.FlatMap(oas.PathItems(call.Root), func(item oas.PathItem) []Finding {
			return servers(item.Node, item.Path)
		}),
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
			return servers(operation.Node, operation.Path)
		}),
	)
}

// serverXInternalCheck checks one server object.
func serverXInternalCheck(server *yaml.Node, path string, extension string) []Finding {
	if !oas.IsMap(server) {
		return nil
	}

	key, value := oas.Entry(server, extension)

	switch {
	case value == nil:
		return []Finding{{Node: server, Path: path, Message: "server must define " + extension}}
	case !oas.IsScalar(value) || (value.Value != "true" && value.Value != serverXInternalFalse) || value.Tag == serverXInternalStringTag:
		return []Finding{{Node: value, Path: oas.Join(path, key.Value), Message: extension + " must be true or false"}}
	default:
		return nil
	}
}
