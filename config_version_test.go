package shimesaba_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

func TestConfigValidateVersion(t *testing.T) {
	cases := []struct {
		name            string
		requiredVersion string
		version         string
		wantErr         bool
	}{
		{name: "no required_version", requiredVersion: "", version: "v1.4.5"},
		{name: "satisfied", requiredVersion: ">=1.0.0", version: "v1.4.5"},
		{name: "satisfied without v prefix", requiredVersion: ">=1.0.0, <2.0.0", version: "1.4.5"},
		{name: "not satisfied", requiredVersion: ">=2.0.0", version: "v1.4.5", wantErr: true},
		{name: "pre-release suffix is ignored", requiredVersion: ">=1.4.5", version: "v1.4.5-rc1"},
		{name: "invalid version is always allowed", requiredVersion: ">=99.0.0", version: "current"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := loadConfigWithRequiredVersion(t, c.requiredVersion)
			err := cfg.ValidateVersion(c.version)
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigLoadInvalidRequiredVersion(t *testing.T) {
	path := writeConfigWithRequiredVersion(t, "not a constraint")
	cfg := shimesaba.NewDefaultConfig()
	require.Error(t, cfg.Load(path))
}

func writeConfigWithRequiredVersion(t *testing.T, requiredVersion string) string {
	t.Helper()
	body := fmt.Sprintf(`required_version: %q
slo:
  - id: test
    destination:
      service_name: shimesaba
    error_budget_size: 1%%
    alert_based_sli:
      - monitor_id: dummy
`, requiredVersion)
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func loadConfigWithRequiredVersion(t *testing.T, requiredVersion string) *shimesaba.Config {
	t.Helper()
	cfg := shimesaba.NewDefaultConfig()
	require.NoError(t, cfg.Load(writeConfigWithRequiredVersion(t, requiredVersion)))
	return cfg
}
