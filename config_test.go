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
			casename: "v1.0.0 over simple config",
			paths:    []string{"testdata/v1.0.0_simple.yaml"},
		},
		{
			casename: "v1.0.0 over check destination",
			paths:    []string{"testdata/v1.0.0_destination.yaml"},
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

func TestSLOConfigStartAt(t *testing.T) {
	cases := []struct {
		now      time.Time
		backfill int
		cfg      *shimesaba.SLOConfig
		expected time.Time
	}{
		{
			now:      time.Date(2022, 1, 14, 3, 13, 23, 999, time.UTC),
			backfill: 3,
			cfg: &shimesaba.SLOConfig{
				ID:            "test",
				RollingPeriod: "1d",
				Destination: &shimesaba.DestinationConfig{
					ServiceName: "shimesaba",
				},
				CalculateInterval: "1h",
				ErrorBudgetSize:   0.05,
			},
			expected: time.Date(2022, 1, 13, 0, 0, 0, 0, time.UTC),
		},
		{
			now:      time.Date(2022, 1, 14, 3, 13, 23, 999, time.UTC),
			backfill: 3,
			cfg: &shimesaba.SLOConfig{
				ID:            "test",
				RollingPeriod: "365d",
				Destination: &shimesaba.DestinationConfig{
					ServiceName: "shimesaba",
				},
				CalculateInterval: "1d",
				ErrorBudgetSize:   0.05,
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

func TestSLOConfigErrorBudgetSize(t *testing.T) {
	cases := []struct {
		cfg         *shimesaba.SLOConfig
		exceptedErr bool
		expected    float64
	}{
		{
			cfg: &shimesaba.SLOConfig{
				ID:            "test",
				RollingPeriod: "28d",
				Destination: &shimesaba.DestinationConfig{
					ServiceName: "shimesaba",
				},
				CalculateInterval: "1h",
				ErrorBudgetSize:   0.001,
			},
			expected: 0.001,
		},
		{
			cfg: &shimesaba.SLOConfig{
				ID:            "test",
				RollingPeriod: "28d",
				Destination: &shimesaba.DestinationConfig{
					ServiceName: "shimesaba",
				},
				CalculateInterval: "1d",
				ErrorBudgetSize:   "40m",
			},
			expected: 0.001,
		},
		{
			cfg: &shimesaba.SLOConfig{
				ID:            "test",
				RollingPeriod: "28d",
				Destination: &shimesaba.DestinationConfig{
					ServiceName: "shimesaba",
				},
				CalculateInterval: "1d",
				ErrorBudgetSize:   "0.1%",
			},
			expected: 0.001,
		},
		{
			cfg: &shimesaba.SLOConfig{
				ID:            "test",
				RollingPeriod: "28d",
				Destination: &shimesaba.DestinationConfig{
					ServiceName: "shimesaba",
				},
				CalculateInterval: "1d",
				ErrorBudgetSize:   "5m0.001%",
			},
			exceptedErr: true,
		},
		{
			cfg: &shimesaba.SLOConfig{
				ID:            "test",
				RollingPeriod: "28d",
				Destination: &shimesaba.DestinationConfig{
					ServiceName: "shimesaba",
				},
				CalculateInterval: "1d",
				ErrorBudgetSize:   "0.01",
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

func TestSLOConfigMetricPrefixSuffix(t *testing.T) {
	cases := []struct {
		cfg            *shimesaba.SLOConfig
		expectedPrefix string
		expectedSuffix string
	}{
		{
			cfg: &shimesaba.SLOConfig{
				ID:            "test",
				RollingPeriod: "28d",
				Destination: &shimesaba.DestinationConfig{
					ServiceName: "shimesaba",
				},
				CalculateInterval: "1h",
				ErrorBudgetSize:   0.001,
			},
			expectedPrefix: "shimesaba",
			expectedSuffix: "test",
		},
		{
			cfg: &shimesaba.SLOConfig{
				ID:            "test",
				RollingPeriod: "28d",
				Destination: &shimesaba.DestinationConfig{
					ServiceName:  "shimesaba",
					MetricPrefix: "hoge",
					MetricSuffix: "fuga",
				},
				CalculateInterval: "1h",
				ErrorBudgetSize:   0.001,
			},
			expectedPrefix: "hoge",
			expectedSuffix: "fuga",
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case.%d", i), func(t *testing.T) {
			err := c.cfg.Restrict()
			require.NoError(t, err)
			require.Equal(t, c.expectedPrefix, c.cfg.Destination.MetricPrefix)
		})
	}
}
