package shimesaba_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/mashiike/shimesaba"
	"github.com/mashiike/shimesaba/internal/logger"
	"github.com/stretchr/testify/require"
)

func TestConfigLoadNoError(t *testing.T) {
	os.Setenv("TARGET_ALB_NAME", "dummy-alb")
	os.Setenv("POST_METRIC_SERVICE", "dummy-service")
	cases := []struct {
		casename string
		paths    []string
	}{
		{
			casename: "default_config",
			paths:    []string{"_example/default.yaml"},
		},
		{
			casename: "simple_config",
			paths:    []string{"testdata/simple.yaml"},
		},
		{
			casename: "alert_source_config",
			paths:    []string{"testdata/alert_source.yaml"},
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			var buf bytes.Buffer
			logger.Setup(&buf, "debug")
			defer func() {
				t.Log(buf.String())
				logger.Setup(os.Stderr, "info")
			}()
			cfg := shimesaba.NewDefaultConfig()
			err := cfg.Load(c.paths...)
			require.NoError(t, err)
			err = cfg.Restrict()
			require.NoError(t, err)
		})
	}

}
