package pipeline

import (
	"os"
	"testing"
)

// Tests for PipelineLoader

func TestNewPipelineLoader(t *testing.T) {
	loader := NewPipelineLoader()

	if loader == nil {
		t.Fatal("Expected non-nil loader")
	}

	if loader.pipelines == nil {
		t.Fatal("Expected non-nil pipelines map")
	}

	if len(loader.pipelines) != 0 {
		t.Errorf("Expected empty pipelines map, got %d pipelines", len(loader.pipelines))
	}
}

func TestListPipelinesEmpty(t *testing.T) {
	loader := NewPipelineLoader()

	ids := loader.ListPipelines()
	if len(ids) != 0 {
		t.Errorf("Expected 0 pipelines, got %d", len(ids))
	}
}

func TestGetPipelineNotFound(t *testing.T) {
	loader := NewPipelineLoader()

	_, err := loader.GetPipeline("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent pipeline")
	}
}

func TestConvertYAMLToOptionsBasic(t *testing.T) {
	loader := NewPipelineLoader()

	yamlOptions := OptionsYAML{
		EnableStreaming: true,
		PreserveContext: true,
		ContinueOnError: false,
		CollectMetrics:  true,
		ParallelStages:  false,
		LatencyCritical: false,
	}

	options := loader.convertYAMLToOptions(yamlOptions)

	if !options.EnableStreaming {
		t.Error("Expected EnableStreaming to be true")
	}

	if !options.PreserveContext {
		t.Error("Expected PreserveContext to be true")
	}

	if options.ContinueOnError {
		t.Error("Expected ContinueOnError to be false")
	}

	if !options.CollectMetrics {
		t.Error("Expected CollectMetrics to be true")
	}

	if options.ParallelStages {
		t.Error("Expected ParallelStages to be false")
	}
}

func TestConvertYAMLToOptionsDefaults(t *testing.T) {
	loader := NewPipelineLoader()

	yamlOptions := OptionsYAML{}

	options := loader.convertYAMLToOptions(yamlOptions)

	if options == nil {
		t.Fatal("Expected non-nil options")
	}

	if options.EnableStreaming {
		t.Error("Expected EnableStreaming to be false by default")
	}

	if options.ContinueOnError {
		t.Error("Expected ContinueOnError to be false by default")
	}
}

