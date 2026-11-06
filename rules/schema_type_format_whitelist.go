package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	schemaTypeFormatWhitelistTypesOption   = "types"
	schemaTypeFormatWhitelistIntegerOption = "integerFormats"
	schemaTypeFormatWhitelistNumberOption  = "numberFormats"
	schemaTypeFormatWhitelistStringOption  = "stringFormats"
	schemaTypeFormatWhitelistArray         = "array"
	schemaTypeFormatWhitelistBinary        = "binary"
)

// SchemaTypeFormatWhitelist returns the extra-schema-type-format-whitelist rule.
//
// # What it checks
//
// The type of a schema must be one of the JSON Schema primitive types and its
// format must be known for that type. Defaults: types array, boolean, integer,
// number, object, string (plus null in OpenAPI 3.1 type lists); integer
// formats int32 and int64; number formats float and double (and the integer
// formats, because numbers may be integral); string formats byte, binary, date,
// date-time, password, email, uuid, uri, url, hostname, ipv4 and ipv6. A format
// is only checked when the schema has exactly one non-null type, and the
// boolean, array and object types accept no format at all. Unknown types such
// as int or float, and formats such as int128 or datetime, are the typical
// typos this rule catches.
//
// # Where it comes from
//
// IBM's openapi-validator (ibm-schema-type-format).
//
// # Options
//
//   - types: allowed type names. Default: array, boolean, integer, number,
//     object, string, null.
//   - integerFormats: allowed integer formats. Default: int32, int64.
//   - numberFormats: allowed number formats. Default: float, double, int32, int64.
//   - stringFormats: allowed string formats (the JSON Schema 2020-12 formats
//     duration, time, uri-reference, iri and the like are included). Default: byte, binary, date,
//     date-time, password, email, uuid, uri, url, hostname, ipv4, ipv6.
//
// # Bad
//
//	components:
//	  schemas:
//	    Count:
//	      type: int
//	    When:
//	      type: string
//	      format: datetime
//
// # Good
//
//	components:
//	  schemas:
//	    Count:
//	      type: integer
//	      format: int64
//	    When:
//	      type: string
//	      format: date-time
//
// # Known false positives
//
// Custom formats that are legal in OpenAPI (for example format: money) are
// flagged unless they are added to the matching option.
func SchemaTypeFormatWhitelist() Rule {
	return New(Spec{
		Name:        "schema-type-format-whitelist",
		Description: "Schema types and formats must come from the allowed lists.",
		HowToFix:    "Use a standard type and a format that is valid for that type.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        schemaTypeFormatWhitelistTypesOption,
				Description: "Allowed schema types.",
				Default: []string{
					schemaTypeFormatWhitelistArray,
					"boolean",
					keywordInteger,
					keywordNumber,
					literalObject,
					literalString,
					keywordNull,
				},
			},
			{
				Name:        schemaTypeFormatWhitelistIntegerOption,
				Description: "Allowed formats of integer schemas.",
				Default:     []string{formatInt32, formatInt64},
			},
			{
				Name:        schemaTypeFormatWhitelistNumberOption,
				Description: "Allowed formats of number schemas.",
				Default:     []string{"float", "double", "int32", "int64"},
			},
			{
				Name:        schemaTypeFormatWhitelistStringOption,
				Description: "Allowed formats of string schemas.",
				Default: []string{
					"byte", schemaTypeFormatWhitelistBinary, formatDate, "date-time", "password", "email", literalUUID,
					"uri", "url", "hostname", "ipv4", "ipv6", "duration", "time", "uri-reference", "iri",
					"iri-reference", "idn-email", "idn-hostname", "json-pointer", "relative-json-pointer",
					"regex", "uri-template",
				},
			},
		},
		Check: PerSchema(checkSchemaTypeFormatWhitelist),
	})
}

// checkSchemaTypeFormatWhitelist reports disallowed types and formats.
func checkSchemaTypeFormatWhitelist(call *Call, schema oas.Schema) []Finding {
	allowed := call.Strings(schemaTypeFormatWhitelistTypesOption)
	typeNode := oas.Get(schema.Node, "type")
	types := seq.FlatMapSlice(oas.Types(schema.Node), func(declared string) []Finding {
		return When(!slices.Contains(allowed, declared), func() Finding {
			return Finding{
				Node:    typeNode,
				Path:    oas.Join(schema.Path, "type"),
				Message: "type " + declared + " is not allowed; use one of " + strings.Join(allowed, ", "),
			}
		})
	})

	return slices.Concat(types, schemaTypeFormatWhitelistFormat(call, schema))
}

// schemaTypeFormatWhitelistFormat reports a format that the single non-null
// type of the schema does not allow.
func schemaTypeFormatWhitelistFormat(call *Call, schema oas.Schema) []Finding {
	formatNode := oas.Get(schema.Node, "format")
	if !oas.IsScalar(formatNode) {
		return nil
	}

	types := seq.Filter(slices.Values(oas.Types(schema.Node)), func(declared string) bool {
		return declared != keywordNull
	})
	if len(types) != 1 {
		return nil
	}

	allowed, known := schemaTypeFormatWhitelistFormats(call, types[0])

	return When(known && !slices.Contains(allowed, formatNode.Value), func() Finding {
		return Finding{
			Node:    formatNode,
			Path:    oas.Join(schema.Path, "format"),
			Message: "format " + formatNode.Value + " is not allowed for type " + types[0],
		}
	})
}

// schemaTypeFormatWhitelistFormats returns the allowed formats of a type, and
// whether formats of the type are checked at all.
func schemaTypeFormatWhitelistFormats(call *Call, typeName string) ([]string, bool) {
	switch typeName {
	case "integer":
		return call.Strings(schemaTypeFormatWhitelistIntegerOption), true
	case keywordNumber:
		return call.Strings(schemaTypeFormatWhitelistNumberOption), true
	case literalString:
		return call.Strings(schemaTypeFormatWhitelistStringOption), true
	case "boolean", schemaTypeFormatWhitelistArray, "object":
		return nil, true
	default:
		return nil, false
	}
}
