package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// FormatMatchesType returns the extra-format-matches-type rule.
//
// # What it checks
//
// Every schema that declares both a type and a standard format must use a type
// that the format applies to. The built-in table is: int32 and int64 apply to
// integer; float and double to number (integer is accepted too, as the
// specification allows it); byte, binary, date, date-time, password, email,
// idn-email, hostname, idn-hostname, ipv4, ipv6, uri, uri-reference, iri,
// iri-reference, uuid, duration, time, regex, json-pointer and
// relative-json-pointer to string. Formats that are not in the table are
// ignored. In OpenAPI 3.1, where type may be a list, the format is accepted when
// any listed type matches.
//
// # Why
//
// A format narrows a type. date-time on an integer or int32 on a string is
// meaningless, and code generators either ignore the format or fail on it.
//
// # Where it comes from
//
// KaiZen OpenAPI Parser (format and type must agree).
//
// # Options
//
// None.
//
// # Bad
//
//	type: string
//	format: int32
//
// # Good
//
//	type: integer
//	format: int32
//
// # Known false positives
//
// Vendor formats that happen to reuse a standard name, for example a numeric
// "time" format expressed as an integer, are flagged.
func FormatMatchesType() Rule {
	return New(Spec{
		Name:        "format-matches-type",
		Description: "Standard formats must be used with a compatible type.",
		HowToFix:    "Change the type to one the format applies to, or use a different format.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkFormatMatchesType),
	})
}

// formatMatchesTypeAllowed returns the types a standard format may be used with.
func formatMatchesTypeAllowed(format string) []string {
	stringFormats := strings.Fields(
		"byte binary date date-time password email idn-email hostname idn-hostname ipv4 ipv6 " +
			"uri uri-reference iri iri-reference uuid duration time regex json-pointer relative-json-pointer",
	)

	switch {
	case format == formatInt32 || format == formatInt64:
		return []string{keywordInteger}
	case format == "float" || format == "double":
		return []string{keywordNumber, keywordInteger}
	case slices.Contains(stringFormats, format):
		return []string{literalString}
	default:
		return nil
	}
}

// checkFormatMatchesType compares the format and type of every schema.
func checkFormatMatchesType(_ *Call, schema oas.Schema) []Finding {
	key, format := oas.Entry(schema.Node, "format")
	allowed := formatMatchesTypeAllowed(oas.Text(format))
	types := oas.Types(schema.Node)

	compatible := slices.ContainsFunc(types, func(kind string) bool {
		return slices.Contains(allowed, kind)
	})

	return When(len(allowed) > 0 && len(types) > 0 && !compatible, func() Finding {
		return Finding{
			Node: format,
			Path: oas.Join(schema.Path, key.Value),
			Message: "format '" + format.Value + "' cannot be used with type " + strings.Join(types, ", ") +
				"; it applies to " + strings.Join(allowed, ", "),
		}
	})
}