func TestConvertYAMLToStageBasic(t *testing.T) {
	loader := NewPipelineLoader()

	yamlStage := StageYAML{
		ID:               "test-stage",
		Type:             "text_generation",
		Description:      "Test stage",
		PreferredBackend: "backend1",
		Model:            "llama3:7b",
	}

	stage, err := loader.convertYAMLToStage(yamlStage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if stage == nil {
		t.Fatal("Expected non-nil stage")
	}

	if stage.ID != "test-stage" {
		t.Errorf("Expected ID 'test-stage', got %s", stage.ID)
	}

	if stage.Type != StageTypeTextGen {
		t.Errorf("Expected type text_generation, got %s", stage.Type)
	}

	if stage.Description != "Test stage" {
		t.Errorf("Expected description 'Test stage', got %s", stage.Description)
	}

	if stage.PreferredBackend != "backend1" {
		t.Errorf("Expected backend 'backend1', got %s", stage.PreferredBackend)
	}

	if stage.Model != "llama3:7b" {
		t.Errorf("Expected model 'llama3:7b', got %s", stage.Model)
	}
}

func TestConvertYAMLToStageAllTypes(t *testing.T) {
	loader := NewPipelineLoader()

	tests := []struct {
		yamlType  string
		stageType StageType
	}{
		{"audio_to_text", StageTypeAudioToText},
		{"text_generation", StageTypeTextGen},
		{"text_to_audio", StageTypeTextToAudio},
		{"embedding", StageTypeEmbed},
		{"custom", StageTypeCustom},
	}

	for _, tt := range tests {
		yamlStage := StageYAML{
			ID:    "test-stage",
			Type:  tt.yamlType,
			Model: "test-model",
		}

		stage, err := loader.convertYAMLToStage(yamlStage)
		if err != nil {
			t.Fatalf("Expected no error for %s, got: %v", tt.yamlType, err)
		}

		if stage.Type != tt.stageType {
			t.Errorf("Expected type %s, got %s", tt.stageType, stage.Type)
		}
	}
}

func TestConvertYAMLToStageWithForwardingPolicy(t *testing.T) {
	loader := NewPipelineLoader()

	yamlStage := StageYAML{
		ID:    "test-stage",
		Type:  "text_generation",
		Model: "llama3:7b",
		ForwardingPolicy: ForwardingPolicyYAML{
			EnableConfidenceCheck: true,
			MinConfidence:         0.75,
			MaxRetries:           3,
			EscalationPath:        []string{"backend1", "backend2"},
			EnableThermalCheck:    true,
			MaxTemperature:        85.0,
			MaxFanPercent:         80,
			EnableQualityCheck:    true,
			QualityThreshold:      0.8,
		},
	}

	stage, err := loader.convertYAMLToStage(yamlStage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if stage.ForwardingPolicy == nil {
		t.Fatal("Expected non-nil forwarding policy")
	}

	if !stage.ForwardingPolicy.EnableConfidenceCheck {
		t.Error("Expected confidence check to be enabled")
	}

	if stage.ForwardingPolicy.MinConfidence != 0.75 {
		t.Errorf("Expected MinConfidence 0.75, got %f", stage.ForwardingPolicy.MinConfidence)
	}

	if stage.ForwardingPolicy.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", stage.ForwardingPolicy.MaxRetries)
	}

	if len(stage.ForwardingPolicy.EscalationPath) != 2 {
		t.Errorf("Expected 2 backends in escalation path, got %d", len(stage.ForwardingPolicy.EscalationPath))
	}

	if !stage.ForwardingPolicy.EnableThermalCheck {
		t.Error("Expected thermal check to be enabled")
	}

	if stage.ForwardingPolicy.MaxTemperature != 85.0 {
		t.Errorf("Expected MaxTemperature 85.0, got %f", stage.ForwardingPolicy.MaxTemperature)
	}

	if !stage.ForwardingPolicy.EnableQualityCheck {
		t.Error("Expected quality check to be enabled")
	}

	if stage.ForwardingPolicy.QualityThreshold != 0.8 {
		t.Errorf("Expected QualityThreshold 0.8, got %f", stage.ForwardingPolicy.QualityThreshold)
	}
}

func TestConvertYAMLToStageNoForwardingPolicy(t *testing.T) {
	loader := NewPipelineLoader()

	yamlStage := StageYAML{
		ID:    "test-stage",
		Type:  "text_generation",
		Model: "llama3:7b",
		ForwardingPolicy: ForwardingPolicyYAML{
			EnableConfidenceCheck: false,
			EnableThermalCheck:    false,
			EnableQualityCheck:    false,
		},
	}

	stage, err := loader.convertYAMLToStage(yamlStage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should not create forwarding policy when all checks are disabled
	if stage.ForwardingPolicy != nil {
		t.Error("Expected nil forwarding policy when all checks disabled")
	}
}

func TestConvertYAMLToPipelineBasic(t *testing.T) {
	loader := NewPipelineLoader()

	yamlPipeline := PipelineYAML{
		ID:          "test-pipeline",
		Name:        "Test Pipeline",
		Description: "A test pipeline",
		Stages: []StageYAML{
			{
				ID:    "stage1",
				Type:  "text_generation",
				Model: "llama3:7b",
			},
			{
				ID:    "stage2",
				Type:  "embedding",
				Model: "nomic-embed-text",
			},
		},
		Options: OptionsYAML{
			EnableStreaming: true,
			CollectMetrics:  true,
		},
	}

	pipeline, err := loader.convertYAMLToPipeline(yamlPipeline)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if pipeline == nil {
		t.Fatal("Expected non-nil pipeline")
	}

	if pipeline.ID != "test-pipeline" {
		t.Errorf("Expected ID 'test-pipeline', got %s", pipeline.ID)
	}

	if pipeline.Name != "Test Pipeline" {
		t.Errorf("Expected name 'Test Pipeline', got %s", pipeline.Name)
	}

	if pipeline.Description != "A test pipeline" {
		t.Errorf("Expected description 'A test pipeline', got %s", pipeline.Description)
	}

	if len(pipeline.Stages) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(pipeline.Stages))
	}

	if pipeline.Stages[0].ID != "stage1" {
		t.Errorf("Expected first stage ID 'stage1', got %s", pipeline.Stages[0].ID)
	}

	if pipeline.Stages[1].ID != "stage2" {
		t.Errorf("Expected second stage ID 'stage2', got %s", pipeline.Stages[1].ID)
	}

	if pipeline.Options == nil {
		t.Fatal("Expected non-nil options")
	}

	if !pipeline.Options.EnableStreaming {
		t.Error("Expected streaming to be enabled")
	}

	if !pipeline.Options.CollectMetrics {
		t.Error("Expected metrics collection to be enabled")
	}
}

