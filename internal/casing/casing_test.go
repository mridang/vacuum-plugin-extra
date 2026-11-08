package casing_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
)

// sample is a multi-word name written in one casing; no sample follows another casing.
type sample struct {
	convention casing.Casing
	name       string
}

func samples() []sample {
	return []sample{
		{casing.Kebab, "pet-owner"},
		{casing.Snake, "pet_owner"},
		{casing.Camel, "petOwner"},
		{casing.Pascal, "PetOwner"},
		{casing.ScreamingSnake, "PET_OWNER"},
		{casing.ScreamingKebab, "PET-OWNER"},
	}
}

// sampleOf returns the sample name of a casing, and the flat name for lowercase.
func sampleOf(convention casing.Casing) string {
	if convention == casing.Lower {
		return "petowner"
	}

	return samples()[slices.IndexFunc(samples(), func(s sample) bool { return s.convention == convention })].name
}

func TestCasingMatches(t *testing.T) {
	t.Parallel()

	for _, convention := range casing.All() {
		t.Run(string(convention), func(t *testing.T) {
			t.Parallel()

			for _, s := range samples() {
				require.Equal(
					t,
					s.convention == convention,
					convention.Matches(s.name),
					"%s against %s",
					s.name,
					convention,
				)
			}
		})
	}
}

func TestCasingMatchesEdgeCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		convention casing.Casing
		value      string
		want       bool
	}{
		{"lower accepts one flat word", casing.Lower, "petowner", true},
		{"lower accepts digits", casing.Lower, "v2", true},
		{"lower rejects separators", casing.Lower, "pet-owner", false},
		{"lower rejects capitals", casing.Lower, "petOwner", false},
		{"single word is kebab", casing.Kebab, "pets", true},
		{"single word is snake", casing.Snake, "pets", true},
		{"single word is camel", casing.Camel, "pets", true},
		{"single capitalised word is pascal", casing.Pascal, "Pets", true},
		{"single capital word is screaming snake", casing.ScreamingSnake, "PETS", true},
		{"digits are word characters", casing.Kebab, "api-v2", true},
		{"acronym runs are camel", casing.Camel, "getHTTPStatus", true},
		{"acronym runs are pascal", casing.Pascal, "HTTPError", true},
		{"pascal rejects a leading digit", casing.Pascal, "2Pets", false},
		{"empty name matches nothing", casing.Kebab, "", false},
		{"empty name is not camel", casing.Camel, "", false},
		{"empty name is not lower", casing.Lower, "", false},
		{"doubled separator is rejected", casing.Snake, "pet__owner", false},
		{"leading separator is rejected", casing.Kebab, "-pet", false},
		{"trailing separator is rejected", casing.Snake, "pet_", false},
		{"mixed separators are rejected", casing.Kebab, "pet-owner_id", false},
		{"non ascii letters are rejected", casing.Snake, "pét", false},
		{"unknown casing matches nothing", casing.Casing("shouty"), "pet", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, tc.convention.Matches(tc.value))
		})
	}
}

func TestCasingConvert(t *testing.T) {
	t.Parallel()

	for _, convention := range casing.All() {
		for _, s := range samples() {
			require.Equal(t, sampleOf(convention), convention.Convert(s.name), "%s to %s", s.name, convention)
			require.True(t, convention.Matches(convention.Convert(s.name)))
		}
	}
}

func TestCasingConvertKeepsAcronymsAndSplitsOnSeparators(t *testing.T) {
	t.Parallel()

	require.Equal(t, "user_id", casing.Snake.Convert("userID"))
	require.Equal(t, "UserId", casing.Pascal.Convert("user-id"))
	require.Equal(t, "userID", casing.Camel.Convert("user_ID"))
	require.Equal(t, "pet-owner-v2", casing.Kebab.Convert("pet.owner v2"))
	require.Equal(t, "***", casing.Kebab.Convert("***"))
	require.Empty(t, casing.Kebab.Convert(""))
	require.Equal(t, "pet", casing.Casing("shouty").Convert("pet"))
}

func TestParse(t *testing.T) {
	t.Parallel()

	for _, convention := range casing.All() {
		parsed, err := casing.Parse(string(convention))
		require.NoError(t, err)
		require.Equal(t, convention, parsed)
	}

	_, err := casing.Parse("snake")
	require.EqualError(t, err, `unknown casing "snake": allowed values are `+
		`kebab-case, snake_case, camelCase, PascalCase, SCREAMING_SNAKE_CASE, SCREAMING-KEBAB-CASE, lowercase`)
	require.ErrorAs(t, err, &casing.UnknownError{})
}

func TestSet(t *testing.T) {
	t.Parallel()

	set, err := casing.ParseSet([]string{"snake_case", "SCREAMING_SNAKE_CASE"})
	require.NoError(t, err)
	require.Equal(t, "snake_case or SCREAMING_SNAKE_CASE", set.String())
	require.True(t, set.Matches("pet_owner"))
	require.True(t, set.Matches("PET_OWNER"))
	require.False(t, set.Matches("petOwner"))
	require.Equal(t, "pet_owner", set.Convert("petOwner"))

	_, err = casing.ParseSet([]string{"snake_case", "nope"})
	require.ErrorContains(t, err, `unknown casing "nope"`)

	empty, err := casing.ParseSet(nil)
	require.NoError(t, err)
	require.True(t, empty.Matches("Anything-goes"))
	require.Equal(t, "Anything-goes", empty.Convert("Anything-goes"))
}
