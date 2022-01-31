package shimesaba

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/Songmu/flextime"
	mackerel "github.com/mackerelio/mackerel-client-go"
)

//App manages life cycle
type App struct {
	repo *Repository

	metricConfigs     MetricConfigs
	definitionConfigs DefinitionConfigs

	cfgPath       string
	dashboardPath string
}

//New creates an app
func New(apikey string, cfg *Config) (*App, error) {
	client := mackerel.NewClient(apikey)
	return NewWithMackerelClient(client, cfg)
}

//NewWithMackerelClient is there to accept mock clients.
func NewWithMackerelClient(client MackerelClient, cfg *Config) (*App, error) {
	app := &App{
		repo:              NewRepository(client),
		metricConfigs:     cfg.Metrics,
		definitionConfigs: cfg.Definitions,
		cfgPath:           cfg.configFilePath,
		dashboardPath:     filepath.Join(cfg.configFilePath, cfg.Dashboard),
	}
	return app, nil
}

//Run performs the calculation of the error bar calculation
func (app *App) Run(ctx context.Context, optFns ...func(*Options)) error {
	orgName, err := app.repo.GetOrgName(ctx)
	if err != nil {
		return err
	}
	log.Printf("[info] start run in the `%s` organization.", orgName)
	opts := &Options{
		backfill: 3,
		dryRun:   false,
	}
	for _, optFn := range optFns {
		optFn(opts)
	}

	repo := app.repo
	if opts.dryRun {
		log.Println("[notice] **with dry run**")
		repo = repo.WithDryRun()
	}

	if opts.backfill <= 0 {
		return errors.New("backfill must over 0")
	}
	log.Printf("[debug] metricConfigs %#v", app.metricConfigs)
	now := flextime.Now()
	startAt := app.definitionConfigs.StartAt(now, opts.backfill)
	log.Printf("[info] fetch metric range %s ~ %s", startAt, now)
	metrics, err := repo.FetchMetrics(ctx, app.metricConfigs, startAt, now)
	if err != nil {
		return err
	}
	log.Println("[info] fetched metrics", metrics)
	log.Printf("[info] fetch alerts range %s ~ %s", startAt, now)
	alerts, err := repo.FetchAlerts(ctx, startAt, now)
	if err != nil {
		return err
	}
	log.Println("[info] fetched alerts", len(alerts))
	for _, defCfg := range app.definitionConfigs {
		d, err := NewDefinition(defCfg)
		if err != nil {
			return err
		}
		log.Printf("[info] check objectives[%s]\n", d.ID())
		reports, err := d.CreateReports(ctx, metrics, alerts,
			defCfg.StartAt(now, opts.backfill),
			now,
		)
		if err != nil {
			return fmt.Errorf("objective[%s] create report failed: %w", d.ID(), err)
		}
		if len(reports) > opts.backfill {
			sort.Slice(reports, func(i, j int) bool {
				return reports[i].DataPoint.Before(reports[j].DataPoint)
			})
			n := len(reports) - opts.backfill
			if n < 0 {
				n = 0
			}
			reports = reports[n:]
		}

		log.Printf("[info] save reports[%s]\n", d.ID())
		if err := repo.SaveReports(ctx, reports); err != nil {
			return fmt.Errorf("objective[%s] save report failed: %w", d.ID(), err)
		}
	}
	runTime := flextime.Now().Sub(now)
	log.Printf("[info] run successes. run time:%s\n", runTime)
	return nil
}