func TestConvertYAMLToPipelineEmptyStages(t *testing.T) {
	loader := NewPipelineLoader()

	yamlPipeline := PipelineYAML{
		ID:     "test-pipeline",
		Name:   "Test Pipeline",
		Stages: []StageYAML{},
		Options: OptionsYAML{},
	}

	pipeline, err := loader.convertYAMLToPipeline(yamlPipeline)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(pipeline.Stages) != 0 {
		t.Errorf("Expected 0 stages, got %d", len(pipeline.Stages))
	}
}

func TestConvertYAMLToPipelineComplexForwarding(t *testing.T) {
	loader := NewPipelineLoader()

	yamlPipeline := PipelineYAML{
		ID:   "complex-pipeline",
		Name: "Complex Pipeline",
		Stages: []StageYAML{
			{
				ID:               "adaptive-stage",
				Type:             "text_generation",
				PreferredHardware: "npu",
				Model:            "qwen2.5:0.5b",
				ForwardingPolicy: ForwardingPolicyYAML{
					EnableConfidenceCheck: true,
					MinConfidence:         0.75,
					MaxRetries:           3,
					EscalationPath:        []string{"backend1", "backend2", "backend3"},
					EnableThermalCheck:    true,
					MaxTemperature:        85.0,
				},
			},
		},
		Options: OptionsYAML{},
	}

	pipeline, err := loader.convertYAMLToPipeline(yamlPipeline)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	stage := pipeline.Stages[0]
	if stage.PreferredHardware != "npu" {
		t.Errorf("Expected hardware 'npu', got %s", stage.PreferredHardware)
	}

	if stage.ForwardingPolicy == nil {
		t.Fatal("Expected non-nil forwarding policy")
	}

	if len(stage.ForwardingPolicy.EscalationPath) != 3 {
		t.Errorf("Expected 3 backends in escalation, got %d", len(stage.ForwardingPolicy.EscalationPath))
	}
}

func TestGetPipelineAfterManualAdd(t *testing.T) {
	loader := NewPipelineLoader()

	pipeline := &Pipeline{
		ID:     "manual-pipeline",
		Name:   "Manual Pipeline",
		Stages: make([]*Stage, 0),
		Options: &PipelineOptions{},
	}

	loader.pipelines["manual-pipeline"] = pipeline

	retrieved, err := loader.GetPipeline("manual-pipeline")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if retrieved.ID != "manual-pipeline" {
		t.Errorf("Expected ID 'manual-pipeline', got %s", retrieved.ID)
	}
}

