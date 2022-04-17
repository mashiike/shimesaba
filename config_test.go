package shimesaba_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

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
			casename: "v0.7.0 over",
			paths:    []string{"testdata/v0.7.0.yaml"},
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

func TestConfigLoadError(t *testing.T) {
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
		{
			casename: "sample_config",
			paths:    []string{"testdata/sample.yaml"},
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
			require.Error(t, err)
		})
	}
}

func TestDefinitionConfigStartAt(t *testing.T) {
	cases := []struct {
		now      time.Time
		backfill int
		cfg      *shimesaba.DefinitionConfig
		expected time.Time
	}{
		{
			now:      time.Date(2022, 1, 14, 3, 13, 23, 999, time.UTC),
			backfill: 3,
			cfg: &shimesaba.DefinitionConfig{
				ID:                "test",
				ServiceName:       "shimesaba",
				TimeFrame:         "1d",
				ErrorBudgetSize:   0.05,
				CalculateInterval: "1h",
			},
			expected: time.Date(2022, 1, 13, 0, 0, 0, 0, time.UTC),
		},
		{
			now:      time.Date(2022, 1, 14, 3, 13, 23, 999, time.UTC),
			backfill: 3,
			cfg: &shimesaba.DefinitionConfig{
				ID:                "test",
				ServiceName:       "shimesaba",
				TimeFrame:         "365d",
				ErrorBudgetSize:   0.05,
				CalculateInterval: "1d",
			},
			expected: time.Date(2021, 1, 11, 0, 0, 0, 0, time.UTC),
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case.%d", i), func(t *testing.T) {
			require.NoError(t, c.cfg.Restrict())
			actual := c.cfg.StartAt(c.now, c.backfill)
			require.EqualValues(t, c.expected, actual)
		})
	}
}

func TestDefinitionConfigErrorBudgetSize(t *testing.T) {
	cases := []struct {
		cfg         *shimesaba.DefinitionConfig
		exceptedErr bool
		expected    float64
	}{
		{
			cfg: &shimesaba.DefinitionConfig{
				ID:                "test",
				ServiceName:       "shimesaba",
				TimeFrame:         "28d",
				ErrorBudgetSize:   0.001,
				CalculateInterval: "1h",
			},
			expected: 0.001,
		},
		{
			cfg: &shimesaba.DefinitionConfig{
				ID:                "test",
				ServiceName:       "shimesaba",
				TimeFrame:         "28d",
				ErrorBudgetSize:   "40m",
				CalculateInterval: "1d",
			},
			expected: 0.001,
		},
		{
			cfg: &shimesaba.DefinitionConfig{
				ID:                "test",
				ServiceName:       "shimesaba",
				TimeFrame:         "28d",
				ErrorBudgetSize:   "0.1%",
				CalculateInterval: "1d",
			},
			expected: 0.001,
		},
		{
			cfg: &shimesaba.DefinitionConfig{
				ID:                "test",
				ServiceName:       "shimesaba",
				TimeFrame:         "28d",
				ErrorBudgetSize:   "5m0.001%",
				CalculateInterval: "1d",
			},
			exceptedErr: true,
		},
		{
			cfg: &shimesaba.DefinitionConfig{
				ID:                "test",
				ServiceName:       "shimesaba",
				TimeFrame:         "28d",
				ErrorBudgetSize:   "0.01",
				CalculateInterval: "1d",
			},
			exceptedErr: true,
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case.%d", i), func(t *testing.T) {
			err := c.cfg.Restrict()
			if !c.exceptedErr {
				require.NoError(t, err)
				require.InEpsilon(t, c.expected, c.cfg.ErrorBudgetSizeParcentage(), 0.01)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestDefinitionConfigMetricPrefixSuffix(t *testing.T) {
	cases := []struct {
		cfg            *shimesaba.DefinitionConfig
		expectedPrefix string
		expectedSuffix string
	}{
		{
			cfg: &shimesaba.DefinitionConfig{
				ID:                "test",
				ServiceName:       "shimesaba",
				TimeFrame:         "28d",
				ErrorBudgetSize:   0.001,
				CalculateInterval: "1h",
			},
			expectedPrefix: "shimesaba",
			expectedSuffix: "test",
		},
		{
			cfg: &shimesaba.DefinitionConfig{
				ID:                "test",
				ServiceName:       "shimesaba",
				TimeFrame:         "28d",
				ErrorBudgetSize:   0.001,
				CalculateInterval: "1h",
				MetricPrefix:      "hoge",
				MetricSuffix:      "fuga",
			},
			expectedPrefix: "hoge",
			expectedSuffix: "fuga",
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case.%d", i), func(t *testing.T) {
			err := c.cfg.Restrict()
			require.NoError(t, err)
			require.Equal(t, c.expectedPrefix, c.cfg.MetricPrefix)
		})
	}
}
