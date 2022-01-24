package shimesaba

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gv "github.com/hashicorp/go-version"
	gc "github.com/kayac/go-config"
	"github.com/mashiike/evaluator"
	"github.com/mashiike/shimesaba/internal/timeutils"
)

//Config for App
type Config struct {
	RequiredVersion string `yaml:"required_version" json:"required_version"`

	Metrics     MetricConfigs     `yaml:"metrics" json:"metrics"`
	Definitions DefinitionConfigs `yaml:"definitions" json:"definitions"`

	//Common definition parameter
	TimeFrame         string      `yaml:"time_frame" json:"time_frame"`
	ServiceName       string      `json:"service_name" yaml:"service_name"`
	MetricPrefix      string      `json:"metric_prefix" yaml:"metric_prefix"`
	ErrorBudgetSize   interface{} `yaml:"error_budget_size" json:"error_budget_size"`
	CalculateInterval string      `yaml:"calculate_interval" json:"calculate_interval"`

	Dashboard          string `json:"dashboard,omitempty" yaml:"dashboard,omitempty"`
	configFilePath     string
	versionConstraints gv.Constraints
}

//MetricConfig handles metric information obtained from Mackerel
type MetricConfig struct {
	ID                  string     `yaml:"id,omitempty" json:"id,omitempty"`
	Type                MetricType `yaml:"type,omitempty" json:"type,omitempty"`
	Name                string     `yaml:"name,omitempty" json:"name,omitempty"`
	ServiceName         string     `yaml:"service_name,omitempty" json:"service_name,omitempty"`
	Roles               []string   `yaml:"roles,omitempty" json:"roles,omitempty"`
	HostName            string     `yaml:"host_name,omitempty" json:"host_name,omitempty"`
	AggregationInterval string     `yaml:"aggregation_interval,omitempty" json:"aggregation_interval,omitempty"`
	AggregationMethod   string     `yaml:"aggregation_method,omitempty" json:"aggregation_method,omitempty"`
	InterpolatedValue   *float64   `yaml:"interpolated_value,omitempty" json:"interpolated_value,omitempty"`
	aggregationInterval time.Duration
}

//String output json
func (c *MetricConfig) String() string {
	bs, _ := json.Marshal(c)
	return string(bs)
}

func coalesceString(strs ...string) string {
	for _, str := range strs {
		if str != "" {
			return str
		}
	}
	return ""
}

//MergeInto merges MetricConfigs together
func (c *MetricConfig) MergeInto(o *MetricConfig) {
	c.ID = coalesceString(o.ID, c.ID)
	c.Name = coalesceString(o.Name, c.Name)
	if o.Type != 0 {
		c.Type = o.Type
	}
	c.ServiceName = coalesceString(o.ServiceName, c.ServiceName)
	c.HostName = coalesceString(o.HostName, c.HostName)
	c.AggregationInterval = coalesceString(o.AggregationInterval, c.AggregationInterval)
	c.AggregationMethod = coalesceString(o.AggregationMethod, c.AggregationMethod)
	roles := make(map[string]struct{}, len(c.Roles))
	for _, role := range c.Roles {
		roles[role] = struct{}{}
	}
	for _, role := range o.Roles {
		roles[role] = struct{}{}
	}
	c.Roles = make([]string, 0, len(roles))
	for role := range roles {
		c.Roles = append(c.Roles, role)
	}
}

// Restrict restricts a configuration.
func (c *MetricConfig) Restrict() error {
	if c.ID == "" {
		return errors.New("id is required")
	}
	if c.ServiceName == "" {
		return errors.New("service_name is required")
	}
	if c.Type == 0 {
		return errors.New("type is required")
	}
	c.AggregationMethod = coalesceString(c.AggregationMethod, "max")

	if c.AggregationInterval == "" {
		c.aggregationInterval = time.Minute
	} else {
		var err error
		c.aggregationInterval, err = timeutils.ParseDuration(c.AggregationInterval)
		if err != nil {
			return fmt.Errorf("aggregation_interval is invalid format: %w", err)
		}
		if c.aggregationInterval < time.Minute {
			return fmt.Errorf("aggregation_interval must over or equal 1m")
		}
	}
	return nil
}

