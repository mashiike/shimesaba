package shimesaba

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"time"

	"github.com/Songmu/flextime"
	mackerel "github.com/mackerelio/mackerel-client-go"
)

//App manages life cycle
type App struct {
	repo Repository

	metricConfigs MetricConfigs
	definitions   []*Definition

	maxTimeFrame  time.Duration
	maxCalculate  time.Duration
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

	definitions := make([]*Definition, 0, len(cfg.Definitions))
	var maxTimeFrame, maxCalculate time.Duration
	for _, defCfg := range cfg.Definitions {
		if defCfg.DurationCalculate() > maxCalculate {
			maxCalculate = defCfg.DurationCalculate()
		}
		if defCfg.DurationTimeFrame() > maxTimeFrame {
			maxTimeFrame = defCfg.DurationTimeFrame()
		}

		d, err := NewDefinition(defCfg)
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
		cfgPath:       cfg.configFilePath,
		dashboardPath: filepath.Join(cfg.configFilePath, cfg.Dashboard),
	}
	return app, nil
}

//Run performs the calculation of the error bar calculation
func (app *App) Run(ctx context.Context, optFns ...func(*Options)) error {
	log.Printf("[info] start run")
	opts := &Options{
		backfill: 3,
		dryRun:   false,
	}
	for _, optFn := range optFns {
		optFn(opts)
	}
	if opts.backfill <= 0 {
		return errors.New("backfill must over 0")
	}
	log.Println("[debug]", app.metricConfigs)
	now := flextime.Now()
	startAt := now.Truncate(app.maxCalculate).
		Add(-(time.Duration(opts.backfill))*app.maxCalculate - app.maxTimeFrame).
		Truncate(app.maxCalculate)
	log.Printf("[info] fetch metric range %s ~ %s", startAt, now)
	metrics, err := app.repo.FetchMetrics(ctx, app.metricConfigs, startAt, now)
	if err != nil {
		return err
	}
	log.Println("[info] fetched metrics", metrics)
	for _, d := range app.definitions {
		log.Printf("[info] check objectives[%s]\n", d.ID())
		reports, err := d.CreateReports(ctx, metrics)
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
		if opts.dryRun {
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
	log.Printf("[info] run successes. run time:%s\n", runTime)
	return nil
}
