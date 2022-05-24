package shimesaba_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Songmu/flextime"
	"github.com/mashiike/shimesaba"
	"github.com/mashiike/shimesaba/internal/logger"
	"github.com/stretchr/testify/require"
)

func TestDefinition(t *testing.T) {
	restore := flextime.Fix(time.Date(2021, 10, 01, 0, 22, 0, 0, time.UTC))
	defer restore()
	alerts := shimesaba.Alerts{
		shimesaba.NewAlert(
			shimesaba.NewMonitor(
				"hogera",
				"hogera.example.com",
				"external",
			),
			time.Date(2021, 10, 1, 0, 3, 0, 0, time.UTC),
			ptrTime(time.Date(2021, 10, 1, 0, 9, 0, 0, time.UTC)),
		),
		shimesaba.NewAlert(
			shimesaba.NewMonitor(
				"hogera",
				"hogera.example.com",
				"external",
			),
			time.Date(2021, 10, 1, 0, 15, 0, 0, time.UTC),
			nil,
		),
	}
	cases := []struct {
		defCfg   *shimesaba.SLOConfig
		expected []*shimesaba.Report
	}{
		{
			defCfg: &shimesaba.SLOConfig{
				ID: "alert_and_metric_mixing",
				Destination: &shimesaba.DestinationConfig{
					ServiceName: "test",
				},
				RollingPeriod:     "10m",
				CalculateInterval: "5m",
				ErrorBudgetSize:   0.3,
				AlertBasedSLI: []*shimesaba.AlertBasedSLIConfig{
					{
						MonitorID: "hogera",
					},
				},
			},
			expected: []*shimesaba.Report{
				{
					DefinitionID: "alert_and_metric_mixing",
					Destination: &shimesaba.Destination{
						ServiceName:  "test",
						MetricPrefix: "shimesaba",
						MetricSuffix: "alert_and_metric_mixing",
					},
					DataPoint:              time.Date(2021, 10, 01, 0, 10, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 0, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 9, 59, 999999999, time.UTC),
					UpTime:                 4 * time.Minute,
					FailureTime:            6 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            -3 * time.Minute,
					ErrorBudgetConsumption: 4 * time.Minute,
				},
				{
					DefinitionID: "alert_and_metric_mixing",
					Destination: &shimesaba.Destination{
						ServiceName:  "test",
						MetricPrefix: "shimesaba",
						MetricSuffix: "alert_and_metric_mixing",
					},
					DataPoint:              time.Date(2021, 10, 01, 0, 15, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 5, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 14, 59, 999999999, time.UTC),
					UpTime:                 6 * time.Minute,
					FailureTime:            4 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            -1 * time.Minute,
					ErrorBudgetConsumption: 0 * time.Minute,
				},
				{
					DefinitionID: "alert_and_metric_mixing",
					Destination: &shimesaba.Destination{
						ServiceName:  "test",
						MetricPrefix: "shimesaba",
						MetricSuffix: "alert_and_metric_mixing",
					},
					DataPoint:              time.Date(2021, 10, 01, 0, 20, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 10, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 19, 59, 999999999, time.UTC),
					UpTime:                 5 * time.Minute,
					FailureTime:            5 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            -2 * time.Minute,
					ErrorBudgetConsumption: 5 * time.Minute,
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.defCfg.ID, func(t *testing.T) {
			var buf bytes.Buffer
			logger.Setup(&buf, "debug")
			defer func() {
				t.Log(buf.String())
				logger.Setup(os.Stderr, "info")
			}()
			err := c.defCfg.Restrict()
			require.NoError(t, err)
			def, err := shimesaba.NewDefinition(c.defCfg)
			require.NoError(t, err)
			actual, err := def.CreateReportsWithAlertsAndPeriod(context.Background(), alerts,
				time.Date(2021, 10, 01, 0, 0, 0, 0, time.UTC),
				time.Date(2021, 10, 01, 0, 20, 0, 0, time.UTC),
			)
			require.NoError(t, err)
			t.Log("actual:")
			for _, a := range actual {
				bs, _ := json.MarshalIndent(a, "", "  ")
				t.Log(string(bs))
			}
			t.Log("expected:")
			for _, e := range c.expected {
				bs, _ := json.MarshalIndent(e, "", "  ")
				t.Log(string(bs))
				if e.Destination.MetricTypeNames == nil {
					e.Destination.MetricTypeNames = make(map[shimesaba.DestinationMetricType]string)
					for _, metricType := range shimesaba.DestinationMetricTypeValues() {
						e.Destination.MetricTypeNames[metricType] = metricType.String()
					}
				}
				if e.Destination.MetricTypeEnabled == nil {
					e.Destination.MetricTypeEnabled = make(map[shimesaba.DestinationMetricType]bool)
					for _, metricType := range shimesaba.DestinationMetricTypeValues() {
						e.Destination.MetricTypeEnabled[metricType] = metricType.DefaultEnabled()
					}
				}

			}
			require.EqualValues(t, c.expected, actual)
		})
	}

}

func TestSLODefinitionStartAt(t *testing.T) {
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
			d, err := shimesaba.NewDefinition(c.cfg)
			require.NoError(t, err)
			actual := d.StartAt(c.now, c.backfill)
			require.EqualValues(t, c.expected, actual)
		})
	}
}