// DurationAggregation converts CalculateInterval as time.Duration
func (c *MetricConfig) DurationAggregation() time.Duration {
	if c.aggregationInterval == 0 {
		var err error
		c.aggregationInterval, err = timeutils.ParseDuration(c.AggregationInterval)
		if err != nil {
			panic(err)
		}
	}
	return c.aggregationInterval
}

//MetricConfigs is a collection of MetricConfig
type MetricConfigs map[string]*MetricConfig

// Restrict restricts a metric configuration.
func (c MetricConfigs) Restrict() error {
	for id, cfg := range c {
		if id != cfg.ID {
			return fmt.Errorf("metrics id=%s not match config id", id)
		}
		if err := cfg.Restrict(); err != nil {
			return fmt.Errorf("metrics[%s] %w", id, err)
		}
	}
	return nil
}

//ToSlice converts the collection to Slice
func (c MetricConfigs) ToSlice() []*MetricConfig {
	ret := make([]*MetricConfig, 0, len(c))
	for _, cfg := range c {
		ret = append(ret, cfg)
	}
	return ret
}

// MarshalYAML controls Yamlization
func (c MetricConfigs) MarshalYAML() (interface{}, error) {
	return c.ToSlice(), nil
}

// String implements fmt.Stringer
func (c MetricConfigs) String() string {
	return fmt.Sprintf("%v", c.ToSlice())
}

// UnmarshalYAML merges duplicate ID MetricConfig
func (c *MetricConfigs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	tmp := make([]*MetricConfig, 0, len(*c))
	if err := unmarshal(&tmp); err != nil {
		return err
	}
	if *c == nil {
		*c = make(MetricConfigs, len(tmp))
	}
	for _, cfg := range tmp {

		if alreadyExist, ok := (*c)[cfg.ID]; ok {
			alreadyExist.MergeInto(cfg)
		} else {
			(*c)[cfg.ID] = cfg
		}
	}
	return nil
}

// DefinitionConfig is a setting related to SLI/SLO
type DefinitionConfig struct {
	ID                string             `json:"id" yaml:"id"`
	TimeFrame         string             `yaml:"time_frame" json:"time_frame"`
	ServiceName       string             `json:"service_name" yaml:"service_name"`
	MetricPrefix      string             `json:"metric_prefix" yaml:"metric_prefix"`
	MetricSuffix      string             `json:"metric_suffix" yaml:"metric_suffix"`
	ErrorBudgetSize   interface{}        `yaml:"error_budget_size" json:"error_budget_size"`
	CalculateInterval string             `yaml:"calculate_interval" json:"calculate_interval"`
	Objectives        []*ObjectiveConfig `json:"objectives" yaml:"objectives"`

	calculateInterval         time.Duration
	timeFrame                 time.Duration
	errorBudgetSizeParcentage float64
}

// MergeInto merges DefinitionConfig together
func (c *DefinitionConfig) MergeInto(o *DefinitionConfig) {
	c.ID = coalesceString(o.ID, c.ID)
	c.TimeFrame = coalesceString(o.TimeFrame, c.TimeFrame)
	c.CalculateInterval = coalesceString(o.CalculateInterval, c.CalculateInterval)
	c.MetricPrefix = coalesceString(o.MetricPrefix, c.MetricPrefix)
	c.MetricSuffix = coalesceString(o.MetricSuffix, c.MetricSuffix)
	if o.ErrorBudgetSize != nil {
		c.ErrorBudgetSize = o.ErrorBudgetSize
	}
	c.ServiceName = coalesceString(o.ServiceName, c.ServiceName)
	c.Objectives = append(c.Objectives, o.Objectives...)
}

const (
	defaultMetricPrefix = "shimesaba"
)

