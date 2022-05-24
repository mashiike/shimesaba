package shimesaba

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gv "github.com/hashicorp/go-version"
	gc "github.com/kayac/go-config"
	"github.com/mashiike/shimesaba/internal/timeutils"
)

//Config for App
type Config struct {
	RequiredVersion string `yaml:"required_version" json:"required_version"`

	SLOConfig `yaml:"-,inline" json:"-,inline"`
	SLO       []*SLOConfig `yaml:"slo" json:"slo"`

	configFilePath     string
	versionConstraints gv.Constraints
}

// SLOConfig is a setting related to SLI/SLO
type SLOConfig struct {
	ID                string                 `json:"id" yaml:"id"`
	RollingPeriod     string                 `yaml:"rolling_period" json:"rolling_period"`
	Destination       *DestinationConfig     `yaml:"destination" json:"destination"`
	ErrorBudgetSize   interface{}            `yaml:"error_budget_size" json:"error_budget_size"`
	AlertBasedSLI     []*AlertBasedSLIConfig `json:"alert_based_sli" yaml:"alert_based_sli"`
	CalculateInterval string                 `yaml:"calculate_interval" json:"calculate_interval"`

	rollingPeriod             time.Duration
	errorBudgetSizePercentage float64
	calculateInterval         time.Duration
}

// DestinationConfig is a configuration for submitting service metrics to Mackerel
type DestinationConfig struct {
	ServiceName  string                              `json:"service_name" yaml:"service_name"`
	MetricPrefix string                              `json:"metric_prefix" yaml:"metric_prefix"`
	MetricSuffix string                              `json:"metric_suffix" yaml:"metric_suffix"`
	Metrics      map[string]*DestinationMetricConfig `json:"metrics" yaml:"metrics"`
}

type DestinationMetricConfig struct {
	MetricTypeName string
	Enabled        *bool
}

type AlertBasedSLIConfig struct {
	MonitorID         string `json:"monitor_id,omitempty" yaml:"monitor_id,omitempty"`
	MonitorName       string `json:"monitor_name,omitempty" yaml:"monitor_name,omitempty"`
	MonitorNamePrefix string `json:"monitor_name_prefix,omitempty" yaml:"monitor_name_prefix,omitempty"`
	MonitorNameSuffix string `json:"monitor_name_suffix,omitempty" yaml:"monitor_name_suffix,omitempty"`
	MonitorType       string `json:"monitor_type,omitempty" yaml:"monitor_type,omitempty"`
	TryReassessment   bool   `json:"try_reassessment,omitempty" yaml:"try_reassessment,omitempty"`
}

const (
	defaultMetricPrefix = "shimesaba"
)

// NewDefaultConfig creates a default configuration.
func NewDefaultConfig() *Config {
	return &Config{
		SLOConfig: SLOConfig{
			RollingPeriod: "28d",
			Destination: &DestinationConfig{
				MetricPrefix: defaultMetricPrefix,
			},
			CalculateInterval: "1h",
		},
	}
}

// Load loads configuration file from file paths.
func (c *Config) Load(paths ...string) error {
	if len(paths) == 0 {
		return errors.New("no config")
	}
	if err := gc.LoadWithEnv(c, paths...); err != nil {
		return err
	}
	c.configFilePath = filepath.Dir(paths[len(paths)-1])
	return c.Restrict()
}

// Restrict restricts a configuration.
func (c *Config) Restrict() error {
	if c.RequiredVersion != "" {
		constraints, err := gv.NewConstraint(c.RequiredVersion)
		if err != nil {
			return fmt.Errorf("required_version has invalid format: %w", err)
		}
		c.versionConstraints = constraints
	}
	if len(c.SLO) == 0 {
		return errors.New("slo definition not found")
	}

	sloIDs := make(map[string]struct{}, len(c.SLO))

	for i, cfg := range c.SLO {
		mergedCfg := c.SLOConfig.Merge(cfg)
		if _, ok := sloIDs[mergedCfg.ID]; ok {
			return fmt.Errorf("slo id=%s is duplicated", mergedCfg.ID)
		}
		c.SLO[i] = mergedCfg
		if err := mergedCfg.Restrict(); err != nil {
			return fmt.Errorf("slo[%s] is invalid: %w", mergedCfg.ID, err)
		}
	}

	return nil
}

