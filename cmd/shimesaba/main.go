package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/handlename/ssmwrap"
	"github.com/mashiike/shimesaba"
	"github.com/mashiike/shimesaba/internal/logger"
)

type stringSlice []string

func (i *stringSlice) String() string {
	return fmt.Sprintf("%v", *i)
}
func (i *stringSlice) Set(v string) error {
	if strings.ContainsRune(v, ',') {
		*i = append(*i, strings.Split(v, ",")...)
	} else {
		*i = append(*i, v)
	}
	return nil
}

var (
	Version        = "current"
	mackerelAPIKey string
	debug          bool
	dryRun         bool
	backfill       uint
	version        bool
	configFiles    stringSlice
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

	flag.Var(&configFiles, "config", "config file path, can set multiple")
	flag.StringVar(&mackerelAPIKey, "mackerel-apikey", "", "for access mackerel API")
	flag.BoolVar(&debug, "debug", false, "output debug log")
	flag.BoolVar(&dryRun, "dry-run", false, "report output stdout and not put mackerel")
	flag.BoolVar(&version, "version", false, "show version")
	flag.UintVar(&backfill, "backfill", 3, "generate report before n point")
	flag.VisitAll(envToFlag)
	flag.Parse()

	minLevel := "info"
	if debug {
		minLevel = "debug"
	}
	logger.Setup(os.Stderr, minLevel)
	if version {
		log.Printf("[info] shimesaba version : %s", Version)
		log.Printf("[info] go runtime version: %s", runtime.Version())
		return
	}
	if ssmwrapErr != nil {
		logger.Setup(os.Stderr, "info")
		log.Printf("[error] ssmwrap.Export failed: %s\n", ssmwrapErr)
		os.Exit(1)
	}
	if backfill == 0 {
		log.Println("[error] backfill count must positive avlue")
		os.Exit(1)
	}
	cfg := shimesaba.NewDefaultConfig()
	if err := cfg.Load(configFiles...); err != nil {
		log.Println("[error]", err)
		os.Exit(1)
	}
	if err := cfg.ValidateVersion(Version); err != nil {
		log.Println("[error]", err)
		os.Exit(1)
	}
	app, err := shimesaba.New(mackerelAPIKey, cfg)
	if err != nil {
		log.Println("[error]", err)
		os.Exit(1)
	}

	if strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") ||
		os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		lambda.Start(lambdaHandler(app))
		return
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer cancel()
	if err := app.Run(ctx, shimesaba.DryRunOption(dryRun), shimesaba.BackfillOption(int(backfill))); err != nil {
		log.Println("[error]", err)
		os.Exit(1)
	}
}

func lambdaHandler(app *shimesaba.App) func(context.Context) error {
	return func(ctx context.Context) error {
		return app.Run(ctx, shimesaba.DryRunOption(dryRun), shimesaba.BackfillOption(int(backfill)))
	}
}

func envToFlag(f *flag.Flag) {
	name := strings.ToUpper(strings.Replace(f.Name, "-", "_", -1))
	if s, ok := os.LookupEnv(name); ok {
		f.Value.Set(s)
	}
}
