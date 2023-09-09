package payload

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	for m, tc := range map[string]struct {
		format   Format
		expected map[string]string
	}{
		"mime": {
			Format("text/spdx"),
			map[string]string{"mime": "text/spdx", "version": "", "encoding": ""},
		},
		"mime+version": {
			Format("text/spdx;version=2.3"),
			map[string]string{"mime": "text/spdx", "version": "2.3", "encoding": ""},
		},
		"mime+encoding": {
			Format("text/spdx+json"),
			map[string]string{"mime": "text/spdx", "version": "", "encoding": "json"},
		},
		"mime+encoding+version": {
			Format("text/spdx+json;version=2.3"),
			map[string]string{"mime": "text/spdx", "version": "2.3", "encoding": "json"},
		},
	} {
		res := tc.format.Parse()
		require.Equal(t, tc.expected, res, m)
	}
}