// Restrict restricts a definition configuration.
func (c *SLOConfig) Restrict() error {
	if c.ID == "" {
		return errors.New("id is required")
	}

	if c.RollingPeriod == "" {
		return errors.New("rolling_period is required")
	}
	var err error
	c.rollingPeriod, err = timeutils.ParseDuration(c.RollingPeriod)
	if err != nil {
		return fmt.Errorf("rolling_period is invalid format: %w", err)
	}
	if c.rollingPeriod < time.Minute {
		return fmt.Errorf("rolling_period must over or equal 1m")
	}

	if c.Destination == nil {
		return errors.New("destination is not configured")
	}
	if err := c.Destination.Restrict(c.ID); err != nil {
		return fmt.Errorf("destination %w", err)
	}

	if errorBudgetSizePercentage, ok := c.ErrorBudgetSize.(float64); ok {
		log.Printf("[warn] make sure to set it in m with units. example %f%%", errorBudgetSizePercentage*100.0)
		c.errorBudgetSizePercentage = errorBudgetSizePercentage
	}
	if errorBudgetSizeString, ok := c.ErrorBudgetSize.(string); ok {
		if strings.ContainsRune(errorBudgetSizeString, '%') {
			value, err := strconv.ParseFloat(strings.TrimRight(errorBudgetSizeString, `%`), 64)
			if err != nil {
				return fmt.Errorf("error_budget can not parse as percentage: %w", err)
			}
			c.errorBudgetSizePercentage = value / 100.0
		} else {
			errorBudgetSizeDuration, err := timeutils.ParseDuration(errorBudgetSizeString)
			if err != nil {
				return fmt.Errorf("error_budget can not parse as duration: %w", err)
			}
			if errorBudgetSizeDuration >= c.rollingPeriod || errorBudgetSizeDuration == 0 {
				return fmt.Errorf("error_budget must between %s and 0m", c.rollingPeriod)
			}
			c.errorBudgetSizePercentage = float64(errorBudgetSizeDuration) / float64(c.rollingPeriod)
		}
	}
	if c.errorBudgetSizePercentage >= 1.0 || c.errorBudgetSizePercentage <= 0.0 {
		return errors.New("error_budget must between 1.0 and 0.0")
	}

	for i, alertBasedSLI := range c.AlertBasedSLI {
		if err := alertBasedSLI.Restrict(); err != nil {
			return fmt.Errorf("alert_based_sli[%d] %w", i, err)
		}
	}

	if c.CalculateInterval == "" {
		return errors.New("calculate_interval is required")
	}
	c.calculateInterval, err = timeutils.ParseDuration(c.CalculateInterval)
	if err != nil {
		return fmt.Errorf("calculate_interval is invalid format: %w", err)
	}
	if c.calculateInterval < time.Minute {
		return fmt.Errorf("calculate_interval must over or equal 1m")
	}
	if c.calculateInterval >= 24*time.Hour {
		log.Printf("[warn] We do not recommend calculate_interval=`%s` setting. because can not post service metrics older than 24 hours to Mackerel.\n", c.CalculateInterval)
	}

	return nil
}

// Restrict restricts a definition configuration.
func (c *DestinationConfig) Restrict(sloID string) error {
	if c.ServiceName == "" {
		return errors.New("service_name is required")
	}
	if c.MetricPrefix == "" {
		log.Printf("[debug] metric_prefix is empty, fallback %s", defaultMetricPrefix)
		c.MetricPrefix = defaultMetricPrefix
	}
	if c.MetricSuffix == "" {
		log.Printf("[debug] metric_suffix is empty, fallback %s", sloID)
		c.MetricSuffix = sloID
	}
	if c.Metrics == nil {
		c.Metrics = make(map[string]*DestinationMetricConfig)
	}
	keys := DestinationMetricTypeValues()
	for _, key := range keys {
		metricCfg, ok := c.Metrics[key.ID()]
		if !ok {
			metricCfg = &DestinationMetricConfig{}
		}
		if err := metricCfg.Restrict(key); err != nil {
			return fmt.Errorf("metrics `%s`: %w", key.ID(), err)
		}
		c.Metrics[key.ID()] = metricCfg
	}

	return nil
}

// Restrict restricts a definition configuration.
func (c *DestinationMetricConfig) Restrict(t DestinationMetricType) error {
	if c.MetricTypeName == "" {
		c.MetricTypeName = t.DefaultTypeName()
	}
	if c.Enabled == nil {
		enabled := t.DefaultEnabled()
		c.Enabled = &enabled
	}
	return nil
}

