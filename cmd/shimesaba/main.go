package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/handlename/ssmwrap"
	"github.com/mashiike/shimesaba"
	"github.com/mashiike/shimesaba/internal/logger"
	"github.com/urfave/cli/v2"
)

var (
	Version           = "current"
	ssmwrapErr        error
	globalDryRun      bool
	globalDumpReports bool
	globalBackfill    int
)

func main() {
	ssmwrapPaths := os.Getenv("SSMWRAP_PATHS")
	paths := strings.Split(ssmwrapPaths, ",")
	if ssmwrapPaths != "" && len(paths) > 0 {
		ssmwrapErr = ssmwrap.Export(ssmwrap.ExportOptions{
			Paths:   paths,
			Retries: 3,
		})
	}
	cliApp := &cli.App{
		Name:      "shimesaba",
		Usage:     "A commandline tool for tracking SLO/ErrorBudget using Mackerel as an SLI measurement service.",
		UsageText: "shimesaba -config <config file> [command options]",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "config file path, can set multiple",
				EnvVars: []string{"CONFIG", "SHIMESABA_CONFIG"},
			},
			&cli.StringFlag{
				Name:        "mackerel-apikey",
				Aliases:     []string{"k"},
				Usage:       "for access mackerel API",
				DefaultText: "*********",
				EnvVars:     []string{"MACKEREL_APIKEY", "SHIMESABA_MACKEREL_APIKEY"},
			},
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "output debug log",
				EnvVars: []string{"SHIMESABA_DEBUG"},
			},
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "report output stdout and not put mackerel",
				EnvVars:     []string{"SHIMESABA_DRY_RUN"},
				Destination: &globalDryRun,
			},
			&cli.BoolFlag{
				Name:        "dump-reports",
				Usage:       "dump error budget report",
				EnvVars:     []string{"SHIMESABA_DUMP_REPORTS"},
				Destination: &globalDumpReports,
			},
			&cli.IntFlag{
				Name:        "backfill",
				DefaultText: "3",
				Value:       3,
				Usage:       "generate report before n point",
				EnvVars:     []string{"BACKFILL", "SHIMESABA_BACKFILL"},
				Destination: &globalBackfill,
			},
		},
		Action: run,
		Commands: []*cli.Command{
			{
				Name:      "run",
				Usage:     "run shimesaba. this is main feature, use no subcommand",
				UsageText: "shimesaba -config <config file> run [command options]",
				Action: func(c *cli.Context) error {
					return run(c)
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "report output stdout and not put mackerel",
					},
					&cli.BoolFlag{
						Name:  "dump-reports",
						Usage: "dump error budget report",
					},
					&cli.IntFlag{
						Name:  "backfill",
						Usage: "generate report before n point",
					},
				},
			},
		},
	}
	sort.Sort(cli.FlagsByName(cliApp.Flags))
	sort.Sort(cli.CommandsByName(cliApp.Commands))
	cliApp.Version = Version
	cliApp.EnableBashCompletion = true
	cliApp.Before = func(c *cli.Context) error {
		minLevel := "info"
		if c.Bool("debug") {
			minLevel = "debug"
		}
		logger.Setup(os.Stderr, minLevel)
		return nil
	}

	if isLambda() {
		if len(os.Args) <= 1 {
			os.Args = append(os.Args, "run")
		}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer cancel()
	if err := cliApp.RunContext(ctx, os.Args); err != nil {
		log.Printf("[error] %s", err)
	}
}

func isLambda() bool {
	return strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") ||
		os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""
}

func buildApp(c *cli.Context) (*shimesaba.App, error) {
	if ssmwrapErr != nil {
		return nil, fmt.Errorf("ssmwrap.Export failed: %w", ssmwrapErr)
	}
	cfg := shimesaba.NewDefaultConfig()
	if err := cfg.Load(c.StringSlice("config")...); err != nil {
		return nil, err
	}
	if err := cfg.ValidateVersion(Version); err != nil {
		return nil, err
	}
	return shimesaba.New(c.String("mackerel-apikey"), cfg)
}

func run(c *cli.Context) error {
	app, err := buildApp(c)
	if err != nil {
		return err
	}
	backfill := globalBackfill
	if c.Int("backfill") > 0 {
		backfill = c.Int("backfill")
	}
	optFns := []func(*shimesaba.Options){
		shimesaba.DryRunOption(c.Bool("dry-run") || globalDryRun),
		shimesaba.DumpReportsOption(c.Bool("dump-reports") || globalDumpReports),
		shimesaba.BackfillOption(backfill),
	}
	handler := func(ctx context.Context) error {
		return app.Run(ctx, optFns...)
	}
	if isLambda() {
		lambda.Start(handler)
		return nil
	}
	return handler(c.Context)
}
