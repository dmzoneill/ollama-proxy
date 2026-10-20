package pipeline

import (
	"os"
	"testing"
)

// Tests for Loader

// TestNewPipelineLoader is part of the public API.
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

// TestListPipelinesEmpty is part of the public API.
func TestListPipelinesEmpty(t *testing.T) {
	loader := NewPipelineLoader()

	ids := loader.ListPipelines()
	if len(ids) != 0 {
		t.Errorf("Expected 0 pipelines, got %d", len(ids))
	}
}

// TestGetPipelineNotFound is part of the public API.
func TestGetPipelineNotFound(t *testing.T) {
	loader := NewPipelineLoader()

	_, err := loader.GetPipeline("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent pipeline")
	}
}

// TestConvertYAMLToOptionsBasic is part of the public API.
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

// TestConvertYAMLToOptionsDefaults is part of the public API.
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

// TestConvertYAMLToStageBasic is part of the public API.
func TestConvertYAMLToStageBasic(t *testing.T) {
	loader := NewPipelineLoader()

	yamlStage := StageYAML{
		ID:               strTestStage,
		Type:             string(StageTypeTextGen),
		Description:      "Test stage",
		PreferredBackend: strBackend1,
		Model:            strLlama37b,
	}

	stage, err := loader.convertYAMLToStage(yamlStage)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if stage == nil {
		t.Fatal("Expected non-nil stage")
	}

	if stage.ID != strTestStage {
		t.Errorf("Expected ID 'test-stage', got %s", stage.ID)
	}

	if stage.Type != StageTypeTextGen {
		t.Errorf("Expected type text_generation, got %s", stage.Type)
	}

	if stage.Description != "Test stage" {
		t.Errorf("Expected description 'Test stage', got %s", stage.Description)
	}

	if stage.PreferredBackend != strBackend1 {
		t.Errorf("Expected backend 'backend1', got %s", stage.PreferredBackend)
	}

	if stage.Model != strLlama37b {
		t.Errorf("Expected model 'llama3:7b', got %s", stage.Model)
	}
}

