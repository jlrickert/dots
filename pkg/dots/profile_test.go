package dots_test

import (
	"errors"
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func TestParseProfile_ExtendsScalar(t *testing.T) {
	p, err := dots.ParseProfile([]byte("name: laptop\nextends: base\n"))
	require.NoError(t, err)
	require.Equal(t, dots.StringList{"base"}, p.Extends)
}

func TestParseProfile_ExtendsList(t *testing.T) {
	p, err := dots.ParseProfile([]byte("name: work\nextends: [base, dev]\npackages: [work/zellij]\nexclude: [personal/tmux]\n"))
	require.NoError(t, err)
	require.Equal(t, dots.StringList{"base", "dev"}, p.Extends)
	require.Equal(t, []string{"work/zellij"}, p.Packages)
	require.Equal(t, []string{"personal/tmux"}, p.Exclude)
}

func TestParseProfile_ExtendsInvalid(t *testing.T) {
	_, err := dots.ParseProfile([]byte("name: bad\nextends: {a: b}\n"))
	require.ErrorIs(t, err, dots.ErrParse)
}

func TestMarshalProfile_ExtendsRoundTrip(t *testing.T) {
	cases := map[string]struct {
		extends dots.StringList
		want    string
	}{
		"single stays scalar": {dots.StringList{"base"}, "extends: base\n"},
		"multiple is a list":  {dots.StringList{"base", "dev"}, "extends:\n    - base\n    - dev\n"},
		"empty is omitted":    {nil, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			data, err := dots.MarshalProfile(&dots.Profile{Name: "p", Extends: tc.extends, Packages: []string{}})
			require.NoError(t, err)
			if tc.want == "" {
				require.NotContains(t, string(data), "extends")
			} else {
				require.Contains(t, string(data), tc.want)
			}

			back, err := dots.ParseProfile(data)
			require.NoError(t, err)
			require.Equal(t, tc.extends, back.Extends)
		})
	}
}

func TestProfileCycleError(t *testing.T) {
	err := &dots.ProfileCycleError{Chain: []string{"work", "dev", "work"}}
	require.True(t, errors.Is(err, dots.ErrConflict))
	require.Equal(t, "profile extends cycle: work → dev → work", err.Error())
}
