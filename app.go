package shimesaba

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"

	"github.com/Songmu/flextime"
	mackerel "github.com/mackerelio/mackerel-client-go"
)

//App manages life cycle
type App struct {
	repo           *Repository
	SLODefinitions []*Definition
}

//New creates an app
func New(apikey string, cfg *Config) (*App, error) {
	client := mackerel.NewClient(apikey)
	return NewWithMackerelClient(client, cfg)
}

//NewWithMackerelClient is there to accept mock clients.
func NewWithMackerelClient(client MackerelClient, cfg *Config) (*App, error) {
	slo := make([]*Definition, 0, len(cfg.SLO))
	for _, c := range cfg.SLO {
		d, err := NewDefinition(c)
		if err != nil {
			return nil, err
		}
		slo = append(slo, d)
	}
	app := &App{
		repo:           NewRepository(client),
		SLODefinitions: slo,
	}
	return app, nil
}

type Options struct {
	dryRun   bool
	backfill int
}

//DryRunOption is an option to output the calculated error budget as standard without posting it to Mackerel.
func DryRunOption(dryRun bool) func(*Options) {
	return func(opt *Options) {
		opt.dryRun = dryRun
	}
}

//BackfillOption specifies how many points of data to calculate retroactively from the current time.
func BackfillOption(count int) func(*Options) {
	return func(opt *Options) {
		opt.backfill = count
	}
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
	now := flextime.Now()

	for _, d := range app.SLODefinitions {
		log.Printf("[info] check serice level objectives[%s]\n", d.ID())
		reports, err := d.CreateReports(ctx, repo, now, opts.backfill)
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
