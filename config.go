package shimesaba

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	gv "github.com/hashicorp/go-version"
	gc "github.com/kayac/go-config"
	"github.com/mashiike/shimesaba/internal/timeutils"
)

//Config for App
type Config struct {
	RequiredVersion string `yaml:"required_version" json:"required_version"`

	Metrics     MetricConfigs     `yaml:"metrics" json:"metrics"`
	Definitions DefinitionConfigs `yaml:"definitions" json:"definitions"`

	versionConstraints gv.Constraints
}

//MetricConfig handles metric information obtained from Mackerel
type MetricConfig struct {
	ID                  string     `yaml:"id" json:"id"`
	Type                MetricType `yaml:"type" json:"type"`
	Name                string     `yaml:"name" json:"name"`
	ServiceName         string     `yaml:"service_name" json:"service_name"`
	Roles               []string   `yaml:"roles" json:"roles"`
	HostName            string     `yaml:"host_name" json:"host_name"`
	AggregationInterval string     `yaml:"aggregation_interval" json:"aggregation_interval"`
	AggregationMethod   string     `json:"aggregation_method" yaml:"aggregation_method"`

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
		return errors.New("aggregation_interval is required")
	}
	var err error
	c.aggregationInterval, err = timeutils.ParseDuration(c.AggregationInterval)
	if err != nil {
		return fmt.Errorf("aggregation_interval is invalid format: %w", err)
	}
	if c.aggregationInterval < time.Minute {
		return fmt.Errorf("aggregation_interval must over or equal 1m")
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

		if aleadyExist, ok := (*c)[cfg.ID]; ok {
			aleadyExist.MergeInto(cfg)
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
	ErrorBudgetSize   float64            `yaml:"error_budget_size" json:"error_budget_size"`
	CalculateInterval string             `yaml:"calculate_interval" json:"calculate_interval"`
	Objectives        []*ObjectiveConfig `json:"objectives" yaml:"objectives"`

	calculateInterval time.Duration
	timeFrame         time.Duration
}

// MergeInto merges DefinitionConfig together
func (c *DefinitionConfig) MergeInto(o *DefinitionConfig) {
	c.ID = coalesceString(o.ID, c.ID)
	c.TimeFrame = coalesceString(o.TimeFrame, c.TimeFrame)
	c.CalculateInterval = coalesceString(o.CalculateInterval, c.CalculateInterval)
	if o.ErrorBudgetSize != 0.0 {
		c.ErrorBudgetSize = o.ErrorBudgetSize
	}
	c.ServiceName = coalesceString(o.ServiceName, c.ServiceName)
	c.Objectives = append(c.Objectives, o.Objectives...)
}

// Restrict restricts a definition configuration.
func (c *DefinitionConfig) Restrict() error {
	if c.ID == "" {
		return errors.New("id is required")
	}
	if c.ServiceName == "" {
		return errors.New("service_name is required")
	}
	if c.ErrorBudgetSize >= 1.0 || c.ErrorBudgetSize <= 0.0 {
		return errors.New("time_frame must between 1.0 and 0.0")
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
	return nil
}

// DurationTimeFrame converts TimeFrame as time.Duration
func (c *DefinitionConfig) DurationTimeFrame() time.Duration {
	if c.timeFrame == 0 {
		var err error
		c.timeFrame, err = timeutils.ParseDuration(c.TimeFrame)
		if err != nil {
			panic(err)
		}
	}
	return c.timeFrame
}

// DurationCalculate converts CalculateInterval as time.Duration
func (c *DefinitionConfig) DurationCalculate() time.Duration {
	if c.calculateInterval == 0 {
		var err error
		c.calculateInterval, err = timeutils.ParseDuration(c.CalculateInterval)
		if err != nil {
			panic(err)
		}
	}
	return c.calculateInterval
}

// Objective Config is a SLO setting
type ObjectiveConfig struct {
	Expr string `yaml:"expr" json:"expr"`

	metricComparator *MetricComparator
}

// Restrict restricts a configuration.
func (c *ObjectiveConfig) Restrict() error {
	if c.Expr == "" {
		return errors.New("exer is required")
	}
	m, err := NewMetricComparator(c.Expr)
	if err != nil {
		return fmt.Errorf("build expr failed: %w", err)
	}
	c.metricComparator = m
	return nil
}

// GetMetricComparator returns a MetricComparator generated from ObjectiveConfig
func (c *ObjectiveConfig) GetMetricComparator() *MetricComparator {
	return c.metricComparator
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

func (c DefinitionConfigs) ToSlice() []*DefinitionConfig {
	ret := make([]*DefinitionConfig, 0, len(c))
	for _, cfg := range c {
		ret = append(ret, cfg)
	}
	return ret
}

// MarshalYAML implements yaml.Marhaller
func (c DefinitionConfigs) MarshalYAML() (interface{}, error) {
	return c.ToSlice(), nil
}

// String implements fmt.Stringer
func (c DefinitionConfigs) String() string {
	return fmt.Sprintf("%v", c.ToSlice())
}

// MarshalYAML implements yaml.Unmarhaller
func (c *DefinitionConfigs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	tmp := make([]*DefinitionConfig, 0, len(*c))
	if err := unmarshal(&tmp); err != nil {
		return err
	}
	if *c == nil {
		*c = make(DefinitionConfigs, len(tmp))
	}
	for _, cfg := range tmp {

		if aleadyExist, ok := (*c)[cfg.ID]; ok {
			aleadyExist.MergeInto(cfg)
		} else {
			(*c)[cfg.ID] = cfg
		}
	}
	return nil
}

// Load loads configuration file from file paths.
func (c *Config) Load(paths ...string) error {
	if err := gc.LoadWithEnv(c, paths...); err != nil {
		return err
	}
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
	if err := c.Metrics.Restrict(); err != nil {
		return fmt.Errorf("metrics has invalid: %w", err)
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
