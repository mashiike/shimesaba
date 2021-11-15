package shimesaba

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"text/template"
	"time"

	jsonnet "github.com/google/go-jsonnet"
	gc "github.com/kayac/go-config"
	"github.com/mashiike/shimesaba/internal/timeutils"
)

func (app *App) DashboardInit(ctx context.Context, dashboardIDOrURL string) error {
	if app.dashboardPath == "" {
		return errors.New("dashboard file path is not configured")
	}
	dashboard, err := app.repo.FindDashboard(dashboardIDOrURL)
	if err != nil {
		return err
	}
	fp, err := os.OpenFile(app.dashboardPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("can not open file: %w", err)
	}
	defer fp.Close()
	if err := app.writeDashboard(fp, dashboard); err != nil {
		return err
	}
	log.Printf("[info] dashboard url_path `%s` write to `%s`", dashboard.URLPath, app.dashboardPath)
	return nil
}

func (app *App) writeDashboard(w io.Writer, dashboard *Dashboard) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(dashboard); err != nil {
		return fmt.Errorf("dashboard encode failed: %w", err)
	}
	return nil
}

func (app *App) DashboardBuild(ctx context.Context, optFns ...func(*Options)) error {
	opts := &Options{
		dryRun: false,
	}
	for _, optFn := range optFns {
		optFn(opts)
	}
	dashboard, err := app.loadDashbaord()
	if err != nil {
		return err
	}
	if opts.dryRun {
		var buf bytes.Buffer
		if err := app.writeDashboard(&buf, dashboard); err != nil {
			return err
		}
		log.Printf("[info] build dashboard **dry run** %s", buf.String())
		return nil
	}
	return app.repo.SaveDashboard(ctx, dashboard)
}

func (app *App) loadDashbaord() (*Dashboard, error) {
	if app.dashboardPath == "" {
		return nil, errors.New("dashboard file path is not configured")
	}
	loader := gc.New()
	definitions := make(map[string]interface{}, len(app.definitions))
	for _, def := range app.definitions {
		objectives := make([]string, 0, len(def.objectives))
		for _, obj := range def.objectives {
			objectives = append(objectives, obj.String())
		}
		definitions[def.id] = map[string]interface{}{
			"TimeFrame":               timeutils.DurationString(def.timeFrame),
			"ServiceName":             def.serviceName,
			"CalculateInterval":       timeutils.DurationString(def.calculate),
			"ErrorBudgetSize":         def.errorBudgetSize,
			"ErrorBudgetSizeDuration": timeutils.DurationString(time.Duration(def.errorBudgetSize * float64(def.timeFrame)).Truncate(time.Minute)),
			"Objectives":              objectives,
		}
	}
	data := map[string]interface{}{
		"Metric":      app.metricConfigs,
		"Definitions": definitions,
	}
	loader.Data = data
	funcs := template.FuncMap{
		"file": func(path string) string {
			bs, err := loader.ReadWithEnv(filepath.Join(app.cfgPath, path))
			if err != nil {
				panic(err)
			}
			return string(bs)
		},
		"to_parcentage": func(a float64) float64 {
			return a * 100.0
		},
	}
	loader.Funcs(funcs)
	var dashboard Dashboard
	switch filepath.Ext(app.dashboardPath) {
	case ".jsonnet":
		vm := jsonnet.MakeVM()
		jsonStr, err := vm.EvaluateFile(app.dashboardPath)
		if err != nil {
			return nil, err
		}
		if err := loader.LoadWithEnvJSONBytes(&dashboard, []byte(jsonStr)); err != nil {
			return nil, err
		}
	case ".json":
		if err := loader.LoadWithEnvJSON(&dashboard, app.dashboardPath); err != nil {
			return nil, err
		}
	}
	return &dashboard, nil
}
