package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aws/aws-lambda-go/lambda"
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
	Version = "current"
)

func main() {
	var (
		mackerelAPIKey string
		debug          bool
		dryRun         bool
		configFiles    stringSlice
	)
	flag.Var(&configFiles, "config", "config file path, can set multiple")
	flag.StringVar(&mackerelAPIKey, "mackerel-apikey", "", "for access mackerel API")
	flag.BoolVar(&debug, "debug", false, "output debug log")
	flag.BoolVar(&dryRun, "dry-run", false, "report output stdout and not put mackerel")
	flag.VisitAll(envToFlag)
	flag.Parse()

	minLevel := "info"
	if debug {
		minLevel = "debug"
	}
	logger.Setup(os.Stderr, minLevel)
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
	if err := app.Run(ctx, dryRun); err != nil {
		log.Println("[error]", err)
		os.Exit(1)
	}
}

func lambdaHandler(app *shimesaba.App) func(context.Context) error {
	return func(ctx context.Context) error {
		return app.Run(ctx, false)
	}
}

func envToFlag(f *flag.Flag) {
	name := strings.ToUpper(strings.Replace(f.Name, "-", "_", -1))
	if s, ok := os.LookupEnv(name); ok {
		f.Value.Set(s)
	}
}