func TestListPipelinesAfterManualAdd(t *testing.T) {
	loader := NewPipelineLoader()

	loader.pipelines["pipeline1"] = &Pipeline{ID: "pipeline1"}
	loader.pipelines["pipeline2"] = &Pipeline{ID: "pipeline2"}
	loader.pipelines["pipeline3"] = &Pipeline{ID: "pipeline3"}

	ids := loader.ListPipelines()

	if len(ids) != 3 {
		t.Errorf("Expected 3 pipelines, got %d", len(ids))
	}

	// Convert to map for easier checking
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	if !idMap["pipeline1"] {
		t.Error("Expected pipeline1 in list")
	}

	if !idMap["pipeline2"] {
		t.Error("Expected pipeline2 in list")
	}

	if !idMap["pipeline3"] {
		t.Error("Expected pipeline3 in list")
	}
}

func TestConvertYAMLToPipelineWithHardwarePreference(t *testing.T) {
	loader := NewPipelineLoader()

	yamlPipeline := PipelineYAML{
		ID:   "hardware-test",
		Name: "Hardware Test",
		Stages: []StageYAML{
			{
				ID:               "npu-stage",
				Type:             "embedding",
				PreferredHardware: "npu",
				Model:            "nomic-embed-text",
			},
			{
				ID:               "gpu-stage",
				Type:             "text_generation",
				PreferredHardware: "nvidia",
				Model:            "llama3:70b",
			},
			{
				ID:               "igpu-stage",
				Type:             "text_generation",
				PreferredHardware: "igpu",
				Model:            "llama3:7b",
			},
		},
		Options: OptionsYAML{},
	}

	pipeline, err := loader.convertYAMLToPipeline(yamlPipeline)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(pipeline.Stages) != 3 {
		t.Errorf("Expected 3 stages, got %d", len(pipeline.Stages))
	}

	if pipeline.Stages[0].PreferredHardware != "npu" {
		t.Error("Expected first stage to prefer npu")
	}

	if pipeline.Stages[1].PreferredHardware != "nvidia" {
		t.Error("Expected second stage to prefer nvidia")
	}

	if pipeline.Stages[2].PreferredHardware != "igpu" {
		t.Error("Expected third stage to prefer igpu")
	}
}