// TestConvertYAMLToStageAllTypes is part of the public API.
func TestConvertYAMLToStageAllTypes(t *testing.T) {
	loader := NewPipelineLoader()

	tests := []struct {
		yamlType  string
		stageType StageType
	}{
		{yamlType: "audio_to_text", stageType: StageTypeAudioToText},
		{yamlType: string(StageTypeTextGen), stageType: StageTypeTextGen},
		{yamlType: "text_to_audio", stageType: StageTypeTextToAudio},
		{yamlType: string(StageTypeEmbed), stageType: StageTypeEmbed},
		{yamlType: string(StageTypeCustom), stageType: StageTypeCustom},
	}

	for _, tt := range tests {
		yamlStage := StageYAML{
			ID:    strTestStage,
			Type:  tt.yamlType,
			Model: strTestModel,
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

// TestConvertYAMLToStageWithForwardingPolicy is part of the public API.
func TestConvertYAMLToStageWithForwardingPolicy(t *testing.T) {
	loader := NewPipelineLoader()

	yamlStage := StageYAML{
		ID:    strTestStage,
		Type:  string(StageTypeTextGen),
		Model: strLlama37b,
		ForwardingPolicy: ForwardingPolicyYAML{
			EnableConfidenceCheck: true,
			MinConfidence:         0.75,
			MaxRetries:            3,
			EscalationPath:        []string{strBackend1, strBackend2},
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

// TestConvertYAMLToStageNoForwardingPolicy is part of the public API.
func TestConvertYAMLToStageNoForwardingPolicy(t *testing.T) {
	loader := NewPipelineLoader()

	yamlStage := StageYAML{
		ID:    strTestStage,
		Type:  string(StageTypeTextGen),
		Model: strLlama37b,
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

// TestConvertYAMLToPipelineBasic is part of the public API.
func TestConvertYAMLToPipelineBasic(t *testing.T) {
	loader := NewPipelineLoader()

	yamlPipeline := YAML{
		ID:          strTestPipeline,
		Name:        strTestPipeline2,
		Description: strATestPipeline,
		Stages: []StageYAML{
			{
				ID:    strStage1,
				Type:  string(StageTypeTextGen),
				Model: strLlama37b,
			},
			{
				ID:    strStage2,
				Type:  string(StageTypeEmbed),
				Model: strNomicEmbedText,
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

	if pipeline.ID != strTestPipeline {
		t.Errorf("Expected ID 'test-pipeline', got %s", pipeline.ID)
	}

	if pipeline.Name != strTestPipeline2 {
		t.Errorf("Expected name 'Test Pipeline', got %s", pipeline.Name)
	}

	if pipeline.Description != strATestPipeline {
		t.Errorf("Expected description 'A test pipeline', got %s", pipeline.Description)
	}

	if len(pipeline.Stages) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(pipeline.Stages))
	}

	if pipeline.Stages[0].ID != strStage1 {
		t.Errorf("Expected first stage ID 'stage1', got %s", pipeline.Stages[0].ID)
	}

	if pipeline.Stages[1].ID != strStage2 {
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

// TestConvertYAMLToPipelineEmptyStages is part of the public API.
func TestConvertYAMLToPipelineEmptyStages(t *testing.T) {
	loader := NewPipelineLoader()

	yamlPipeline := YAML{
		ID:      strTestPipeline,
		Name:    strTestPipeline2,
		Stages:  []StageYAML{},
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

// TestConvertYAMLToPipelineComplexForwarding is part of the public API.
func TestConvertYAMLToPipelineComplexForwarding(t *testing.T) {
	loader := NewPipelineLoader()

	yamlPipeline := YAML{
		ID:   "complex-pipeline",
		Name: "Complex Pipeline",
		Stages: []StageYAML{
			{
				ID:                "adaptive-stage",
				Type:              string(StageTypeTextGen),
				PreferredHardware: strNpu,
				Model:             strQwen2505b,
				ForwardingPolicy: ForwardingPolicyYAML{
					EnableConfidenceCheck: true,
					MinConfidence:         0.75,
					MaxRetries:            3,
					EscalationPath:        []string{strBackend1, strBackend2, "backend3"},
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
	if stage.PreferredHardware != strNpu {
		t.Errorf("Expected hardware 'npu', got %s", stage.PreferredHardware)
	}

	if stage.ForwardingPolicy == nil {
		t.Fatal("Expected non-nil forwarding policy")
	}

	if len(stage.ForwardingPolicy.EscalationPath) != 3 {
		t.Errorf("Expected 3 backends in escalation, got %d", len(stage.ForwardingPolicy.EscalationPath))
	}
}

// TestGetPipelineAfterManualAdd is part of the public API.
func TestGetPipelineAfterManualAdd(t *testing.T) {
	loader := NewPipelineLoader()

	pipeline := &Pipeline{
		ID:      "manual-pipeline",
		Name:    "Manual Pipeline",
		Stages:  make([]*Stage, 0),
		Options: &Options{},
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

// TestListPipelinesAfterManualAdd is part of the public API.
func TestListPipelinesAfterManualAdd(t *testing.T) {
	loader := NewPipelineLoader()

	loader.pipelines[strPipeline1] = &Pipeline{ID: strPipeline1}
	loader.pipelines[strPipeline2] = &Pipeline{ID: strPipeline2}
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

	if !idMap[strPipeline1] {
		t.Error("Expected pipeline1 in list")
	}

	if !idMap[strPipeline2] {
		t.Error("Expected pipeline2 in list")
	}

	if !idMap["pipeline3"] {
		t.Error("Expected pipeline3 in list")
	}
}

// TestConvertYAMLToPipelineWithHardwarePreference is part of the public API.
func TestConvertYAMLToPipelineWithHardwarePreference(t *testing.T) {
	loader := NewPipelineLoader()

	yamlPipeline := YAML{
		ID:   "hardware-test",
		Name: "Hardware Test",
		Stages: []StageYAML{
			{
				ID:                "npu-stage",
				Type:              string(StageTypeEmbed),
				PreferredHardware: strNpu,
				Model:             strNomicEmbedText,
			},
			{
				ID:                "gpu-stage",
				Type:              string(StageTypeTextGen),
				PreferredHardware: strNvidia,
				Model:             strLlama370b,
			},
			{
				ID:                "igpu-stage",
				Type:              string(StageTypeTextGen),
				PreferredHardware: strIgpu,
				Model:             strLlama37b,
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

	if pipeline.Stages[0].PreferredHardware != strNpu {
		t.Error("Expected first stage to prefer npu")
	}

	if pipeline.Stages[1].PreferredHardware != strNvidia {
		t.Error("Expected second stage to prefer nvidia")
	}

	if pipeline.Stages[2].PreferredHardware != strIgpu {
		t.Error("Expected third stage to prefer igpu")
	}
}

// TestForwardingPolicyYAMLDefaults is part of the public API.
func TestForwardingPolicyYAMLDefaults(t *testing.T) {
	loader := NewPipelineLoader()

	yamlStage := StageYAML{
		ID:    strTest,
		Type:  string(StageTypeTextGen),
		Model: strTest,
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

// TestConvertYAMLToStageWithMultipleForwardingTypes is part of the public API.
func TestConvertYAMLToStageWithMultipleForwardingTypes(t *testing.T) {
	loader := NewPipelineLoader()

	yamlStage := StageYAML{
		ID:    "multi-check",
		Type:  string(StageTypeTextGen),
		Model: strTest,
		ForwardingPolicy: ForwardingPolicyYAML{
			EnableConfidenceCheck: true,
			MinConfidence:         0.7,
			EnableThermalCheck:    true,
			MaxTemperature:        80.0,
			EnableQualityCheck:    true,
			QualityThreshold:      0.75,
			MaxRetries:            2,
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

// TestConvertYAMLToStagePartialForwarding is part of the public API.
func TestConvertYAMLToStagePartialForwarding(t *testing.T) {
	loader := NewPipelineLoader()

	// Only enable confidence check, not others
	yamlStage := StageYAML{
		ID:    "confidence-only",
		Type:  string(StageTypeTextGen),
		Model: strTest,
		ForwardingPolicy: ForwardingPolicyYAML{
			EnableConfidenceCheck: true,
			MinConfidence:         0.8,
			MaxRetries:            2,
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

// TestPipelineConfigStructure is part of the public API.
func TestPipelineConfigStructure(t *testing.T) {
	config := Config{
		Pipelines: []YAML{
			{
				ID:   strPipeline1,
				Name: "First Pipeline",
			},
			{
				ID:   strPipeline2,
				Name: "Second Pipeline",
			},
		},
	}

	if len(config.Pipelines) != 2 {
		t.Errorf("Expected 2 pipelines in config, got %d", len(config.Pipelines))
	}

	if config.Pipelines[0].ID != strPipeline1 {
		t.Error("Expected first pipeline ID to be pipeline1")
	}

	if config.Pipelines[1].ID != strPipeline2 {
		t.Error("Expected second pipeline ID to be pipeline2")
	}
}

// TestStageYAMLStructure is part of the public API.
func TestStageYAMLStructure(t *testing.T) {
	stage := StageYAML{
		ID:                strTestStage,
		Type:              string(StageTypeTextGen),
		Description:       "Test description",
		PreferredBackend:  strBackend1,
		PreferredHardware: "gpu",
		Model:             strLlama37b,
		InputTransform:    map[string]interface{}{"type": "prefix", "value": strTest},
		OutputTransform:   map[string]interface{}{"type": "suffix", "value": "done"},
	}

	if stage.ID != strTestStage {
		t.Error("Expected stage ID")
	}

	if stage.Type != string(StageTypeTextGen) {
		t.Error("Expected stage type")
	}

	if stage.PreferredBackend != strBackend1 {
		t.Error("Expected preferred backend")
	}

	if stage.PreferredHardware != "gpu" {
		t.Error("Expected preferred hardware")
	}

	if stage.Model != strLlama37b {
		t.Error("Expected model")
	}

	if stage.InputTransform == nil {
		t.Error("Expected input transform")
	}

	if stage.OutputTransform == nil {
		t.Error("Expected output transform")
	}
}

// TestOptionsYAMLStructure is part of the public API.
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

// TestAddPipeline is part of the public API.
func TestAddPipeline(t *testing.T) {
	loader := NewPipelineLoader()

	pipeline := &Pipeline{
		ID:          strTestPipeline,
		Name:        strTestPipeline2,
		Description: strATestPipeline,
		Stages:      make([]*Stage, 0),
		Options:     &Options{},
	}

	// Add pipeline
	loader.AddPipeline(pipeline)

	// Verify it was added
	ids := loader.ListPipelines()
	if len(ids) != 1 {
		t.Errorf("Expected 1 pipeline, got %d", len(ids))
	}

	if ids[0] != strTestPipeline {
		t.Errorf("Expected pipeline ID 'test-pipeline', got %s", ids[0])
	}

	// Verify we can retrieve it
	retrieved, err := loader.GetPipeline(strTestPipeline)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if retrieved.ID != strTestPipeline {
		t.Errorf("Expected ID 'test-pipeline', got %s", retrieved.ID)
	}

	if retrieved.Name != strTestPipeline2 {
		t.Errorf("Expected name 'Test Pipeline', got %s", retrieved.Name)
	}
}

// TestLoadFromFileNotFound is part of the public API.
func TestLoadFromFileNotFound(t *testing.T) {
	loader := NewPipelineLoader()

	// Try to load from a non-existent file
	err := loader.LoadFromFile("/nonexistent/path/to/file.yaml")
	if err == nil {
		t.Fatal("Expected error when loading non-existent file")
	}
}

// TestLoadFromFileInvalidYAML is part of the public API.
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

// TestLoadFromFileValidYAML is part of the public API.
func TestLoadFromFileValidYAML(t *testing.T) {
	loader := NewPipelineLoader()

	// Create a temporary file with valid pipeline YAML
	tmpfile, err := os.CreateTemp("", "pipeline-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write valid pipeline YAML
	yamlContent := "pipelines:\n" +
		"- id: test-pipeline-1\n" +
		"  name: Test Pipeline 1\n" +
		"  description: First test pipeline\n" +
		"  stages:\n" +
		"  - id: stage1\n" +
		"    type: text_generation\n" +
		"    model: llama3:7b\n" +
		"  options:\n" +
		"    enable_streaming: true\n" +
		"- id: test-pipeline-2\n" +
		"  name: Test Pipeline 2\n" +
		"  description: Second test pipeline\n" +
		"  stages:\n" +
		"  - id: stage2\n" +
		"    type: embedding\n" +
		"    model: nomic-embed-text\n" +
		"  options:\n" +
		"    collect_metrics: true\n"
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