// Restrict restricts a definition configuration.
func (c *DefinitionConfig) Restrict() error {
	if c.ID == "" {
		return errors.New("id is required")
	}
	if c.ServiceName == "" {
		return errors.New("service_name is required")
	}
	if c.MetricPrefix == "" {
		log.Printf("[debug] metric_prefix is empty, fallback %s", defaultMetricPrefix)
		c.MetricPrefix = defaultMetricPrefix
	}
	if c.MetricSuffix == "" {
		log.Printf("[debug] metric_suffix is empty, fallback %s", c.ID)
		c.MetricSuffix = c.ID
	}
	for i, objective := range c.Objectives {
		if err := objective.Restrict(); err != nil {
			return fmt.Errorf("objective[%d] %w", i, err)
		}
	}

	if c.TimeFrame == "" {
		return errors.New("time_frame is required")
	}
	var err error
	c.timeFrame, err = timeutils.ParseDuration(c.TimeFrame)
	if err != nil {
		return fmt.Errorf("time_frame is invalid format: %w", err)
	}
	if c.timeFrame < time.Minute {
		return fmt.Errorf("time_frame must over or equal 1m")
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

	if errorBudgetSizeParcentage, ok := c.ErrorBudgetSize.(float64); ok {
		log.Printf("[warn] make sure to set it in m with units. example %f%%", errorBudgetSizeParcentage*100.0)
		c.errorBudgetSizeParcentage = errorBudgetSizeParcentage
	}
	if errorBudgetSizeString, ok := c.ErrorBudgetSize.(string); ok {
		if strings.ContainsRune(errorBudgetSizeString, '%') {
			value, err := strconv.ParseFloat(strings.TrimRight(errorBudgetSizeString, `%`), 64)
			if err != nil {
				return fmt.Errorf("error_budget can not parse as percentage: %w", err)
			}
			c.errorBudgetSizeParcentage = value / 100.0
		} else {
			errorBudgetSizeDuration, err := timeutils.ParseDuration(errorBudgetSizeString)
			if err != nil {
				return fmt.Errorf("error_budget can not parse as duration: %w", err)
			}
			if errorBudgetSizeDuration >= c.timeFrame || errorBudgetSizeDuration == 0 {
				return fmt.Errorf("error_budget must between %s and 0m", c.timeFrame)
			}
			c.errorBudgetSizeParcentage = float64(errorBudgetSizeDuration) / float64(c.timeFrame)
		}
	}
	if c.errorBudgetSizeParcentage >= 1.0 || c.errorBudgetSizeParcentage <= 0.0 {
		return errors.New("error_budget must between 1.0 and 0.0")
	}

	return nil
}

// DurationTimeFrame converts TimeFrame as time.Duration
func (c *DefinitionConfig) DurationTimeFrame() time.Duration {
	return c.timeFrame
}

// DurationCalculate converts CalculateInterval as time.Duration
func (c *DefinitionConfig) DurationCalculate() time.Duration {
	return c.calculateInterval
}

func (c *DefinitionConfig) ErrorBudgetSizeParcentage() float64 {
	return c.errorBudgetSizeParcentage
}

func (c *DefinitionConfig) StartAt(now time.Time, backfill int) time.Time {
	return now.Truncate(c.calculateInterval).Add(-(time.Duration(backfill) * c.calculateInterval) - c.timeFrame)
}

// Objective Config is a SLO setting
type ObjectiveConfig struct {
	Expr                 string                `yaml:"expr" json:"expr"`
	Alert                *AlertObjectiveConfig `yaml:"alert" json:"alert"`
	AlertObjectiveConfig `yaml:",inline"`

	comparator evaluator.Comparator
}

// Restrict restricts a configuration.
func (c *ObjectiveConfig) Restrict() error {
	globalErr := c.AlertObjectiveConfig.Restrict()
	if globalErr == nil {
		c.Alert = &c.AlertObjectiveConfig
		return nil
	}
	log.Println("[warn] this objective config is deprecated and will not be available starting from v0.8.0.")
	if c.Expr == "" && c.Alert == nil {
		return errors.New("alert config is required")
	}
	if c.Expr != "" && c.Alert != nil {
		return errors.New("only one of expr or alert can be set")
	}
	if c.Expr != "" {
		log.Println("[warn] the Objective feature with `expr` will not be available after v0.8.0, please set up an equivalent monitor on Mackerel to use the Objective feature with alerts.")
		return c.buildComparator()
	}
	if c.Alert != nil {
		log.Println("[warn] Since v0.8.0, the `expr` objective feature will be removed, so the key `alert` will no longer be needed.")
		return c.Alert.Restrict()
	}
	return globalErr
}

func (c *ObjectiveConfig) buildComparator() error {
	e, err := evaluator.New(c.Expr)
	if err != nil {
		return fmt.Errorf("build expr failed: %w", err)
	}
	var ok bool
	c.comparator, ok = e.AsComparator()
	if !ok {
		return errors.New("expr is not comparative")
	}
	return nil
}

// GetComparator returns a Comparator generated from ObjectiveConfig
func (c *ObjectiveConfig) GetComparator() evaluator.Comparator {
	if c.comparator == nil {
		if err := c.buildComparator(); err != nil {
			panic(err)
		}
	}
	return c.comparator
}

//Type returns objective type string
func (c *ObjectiveConfig) Type() string {
	if c.Expr != "" {
		return "expr"
	}
	return "alert"
}

type AlertObjectiveConfig struct {
	MonitorID         string `json:"monitor_id,omitempty" yaml:"monitor_id,omitempty"`
	MonitorNamePrefix string `json:"monitor_name_prefix,omitempty" yaml:"monitor_name_prefix,omitempty"`
	MonitorNameSuffix string `json:"monitor_name_suffix,omitempty" yaml:"monitor_name_suffix,omitempty"`
	MonitorType       string `json:"monitor_type,omitempty" yaml:"monitor_type,omitempty"`
}

// Restrict restricts a configuration.
func (c *AlertObjectiveConfig) Restrict() error {
	if c.MonitorID != "" {
		return nil
	}
	if c.MonitorNamePrefix != "" {
		return nil
	}
	if c.MonitorNameSuffix != "" {
		return nil
	}

	return errors.New("either monitor_id, monitor_name_prefix, monitor_name_suffix or monitor_type is required")
}

// DefinitionConfigs is a collection of DefinitionConfigs that corrects the uniqueness of IDs.
type DefinitionConfigs map[string]*DefinitionConfig

// Restrict restricts a definition configuration.
func (c DefinitionConfigs) Restrict() error {
	for id, cfg := range c {
		if id != cfg.ID {
			return fmt.Errorf("definitionConfigs id=%s not match config id", id)
		}
		if err := cfg.Restrict(); err != nil {
			return fmt.Errorf("definitions[%s].%w", id, err)
		}
	}
	return nil
}

func (c DefinitionConfigs) StartAt(now time.Time, backfill int) time.Time {
	startAt := now
	for _, cfg := range c {
		tmp := cfg.StartAt(now, backfill)
		if tmp.Before(startAt) {
			startAt = tmp
		}
	}
	return startAt
}

func (c DefinitionConfigs) ToSlice() []*DefinitionConfig {
	ret := make([]*DefinitionConfig, 0, len(c))
	for _, cfg := range c {
		ret = append(ret, cfg)
	}
	return ret
}

// MarshalYAML implements yaml.Marshaller
func (c DefinitionConfigs) MarshalYAML() (interface{}, error) {
	return c.ToSlice(), nil
}

// String implements fmt.Stringer
func (c DefinitionConfigs) String() string {
	return fmt.Sprintf("%v", c.ToSlice())
}

// MarshalYAML implements yaml.Unmarshaler
func (c *DefinitionConfigs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	tmp := make([]*DefinitionConfig, 0, len(*c))
	if err := unmarshal(&tmp); err != nil {
		return err
	}
	if *c == nil {
		*c = make(DefinitionConfigs, len(tmp))
	}
	for _, cfg := range tmp {

		if alreadyExist, ok := (*c)[cfg.ID]; ok {
			alreadyExist.MergeInto(cfg)
		} else {
			(*c)[cfg.ID] = cfg
		}
	}
	return nil
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
	if len(c.Metrics) > 0 {
		log.Println("[warn] Since v0.8.0, the setting `metrics` is no longer necessary because the expr function will be removed. This setting is deprecated.")
		if err := c.Metrics.Restrict(); err != nil {
			return fmt.Errorf("metrics has invalid: %w", err)
		}
	}
	for id, cfg := range c.Definitions {
		base := &DefinitionConfig{
			TimeFrame:         c.TimeFrame,
			ServiceName:       c.ServiceName,
			MetricPrefix:      c.MetricPrefix,
			ErrorBudgetSize:   c.ErrorBudgetSize,
			CalculateInterval: c.CalculateInterval,
		}
		base.MergeInto(cfg)
		c.Definitions[id] = base
	}
	if err := c.Definitions.Restrict(); err != nil {
		return fmt.Errorf("definitions has invalid: %w", err)
	}

	return nil
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

// NewDefaultConfig creates a default configuration.
func NewDefaultConfig() *Config {
	return &Config{}
}
