package shimesaba

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
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
	encoder := json.NewEncoder(fp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(dashboard); err != nil {
		return fmt.Errorf("dashboard encode failed: %w", err)
	}
	log.Printf("[info] dashboard url_path `%s` write to `%s`", dashboard.URLPath, app.dashboardPath)
	return nil
}