func TestForwardingPolicyYAMLDefaults(t *testing.T) {
	loader := NewPipelineLoader()

	yamlStage := StageYAML{
		ID:    "test",
		Type:  "text_generation",
		Model: "test",
		ForwardingPolicy: ForwardingPolicyYAML{
			EnableConfidenceCheck: true,
			// Other fields use zero values
		},
	}

	stage, err := loader.convertYAMLToStage(yamlStage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if stage.ForwardingPolicy == nil {
		t.Fatal("Expected non-nil forwarding policy")
	}

	if !stage.ForwardingPolicy.EnableConfidenceCheck {
		t.Error("Expected confidence check enabled")
	}

	if stage.ForwardingPolicy.MinConfidence != 0 {
		t.Errorf("Expected MinConfidence 0, got %f", stage.ForwardingPolicy.MinConfidence)
	}

	if stage.ForwardingPolicy.MaxRetries != 0 {
		t.Errorf("Expected MaxRetries 0, got %d", stage.ForwardingPolicy.MaxRetries)
	}
}

func TestConvertYAMLToStageWithMultipleForwardingTypes(t *testing.T) {
	loader := NewPipelineLoader()

	yamlStage := StageYAML{
		ID:    "multi-check",
		Type:  "text_generation",
		Model: "test",
		ForwardingPolicy: ForwardingPolicyYAML{
			EnableConfidenceCheck: true,
			MinConfidence:         0.7,
			EnableThermalCheck:    true,
			MaxTemperature:        80.0,
			EnableQualityCheck:    true,
			QualityThreshold:      0.75,
			MaxRetries:           2,
			EscalationPath:        []string{"b1", "b2"},
		},
	}

	stage, err := loader.convertYAMLToStage(yamlStage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	policy := stage.ForwardingPolicy
	if policy == nil {
		t.Fatal("Expected non-nil policy")
	}

	if !policy.EnableConfidenceCheck || !policy.EnableThermalCheck || !policy.EnableQualityCheck {
		t.Error("Expected all checks to be enabled")
	}

	if policy.MinConfidence != 0.7 {
		t.Errorf("Expected MinConfidence 0.7, got %f", policy.MinConfidence)
	}

	if policy.MaxTemperature != 80.0 {
		t.Errorf("Expected MaxTemperature 80.0, got %f", policy.MaxTemperature)
	}

	if policy.QualityThreshold != 0.75 {
		t.Errorf("Expected QualityThreshold 0.75, got %f", policy.QualityThreshold)
	}
}

func TestConvertYAMLToStagePartialForwarding(t *testing.T) {
	loader := NewPipelineLoader()

	// Only enable confidence check, not others
	yamlStage := StageYAML{
		ID:    "confidence-only",
		Type:  "text_generation",
		Model: "test",
		ForwardingPolicy: ForwardingPolicyYAML{
			EnableConfidenceCheck: true,
			MinConfidence:         0.8,
			MaxRetries:           2,
			EscalationPath:        []string{"b1", "b2"},
			EnableThermalCheck:    false,
			EnableQualityCheck:    false,
		},
	}

	stage, err := loader.convertYAMLToStage(yamlStage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if stage.ForwardingPolicy == nil {
		t.Fatal("Expected non-nil policy")
	}

	if !stage.ForwardingPolicy.EnableConfidenceCheck {
		t.Error("Expected confidence check enabled")
	}

	if stage.ForwardingPolicy.EnableThermalCheck || stage.ForwardingPolicy.EnableQualityCheck {
		t.Error("Expected other checks to be disabled")
	}
}

func TestPipelineConfigStructure(t *testing.T) {
	config := PipelineConfig{
		Pipelines: []PipelineYAML{
			{
				ID:   "pipeline1",
				Name: "First Pipeline",
			},
			{
				ID:   "pipeline2",
				Name: "Second Pipeline",
			},
		},
	}

	if len(config.Pipelines) != 2 {
		t.Errorf("Expected 2 pipelines in config, got %d", len(config.Pipelines))
	}

	if config.Pipelines[0].ID != "pipeline1" {
		t.Error("Expected first pipeline ID to be pipeline1")
	}

	if config.Pipelines[1].ID != "pipeline2" {
		t.Error("Expected second pipeline ID to be pipeline2")
	}
}

func TestStageYAMLStructure(t *testing.T) {
	stage := StageYAML{
		ID:               "test-stage",
		Type:             "text_generation",
		Description:      "Test description",
		PreferredBackend: "backend1",
		PreferredHardware: "gpu",
		Model:            "llama3:7b",
		InputTransform:   map[string]interface{}{"type": "prefix", "value": "test"},
		OutputTransform:  map[string]interface{}{"type": "suffix", "value": "done"},
	}

	if stage.ID != "test-stage" {
		t.Error("Expected stage ID")
	}

	if stage.Type != "text_generation" {
		t.Error("Expected stage type")
	}

	if stage.PreferredBackend != "backend1" {
		t.Error("Expected preferred backend")
	}

	if stage.PreferredHardware != "gpu" {
		t.Error("Expected preferred hardware")
	}

	if stage.Model != "llama3:7b" {
		t.Error("Expected model")
	}

	if stage.InputTransform == nil {
		t.Error("Expected input transform")
	}

	if stage.OutputTransform == nil {
		t.Error("Expected output transform")
	}
}

func TestOptionsYAMLStructure(t *testing.T) {
	options := OptionsYAML{
		EnableStreaming: true,
		PreserveContext: true,
		ContinueOnError: true,
		CollectMetrics:  true,
		ParallelStages:  true,
		LatencyCritical: true,
	}

	if !options.EnableStreaming {
		t.Error("Expected EnableStreaming")
	}

	if !options.PreserveContext {
		t.Error("Expected PreserveContext")
	}

	if !options.ContinueOnError {
		t.Error("Expected ContinueOnError")
	}

	if !options.CollectMetrics {
		t.Error("Expected CollectMetrics")
	}

	if !options.ParallelStages {
		t.Error("Expected ParallelStages")
	}

	if !options.LatencyCritical {
		t.Error("Expected LatencyCritical")
	}
}

func TestAddPipeline(t *testing.T) {
	loader := NewPipelineLoader()

	pipeline := &Pipeline{
		ID:          "test-pipeline",
		Name:        "Test Pipeline",
		Description: "A test pipeline",
		Stages:      make([]*Stage, 0),
		Options:     &PipelineOptions{},
	}

	// Add pipeline
	loader.AddPipeline(pipeline)

	// Verify it was added
	ids := loader.ListPipelines()
	if len(ids) != 1 {
		t.Errorf("Expected 1 pipeline, got %d", len(ids))
	}

	if ids[0] != "test-pipeline" {
		t.Errorf("Expected pipeline ID 'test-pipeline', got %s", ids[0])
	}

	// Verify we can retrieve it
	retrieved, err := loader.GetPipeline("test-pipeline")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if retrieved.ID != "test-pipeline" {
		t.Errorf("Expected ID 'test-pipeline', got %s", retrieved.ID)
	}

	if retrieved.Name != "Test Pipeline" {
		t.Errorf("Expected name 'Test Pipeline', got %s", retrieved.Name)
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	loader := NewPipelineLoader()

	// Try to load from a non-existent file
	err := loader.LoadFromFile("/nonexistent/path/to/file.yaml")
	if err == nil {
		t.Fatal("Expected error when loading non-existent file")
	}
}

func TestLoadFromFileInvalidYAML(t *testing.T) {
	loader := NewPipelineLoader()

	// Create a temporary file with invalid YAML
	tmpfile, err := os.CreateTemp("", "invalid-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write invalid YAML
	if _, err := tmpfile.WriteString("invalid: yaml: syntax: error:\n\t-"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	// Try to load
	err = loader.LoadFromFile(tmpfile.Name())
	if err == nil {
		t.Fatal("Expected error when parsing invalid YAML")
	}
}

func TestLoadFromFileValidYAML(t *testing.T) {
	loader := NewPipelineLoader()

	// Create a temporary file with valid pipeline YAML
	tmpfile, err := os.CreateTemp("", "pipeline-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write valid pipeline YAML
	yamlContent := `pipelines:
  - id: test-pipeline-1
    name: Test Pipeline 1
    description: First test pipeline
    stages:
      - id: stage1
        type: text_generation
        model: llama3:7b
    options:
      enable_streaming: true
  - id: test-pipeline-2
    name: Test Pipeline 2
    description: Second test pipeline
    stages:
      - id: stage2
        type: embedding
        model: nomic-embed-text
    options:
      collect_metrics: true
`
	if _, err := tmpfile.WriteString(yamlContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	// Load from file
	err = loader.LoadFromFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load from file: %v", err)
	}

	// Verify pipelines were loaded
	ids := loader.ListPipelines()
	if len(ids) != 2 {
		t.Errorf("Expected 2 pipelines, got %d", len(ids))
	}

	// Verify first pipeline
	pipeline1, err := loader.GetPipeline("test-pipeline-1")
	if err != nil {
		t.Fatalf("Failed to get pipeline 1: %v", err)
	}

	if pipeline1.Name != "Test Pipeline 1" {
		t.Errorf("Expected name 'Test Pipeline 1', got %s", pipeline1.Name)
	}

	if len(pipeline1.Stages) != 1 {
		t.Errorf("Expected 1 stage, got %d", len(pipeline1.Stages))
	}

	// Verify second pipeline
	pipeline2, err := loader.GetPipeline("test-pipeline-2")
	if err != nil {
		t.Fatalf("Failed to get pipeline 2: %v", err)
	}

	if pipeline2.Name != "Test Pipeline 2" {
		t.Errorf("Expected name 'Test Pipeline 2', got %s", pipeline2.Name)
	}
}
