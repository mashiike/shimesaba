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
	"strconv"
	"text/template"
	"time"

	jsonnet "github.com/google/go-jsonnet"
	gc "github.com/kayac/go-config"
	"github.com/mashiike/evaluator"
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
	dashboard, err := app.loadDashboard()
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

func (app *App) loadDashboard() (*Dashboard, error) {
	if app.dashboardPath == "" {
		return nil, errors.New("dashboard file path is not configured")
	}
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
	loader := newLoader(app.cfgPath)
	loader.Data = data
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

func newLoader(pathBase string) *gc.Loader {
	loader := gc.New()
	funcs := template.FuncMap{
		"file": func(path string) string {
			bs, err := loader.ReadWithEnv(filepath.Join(pathBase, path))
			if err != nil {
				panic(err)
			}
			return string(bs)
		},
		"to_percentage": func(a interface{}) interface{} {
			switch num := a.(type) {
			case float32:
				return num * 100.0
			case float64:
				return num * 100.0
			case string:
				value, err := strconv.ParseFloat(num, 64)
				if err != nil {
					panic(err)
				}
				return value * 100.0
			}
			panic("unexpected type")
		},
		"eval_expr": func(expr string, variables ...interface{}) interface{} {
			e, err := evaluator.New(expr)
			if err != nil {
				panic(err)
			}
			nVar := len(variables)
			mapVariables := make(evaluator.Variables, nVar)
			if nVar > 0 {
				if _, ok := variables[0].(string); ok {
					for i := 0; i < len(variables); i += 2 {
						mapVariables[fmt.Sprintf("%s", variables[i])] = variables[i+1]
					}
				} else {
					for i := 0; i < len(variables); i++ {
						mapVariables[fmt.Sprintf("var%d", i+1)] = variables[i]
					}
				}
			}
			ret, err := e.Eval(mapVariables)
			if err != nil {
				panic(err)
			}
			return ret
		},
	}
	loader.Funcs(funcs)
	return loader
}
