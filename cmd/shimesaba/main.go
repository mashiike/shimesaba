package main

import (
	"context"
	"errors"
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
	Version = "current"
	app     *shimesaba.App
)

func main() {
	paths := strings.Split(os.Getenv("SSMWRAP_PATHS"), ",")
	var ssmwrapErr error
	if len(paths) > 0 {
		ssmwrapErr = ssmwrap.Export(ssmwrap.ExportOptions{
			Paths:   paths,
			Retries: 3,
		})
	}
	cliApp := &cli.App{
		Name:  "shimesaba",
		Usage: "A commandline tool for tracking SLO/ErrorBudget using Mackerel as an SLI measurement service.",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "config",
				Aliases:  []string{"c"},
				Usage:    "config file path, can set multiple",
				Required: true,
				EnvVars:  []string{"CONFIG", "SHIMESABA_CONFIG"},
			},
			&cli.StringFlag{
				Name:        "mackerel-apikey",
				Aliases:     []string{"k"},
				Usage:       "for access mackerel API",
				Required:    true,
				DefaultText: "*********",
				EnvVars:     []string{"MACKEREL_APIKEY", "SHIMESABA_MACKEREL_APIKEY"},
			},
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "output debug log",
				EnvVars: []string{"SHIMESABA_DEBUG"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "run shimesaba. this is main feature",
				Action: func(c *cli.Context) error {
					if c.Int("backfill") <= 0 {
						return errors.New("backfill count must positive value")
					}
					opts := []shimesaba.RunOption{
						shimesaba.DryRunOption(c.Bool("dry-run")),
						shimesaba.BackfillOption(c.Int("backfill")),
					}
					handler := func(ctx context.Context) error {
						return app.Run(ctx, opts...)
					}
					if isLabmda() {
						lambda.Start(handler)
						return nil
					}
					return handler(c.Context)
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "dry-run",
						Usage:   "report output stdout and not put mackerel",
						EnvVars: []string{"SHIMESABA_DRY_RUN"},
					},
					&cli.IntFlag{
						Name:        "backfill",
						DefaultText: "3",
						Value:       3,
						Usage:       "generate report before n point",
						EnvVars:     []string{"BACKFILL", "SHIMESABA_BACKFILL"},
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
		switch c.Args().First() {
		case "help", "h", "version":
			return nil
		default:
		}
		if ssmwrapErr != nil {
			return fmt.Errorf("ssmwrap.Export failed: %w", ssmwrapErr)
		}
		cfg := shimesaba.NewDefaultConfig()
		if err := cfg.Load(c.StringSlice("config")...); err != nil {
			return err
		}
		if err := cfg.ValidateVersion(Version); err != nil {
			return err
		}
		var err error
		app, err = shimesaba.New(c.String("mackerel-apikey"), cfg)
		if err != nil {
			return err
		}
		return nil
	}

	if isLabmda() {
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

func isLabmda() bool {
	return strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") ||
		os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""
}
