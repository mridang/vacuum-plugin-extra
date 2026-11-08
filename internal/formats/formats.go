// Package formats holds the knowledge about string formats that the rules
// demanding string constraints share.
package formats

import "slices"

// NonEmpty returns the string formats whose values are never the empty string
// and whose shape the format already describes, so a minLength or a pattern
// next to them adds nothing. A maxLength can still be useful for the unbounded
// ones such as uri and email. The list is a fresh copy on every call.
func NonEmpty() []string {
	return []string{
		"date",
		"date-time",
		"time",
		"duration",
		"uuid",
		"uri",
		"uri-reference",
		"iri",
		"url",
		"email",
		"idn-email",
		"hostname",
		"idn-hostname",
		"ipv4",
		"ipv6",
		"json-pointer",
		"regex",
		"byte",
	}
}

// Contains reports whether format is one of the given formats. An empty format
// never matches.
func Contains(list []string, format string) bool {
	return format != "" && slices.Contains(list, format)
}
