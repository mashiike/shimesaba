package shimesaba

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Songmu/flextime"
	mackerel "github.com/mackerelio/mackerel-client-go"
)

//App manages life cycle
type App struct {
	repo Repository

	metricConfigs MetricConfigs
	definitions   []*Definition

	maxTimeFrame time.Duration
	maxCalculate time.Duration
}

//New creates an app
func New(apikey string, cfg *Config) (*App, error) {
	client := mackerel.NewClient(apikey)
	return NewWithMackerelClient(client, cfg)
}

//NewWithMackerelClient is there to accept mock clients.
func NewWithMackerelClient(client MackerelClient, cfg *Config) (*App, error) {

	definitions := make([]*Definition, 0, len(cfg.Definitions))
	var maxTimeFrame, maxCalculate time.Duration
	for _, dcfg := range cfg.Definitions {
		if dcfg.DurationCalculate() > maxCalculate {
			maxCalculate = dcfg.DurationCalculate()
		}
		if dcfg.DurationTimeFrame() > maxTimeFrame {
			maxTimeFrame = dcfg.DurationTimeFrame()
		}

		d, err := NewDefinition(dcfg)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, d)
	}
	app := &App{
		repo:          *NewRepository(client),
		metricConfigs: cfg.Metrics,
		definitions:   definitions,
		maxTimeFrame:  maxTimeFrame,
		maxCalculate:  maxCalculate,
	}
	return app, nil
}

type runConfig struct {
	dryRun   bool
	backfill int
}

//RunOption is an App.Run option
type RunOption interface {
	apply(*runConfig)
}

type runOptionFunc func(*runConfig)

func (f runOptionFunc) apply(rc *runConfig) {
	f(rc)
}

//DryRunOption is an option to output the calculated error budget as standard without posting it to Mackerel.
func DryRunOption(dryRun bool) RunOption {
	return runOptionFunc(func(rc *runConfig) {
		rc.dryRun = dryRun
	})
}

//BackfillOption specifies how many points of data to calculate retroactively from the current time.
func BackfillOption(count int) RunOption {
	return runOptionFunc(func(rc *runConfig) {
		rc.backfill = count
	})
}

//Run performs the calculation of the error bar calculation
func (app *App) Run(ctx context.Context, opts ...RunOption) error {
	log.Printf("[info] start run")
	rc := &runConfig{
		backfill: 3,
		dryRun:   false,
	}
	for _, opt := range opts {
		opt.apply(rc)
	}
	log.Println("[debug]", app.metricConfigs)
	now := flextime.Now()
	startAt := now.Truncate(app.maxCalculate).
		Add(-(time.Duration(rc.backfill))*app.maxCalculate - app.maxTimeFrame).
		Truncate(app.maxCalculate)
	log.Printf("[info] fetch metric range %s ~ %s", startAt, now)
	metrics, err := app.repo.FetchMetrics(ctx, app.metricConfigs, startAt, now)
	if err != nil {
		return err
	}
	log.Println("[info] fetched metrics", metrics)
	for _, d := range app.definitions {
		log.Printf("[info] check objectives[%s]\n", d.ID())
		reports, err := d.CreateRepoorts(ctx, metrics)
		if err != nil {
			return fmt.Errorf("objective[%s] create report failed: %w", d.ID(), err)
		}
		if rc.dryRun {
			log.Printf("[info] dryrun! output stdout reports[%s]\n", d.ID())
			bs, err := json.MarshalIndent(reports, "", "  ")
			if err != nil {
				return fmt.Errorf("objective[%s] marshal reports failed: %w", d.ID(), err)
			}
			fmt.Println(string(bs))
		} else {
			log.Printf("[info] save reports[%s]\n", d.ID())
			if err := app.repo.SaveReports(ctx, reports); err != nil {
				return fmt.Errorf("objective[%s] save report failed: %w", d.ID(), err)
			}
		}
	}
	runTime := flextime.Now().Sub(now)
	log.Printf("[info] run successed. run time:%s\n", runTime)
	return nil
}
