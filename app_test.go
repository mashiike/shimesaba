package shimesaba_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Songmu/flextime"
	"github.com/mashiike/shimesaba"
	"github.com/mashiike/shimesaba/internal/logger"
	"github.com/stretchr/testify/require"
)

func TestAppWithMock(t *testing.T) {
	backfillCounts := []int{3, 4, 5}
	for _, backfill := range backfillCounts {
		t.Run(fmt.Sprintf("backfill=%d", backfill), func(t *testing.T) {
			cases := []struct {
				configFile string
				expected   map[string]int
			}{
				{
					configFile: "testdata/app_test.yaml",
					expected: map[string]int{
						"shimesaba.error_budget.alerts":                        backfill,
						"shimesaba.error_budget_consumption.alerts":            backfill,
						"shimesaba.error_budget_consumption_percentage.alerts": backfill,
						"shimesaba.error_budget_percentage.alerts":             backfill,
						"shimesaba.error_budget_remaining_percentage.alerts":   backfill,
					},
				},
				{
					configFile: "testdata/app_disable_test.yaml",
					expected: map[string]int{
						"app_test.eb.availability":  backfill,
						"app_test.ebr.availability": backfill,
					},
				},
				{
					configFile: "testdata/app_uptime_and_failuretime.yaml",
					expected: map[string]int{
						"shimesaba.error_budget.alerts":                        backfill,
						"shimesaba.error_budget_consumption.alerts":            backfill,
						"shimesaba.error_budget_consumption_percentage.alerts": backfill,
						"shimesaba.error_budget_percentage.alerts":             backfill,
						"shimesaba.error_budget_remaining_percentage.alerts":   backfill,
						"shimesaba.failure_time.alerts":                        backfill,
						"shimesaba.uptime.alerts":                              backfill,
					},
				},
			}
			for _, c := range cases {
				t.Run(c.configFile, func(t *testing.T) {
					var buf bytes.Buffer
					logger.Setup(&buf, "debug")
					defer func() {
						t.Log(buf.String())
						logger.Setup(os.Stderr, "info")
					}()
					cfg := shimesaba.NewDefaultConfig()
					err := cfg.Load(c.configFile)
					require.NoError(t, err, "load cfg")
					client := newMockMackerelClient(t)
					app, err := shimesaba.NewWithMackerelClient(client, cfg)
					require.NoError(t, err, "create app")
					restore := flextime.Set(time.Date(2021, 10, 1, 0, 21, 0, 0, time.UTC))
					defer restore()
					err = app.Run(context.Background(), shimesaba.BackfillOption(backfill))
					require.NoError(t, err, "run app")

					actual := make(map[string]int)
					for _, v := range client.posted {
						if _, ok := actual[v.Name]; !ok {
							actual[v.Name] = 0
						}
						actual[v.Name]++
					}
					require.EqualValues(t, c.expected, actual)
				})
			}
		})
	}
}
