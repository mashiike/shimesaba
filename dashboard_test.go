package shimesaba_test

import (
	"testing"

	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

func TestLoader(t *testing.T) {
	cases := []struct {
		tpl      string
		expected string
	}{
		{
			tpl:      "{{ file `dummy.md` }}",
			expected: "hoge\n",
		},
		{
			tpl:      "{{ to_percentage `0.001` }}",
			expected: "0.1",
		},
		{
			tpl:      "{{ to_percentage 0.0001 }}",
			expected: "0.01",
		},
		{
			tpl:      "{{ eval_expr `10` }}",
			expected: "10",
		},
		{
			tpl:      "{{ eval_expr `10 > var1` 0 }}",
			expected: "true",
		},
		{
			tpl:      "{{ eval_expr `request_success_rate >= 0.95` `request_success_rate` 0.90 }}",
			expected: "false",
		},
	}

	for _, c := range cases {
		t.Run(c.tpl, func(t *testing.T) {
			loader := shimesaba.NewLoader("testdata")
			actual, err := loader.ReadWithEnvBytes([]byte(c.tpl))
			require.NoError(t, err)
			require.EqualValues(t, c.expected, string(actual))
		})
	}
}