// Restrict restricts a configuration.
func (c *AlertBasedSLIConfig) Restrict() error {
	if c.MonitorID != "" {
		return nil
	}
	if c.MonitorName != "" {
		return nil
	}
	if c.MonitorNamePrefix != "" {
		return nil
	}
	if c.MonitorNameSuffix != "" {
		return nil
	}

	return errors.New("either monitor_id, monitor_name, monitor_name_prefix, monitor_name_suffix or monitor_type is required")
}

// Merge merges SLOConfig together
func (c *SLOConfig) Merge(o *SLOConfig) *SLOConfig {
	ret := &SLOConfig{
		ID:                coalesceString(o.ID, c.ID),
		RollingPeriod:     coalesceString(o.RollingPeriod, c.RollingPeriod),
		Destination:       c.Destination.Merge(o.Destination),
		ErrorBudgetSize:   c.ErrorBudgetSize,
		CalculateInterval: coalesceString(o.CalculateInterval, c.CalculateInterval),
	}
	if o.ErrorBudgetSize != nil {
		ret.ErrorBudgetSize = o.ErrorBudgetSize
	}
	ret.AlertBasedSLI = append(ret.AlertBasedSLI, c.AlertBasedSLI...)
	ret.AlertBasedSLI = append(ret.AlertBasedSLI, o.AlertBasedSLI...)

	return ret
}

// Merge merges DestinationConfig together
func (c *DestinationConfig) Merge(o *DestinationConfig) *DestinationConfig {
	if o == nil {
		o = &DestinationConfig{}
	}
	ret := &DestinationConfig{
		ServiceName:  coalesceString(o.ServiceName, c.ServiceName),
		MetricPrefix: coalesceString(o.MetricPrefix, c.MetricPrefix),
		MetricSuffix: coalesceString(o.MetricSuffix, c.MetricSuffix),
	}
	keys := DestinationMetricTypeStrings()
	metrics := make(map[string]*DestinationMetricConfig, len(keys))
	base := c.Metrics
	if base == nil {
		base = make(map[string]*DestinationMetricConfig)
	}
	if o.Metrics != nil {
		for _, key := range keys {
			metricCfg, ok := base[key]
			if !ok {
				metricCfg = &DestinationMetricConfig{}
			}
			metrics[key] = metricCfg.Merge(o.Metrics[key])
		}
	}
	ret.Metrics = metrics
	return ret
}

// Merge merges DestinationMetricConfig together
func (c *DestinationMetricConfig) Merge(o *DestinationMetricConfig) *DestinationMetricConfig {
	if o == nil {
		o = &DestinationMetricConfig{}
	}
	ret := &DestinationMetricConfig{
		MetricTypeName: coalesceString(o.MetricTypeName, c.MetricTypeName),
		Enabled:        coalesce(o.Enabled, c.Enabled),
	}
	return ret
}

// ValidateVersion validates a version satisfies required_version.
func (c *Config) ValidateVersion(version string) error {
	if c.versionConstraints == nil {
		log.Println("[warn] required_version is empty. Skip checking required_version.")
		return nil
	}
	versionParts := strings.SplitN(version, "-", 2)
	v, err := gv.NewVersion(versionParts[0])
	if err != nil {
		log.Printf("[warn]: Invalid version format \"%s\". Skip checking required_version.", version)
		// invalid version string (e.g. "current") always allowed
		return nil
	}
	if !c.versionConstraints.Check(v) {
		return fmt.Errorf("version %s does not satisfy constraints required_version: %s", version, c.versionConstraints)
	}
	return nil
}

// DurationRollingPeriod converts RollingPeriod as time.Duration
func (c *SLOConfig) DurationRollingPeriod() time.Duration {
	return c.rollingPeriod
}

// DurationCalculate converts CalculateInterval as time.Duration
func (c *SLOConfig) DurationCalculate() time.Duration {
	return c.calculateInterval
}

func (c *SLOConfig) ErrorBudgetSizePercentage() float64 {
	return c.errorBudgetSizePercentage
}

func coalesceString(strs ...string) string {
	for _, str := range strs {
		if str != "" {
			return str
		}
	}
	return ""
}

func coalesce[T any](elements ...*T) *T {
	for _, element := range elements {
		if element != nil {
			var ret T
			ret = *element
			return &ret
		}
	}
	return nil
}
