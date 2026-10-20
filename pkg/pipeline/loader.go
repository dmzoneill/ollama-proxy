package pipeline

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the YAML configuration structure
type Config struct {
	Pipelines []YAML `yaml:"pipelines"`
}

// YAML represents a pipeline in YAML format
type YAML struct {
	ID          string      `yaml:"id"`
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Stages      []StageYAML `yaml:"stages"`
	Options     OptionsYAML `yaml:"options"`
}

// StageYAML represents a stage in YAML format
type StageYAML struct {
	InputTransform    map[string]interface{} `yaml:"input_transform"`
	OutputTransform   map[string]interface{} `yaml:"output_transform"`
	ID                string                 `yaml:"id"`
	Type              string                 `yaml:"type"`
	Description       string                 `yaml:"description"`
	PreferredBackend  string                 `yaml:"preferred_backend"`
	PreferredHardware string                 `yaml:"preferred_hardware"`
	Model             string                 `yaml:"model"`
	ForwardingPolicy  ForwardingPolicyYAML   `yaml:"forwarding_policy"`
}

// ForwardingPolicyYAML represents forwarding policy in YAML
type ForwardingPolicyYAML struct {
	EscalationPath        []string `yaml:"escalation_path"`
	MinConfidence         float64  `yaml:"min_confidence"`
	MaxRetries            int      `yaml:"max_retries"`
	MaxTemperature        float64  `yaml:"max_temperature"`
	MaxFanPercent         int      `yaml:"max_fan_percent"`
	QualityThreshold      float64  `yaml:"quality_threshold"`
	EnableConfidenceCheck bool     `yaml:"enable_confidence_check"`
	EnableThermalCheck    bool     `yaml:"enable_thermal_check"`
	EnableQualityCheck    bool     `yaml:"enable_quality_check"`
}

// OptionsYAML represents pipeline options in YAML
type OptionsYAML struct {
	EnableStreaming bool `yaml:"enable_streaming"`
	PreserveContext bool `yaml:"preserve_context"`
	ContinueOnError bool `yaml:"continue_on_error"`
	CollectMetrics  bool `yaml:"collect_metrics"`
	ParallelStages  bool `yaml:"parallel_stages"`
	LatencyCritical bool `yaml:"latency_critical"`
}

// Loader loads pipelines from YAML configuration
type Loader struct {
	pipelines map[string]*Pipeline
}

// NewPipelineLoader creates a new pipeline loader
func NewPipelineLoader() *Loader {
	return &Loader{
		pipelines: make(map[string]*Pipeline),
	}
}

// LoadFromFile loads pipelines from a YAML file
func (pl *Loader) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read pipeline config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse pipeline config: %w", err)
	}

	for _, pipelineYAML := range config.Pipelines {
		pipeline, err := pl.convertYAMLToPipeline(pipelineYAML)
		if err != nil {
			return fmt.Errorf("failed to convert pipeline %s: %w", pipelineYAML.ID, err)
		}

		pl.pipelines[pipeline.ID] = pipeline
	}

	return nil
}

// GetPipeline retrieves a pipeline by ID
func (pl *Loader) GetPipeline(id string) (*Pipeline, error) {
	pipeline, ok := pl.pipelines[id]
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", id)
	}

	return pipeline, nil
}

// AddPipeline adds a pipeline to the loader (primarily for testing)
func (pl *Loader) AddPipeline(pipeline *Pipeline) {
	pl.pipelines[pipeline.ID] = pipeline
}

// ListPipelines returns all loaded pipeline IDs
func (pl *Loader) ListPipelines() []string {
	ids := make([]string, 0, len(pl.pipelines))
	for id := range pl.pipelines {
		ids = append(ids, id)
	}
	return ids
}

// convertYAMLToPipeline converts YAML representation to Pipeline struct
func (pl *Loader) convertYAMLToPipeline(yamlPipeline YAML) (*Pipeline, error) {
	stages := make([]*Stage, 0, len(yamlPipeline.Stages))

	for _, stageYAML := range yamlPipeline.Stages {
		stage, err := pl.convertYAMLToStage(stageYAML)
		if err != nil {
			return nil, fmt.Errorf("failed to convert stage %s: %w", stageYAML.ID, err)
		}
		stages = append(stages, stage)
	}

	pipeline := &Pipeline{
		ID:          yamlPipeline.ID,
		Name:        yamlPipeline.Name,
		Description: yamlPipeline.Description,
		Stages:      stages,
		Options:     pl.convertYAMLToOptions(yamlPipeline.Options),
	}

	return pipeline, nil
}

// convertYAMLToStage converts YAML stage to Stage struct
//
//nolint:unparam // error return kept for future validation logic
func (pl *Loader) convertYAMLToStage(yamlStage StageYAML) (*Stage, error) {
	stage := &Stage{
		ID:                yamlStage.ID,
		Type:              StageType(yamlStage.Type),
		Description:       yamlStage.Description,
		PreferredBackend:  yamlStage.PreferredBackend,
		PreferredHardware: yamlStage.PreferredHardware,
		Model:             yamlStage.Model,
	}

	// Convert forwarding policy
	if yamlStage.ForwardingPolicy.EnableConfidenceCheck ||
		yamlStage.ForwardingPolicy.EnableThermalCheck ||
		yamlStage.ForwardingPolicy.EnableQualityCheck {
		stage.ForwardingPolicy = &ForwardingPolicy{
			EnableConfidenceCheck: yamlStage.ForwardingPolicy.EnableConfidenceCheck,
			MinConfidence:         yamlStage.ForwardingPolicy.MinConfidence,
			MaxRetries:            yamlStage.ForwardingPolicy.MaxRetries,
			EscalationPath:        yamlStage.ForwardingPolicy.EscalationPath,
			EnableThermalCheck:    yamlStage.ForwardingPolicy.EnableThermalCheck,
			MaxTemperature:        yamlStage.ForwardingPolicy.MaxTemperature,
			MaxFanPercent:         yamlStage.ForwardingPolicy.MaxFanPercent,
			EnableQualityCheck:    yamlStage.ForwardingPolicy.EnableQualityCheck,
			QualityThreshold:      yamlStage.ForwardingPolicy.QualityThreshold,
		}
	}

	// TODO: Convert input/output transforms (requires template engine)

	return stage, nil
}

// convertYAMLToOptions converts YAML options to Options
func (pl *Loader) convertYAMLToOptions(yamlOptions OptionsYAML) *Options {
	return &Options{
		EnableStreaming: yamlOptions.EnableStreaming,
		PreserveContext: yamlOptions.PreserveContext,
		ContinueOnError: yamlOptions.ContinueOnError,
		CollectMetrics:  yamlOptions.CollectMetrics,
		ParallelStages:  yamlOptions.ParallelStages,
	}
}
