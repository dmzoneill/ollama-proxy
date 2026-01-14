package pipeline

import (
	"testing"
)

// Tests for example pipelines

func TestVoiceAssistantPipeline(t *testing.T) {
	pipeline := VoiceAssistantPipeline()

	if pipeline == nil {
		t.Fatal("Expected non-nil pipeline")
	}

	if pipeline.ID != "voice-assistant" {
		t.Errorf("Expected ID 'voice-assistant', got %s", pipeline.ID)
	}

	if pipeline.Name != "Voice Assistant" {
		t.Errorf("Expected name 'Voice Assistant', got %s", pipeline.Name)
	}

	if len(pipeline.Stages) != 3 {
		t.Errorf("Expected 3 stages, got %d", len(pipeline.Stages))
	}

	// Check first stage
	if pipeline.Stages[0].ID != "voice-to-text" {
		t.Error("Expected first stage to be 'voice-to-text'")
	}

	if pipeline.Stages[0].Type != StageTypeAudioToText {
		t.Error("Expected first stage type to be audio_to_text")
	}

	if pipeline.Stages[0].PreferredHardware != "npu" {
		t.Error("Expected first stage to prefer NPU")
	}

	if pipeline.Stages[0].ForwardingPolicy == nil {
		t.Error("Expected first stage to have forwarding policy")
	}

	if !pipeline.Stages[0].ForwardingPolicy.EnableConfidenceCheck {
		t.Error("Expected first stage to have confidence check")
	}

	if pipeline.Stages[0].ForwardingPolicy.MinConfidence != 0.7 {
		t.Errorf("Expected min confidence 0.7, got %f", pipeline.Stages[0].ForwardingPolicy.MinConfidence)
	}

	// Check second stage
	if pipeline.Stages[1].ID != "process-text" {
		t.Error("Expected second stage to be 'process-text'")
	}

	if pipeline.Stages[1].Type != StageTypeTextGen {
		t.Error("Expected second stage type to be text_generation")
	}

	if pipeline.Stages[1].PreferredHardware != "igpu" {
		t.Error("Expected second stage to prefer iGPU")
	}

	if pipeline.Stages[1].ForwardingPolicy == nil {
		t.Error("Expected second stage to have forwarding policy")
	}

	if !pipeline.Stages[1].ForwardingPolicy.EnableThermalCheck {
		t.Error("Expected second stage to have thermal check")
	}

	// Check third stage
	if pipeline.Stages[2].ID != "text-to-voice" {
		t.Error("Expected third stage to be 'text-to-voice'")
	}

	if pipeline.Stages[2].Type != StageTypeTextToAudio {
		t.Error("Expected third stage type to be text_to_audio")
	}

	// Check options
	if pipeline.Options == nil {
		t.Fatal("Expected non-nil options")
	}

	if !pipeline.Options.EnableStreaming {
		t.Error("Expected streaming to be enabled")
	}

	if !pipeline.Options.PreserveContext {
		t.Error("Expected context preservation to be enabled")
	}

	if pipeline.Options.ContinueOnError {
		t.Error("Expected ContinueOnError to be false")
	}

	if !pipeline.Options.CollectMetrics {
		t.Error("Expected metrics collection to be enabled")
	}
}

func TestAdaptiveTextPipeline(t *testing.T) {
	pipeline := AdaptiveTextPipeline()

	if pipeline == nil {
		t.Fatal("Expected non-nil pipeline")
	}

	if pipeline.ID != "adaptive-text" {
		t.Errorf("Expected ID 'adaptive-text', got %s", pipeline.ID)
	}

	if pipeline.Name != "Adaptive Text Generation" {
		t.Errorf("Expected name 'Adaptive Text Generation', got %s", pipeline.Name)
	}

	if len(pipeline.Stages) != 1 {
		t.Errorf("Expected 1 stage, got %d", len(pipeline.Stages))
	}

	stage := pipeline.Stages[0]
	if stage.ID != "generate-text" {
		t.Error("Expected stage ID 'generate-text'")
	}

	if stage.Type != StageTypeTextGen {
		t.Error("Expected text generation stage type")
	}

	if stage.PreferredHardware != "npu" {
		t.Error("Expected NPU preference")
	}

	if stage.Model != "qwen2.5:0.5b" {
		t.Errorf("Expected model qwen2.5:0.5b, got %s", stage.Model)
	}

	if stage.ForwardingPolicy == nil {
		t.Fatal("Expected non-nil forwarding policy")
	}

	if !stage.ForwardingPolicy.EnableConfidenceCheck {
		t.Error("Expected confidence check")
	}

	if stage.ForwardingPolicy.MinConfidence != 0.75 {
		t.Errorf("Expected min confidence 0.75, got %f", stage.ForwardingPolicy.MinConfidence)
	}

	if stage.ForwardingPolicy.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", stage.ForwardingPolicy.MaxRetries)
	}

	if len(stage.ForwardingPolicy.EscalationPath) != 3 {
		t.Errorf("Expected 3 backends in escalation path, got %d", len(stage.ForwardingPolicy.EscalationPath))
	}

	if !stage.ForwardingPolicy.EnableThermalCheck {
		t.Error("Expected thermal check")
	}

	if stage.ForwardingPolicy.MaxTemperature != 85.0 {
		t.Errorf("Expected max temperature 85.0, got %f", stage.ForwardingPolicy.MaxTemperature)
	}

	if !pipeline.Options.CollectMetrics {
		t.Error("Expected metrics collection")
	}

	if pipeline.Options.EnableStreaming {
		t.Error("Expected streaming to be disabled")
	}
}

func TestCodeGenerationPipeline(t *testing.T) {
	pipeline := CodeGenerationPipeline()

	if pipeline == nil {
		t.Fatal("Expected non-nil pipeline")
	}

	if pipeline.ID != "code-generation" {
		t.Errorf("Expected ID 'code-generation', got %s", pipeline.ID)
	}

	if pipeline.Name != "Code Generation with Review" {
		t.Errorf("Expected name 'Code Generation with Review', got %s", pipeline.Name)
	}

	if len(pipeline.Stages) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(pipeline.Stages))
	}

	// Check first stage
	if pipeline.Stages[0].ID != "draft-code" {
		t.Error("Expected first stage to be 'draft-code'")
	}

	if pipeline.Stages[0].Type != StageTypeTextGen {
		t.Error("Expected text generation stage")
	}

	if pipeline.Stages[0].PreferredHardware != "igpu" {
		t.Error("Expected iGPU preference")
	}

	if pipeline.Stages[0].Model != "deepseek-coder:6.7b" {
		t.Errorf("Expected deepseek-coder:6.7b model")
	}

	if pipeline.Stages[0].ForwardingPolicy == nil {
		t.Error("Expected forwarding policy")
	}

	if !pipeline.Stages[0].ForwardingPolicy.EnableQualityCheck {
		t.Error("Expected quality check")
	}

	if pipeline.Stages[0].ForwardingPolicy.QualityThreshold != 0.8 {
		t.Errorf("Expected quality threshold 0.8, got %f", pipeline.Stages[0].ForwardingPolicy.QualityThreshold)
	}

	// Check second stage
	if pipeline.Stages[1].ID != "review-code" {
		t.Error("Expected second stage to be 'review-code'")
	}

	if pipeline.Stages[1].PreferredHardware != "nvidia" {
		t.Error("Expected NVIDIA preference")
	}

	if pipeline.Stages[1].Model != "llama3:70b" {
		t.Error("Expected llama3:70b model")
	}

	if pipeline.Stages[1].InputTransform == nil {
		t.Error("Expected input transform")
	}

	// Check options
	if !pipeline.Options.PreserveContext {
		t.Error("Expected context preservation")
	}

	if !pipeline.Options.CollectMetrics {
		t.Error("Expected metrics collection")
	}

	if pipeline.Options.EnableStreaming {
		t.Error("Expected streaming to be disabled")
	}
}

func TestThermalFailoverPipeline(t *testing.T) {
	pipeline := ThermalFailoverPipeline()

	if pipeline == nil {
		t.Fatal("Expected non-nil pipeline")
	}

	if pipeline.ID != "thermal-failover" {
		t.Errorf("Expected ID 'thermal-failover', got %s", pipeline.ID)
	}

	if pipeline.Name != "Long Generation with Thermal Protection" {
		t.Errorf("Expected name 'Long Generation with Thermal Protection', got %s", pipeline.Name)
	}

	if len(pipeline.Stages) != 1 {
		t.Errorf("Expected 1 stage, got %d", len(pipeline.Stages))
	}

	stage := pipeline.Stages[0]
	if stage.ID != "long-generation" {
		t.Error("Expected stage 'long-generation'")
	}

	if stage.Type != StageTypeTextGen {
		t.Error("Expected text generation stage")
	}

	if stage.PreferredHardware != "nvidia" {
		t.Error("Expected NVIDIA preference")
	}

	if stage.Model != "llama3:70b" {
		t.Error("Expected llama3:70b model")
	}

	if stage.ForwardingPolicy == nil {
		t.Fatal("Expected non-nil forwarding policy")
	}

	if !stage.ForwardingPolicy.EnableThermalCheck {
		t.Error("Expected thermal check")
	}

	if stage.ForwardingPolicy.MaxTemperature != 87.0 {
		t.Errorf("Expected max temperature 87.0, got %f", stage.ForwardingPolicy.MaxTemperature)
	}

	if stage.ForwardingPolicy.MaxFanPercent != 85 {
		t.Errorf("Expected max fan percent 85, got %d", stage.ForwardingPolicy.MaxFanPercent)
	}

	if len(stage.ForwardingPolicy.EscalationPath) != 3 {
		t.Errorf("Expected 3 backends in escalation path, got %d", len(stage.ForwardingPolicy.EscalationPath))
	}

	// Check options
	if !pipeline.Options.EnableStreaming {
		t.Error("Expected streaming to be enabled")
	}

	if !pipeline.Options.PreserveContext {
		t.Error("Expected context preservation")
	}

	if pipeline.Options.ContinueOnError {
		t.Error("Expected ContinueOnError to be false")
	}

	if !pipeline.Options.CollectMetrics {
		t.Error("Expected metrics collection")
	}
}

func TestMultiModalRAGPipeline(t *testing.T) {
	pipeline := MultiModalRAGPipeline()

	if pipeline == nil {
		t.Fatal("Expected non-nil pipeline")
	}

	if pipeline.ID != "rag-pipeline" {
		t.Errorf("Expected ID 'rag-pipeline', got %s", pipeline.ID)
	}

	if pipeline.Name != "RAG with Embeddings" {
		t.Errorf("Expected name 'RAG with Embeddings', got %s", pipeline.Name)
	}

	if len(pipeline.Stages) != 3 {
		t.Errorf("Expected 3 stages, got %d", len(pipeline.Stages))
	}

	// Check first stage - embedding
	if pipeline.Stages[0].ID != "generate-embedding" {
		t.Error("Expected first stage 'generate-embedding'")
	}

	if pipeline.Stages[0].Type != StageTypeEmbed {
		t.Error("Expected embedding stage type")
	}

	if pipeline.Stages[0].PreferredHardware != "npu" {
		t.Error("Expected NPU preference")
	}

	if pipeline.Stages[0].Model != "nomic-embed-text" {
		t.Error("Expected nomic-embed-text model")
	}

	// Check second stage - retrieval (custom)
	if pipeline.Stages[1].ID != "retrieve-context" {
		t.Error("Expected second stage 'retrieve-context'")
	}

	if pipeline.Stages[1].Type != StageTypeCustom {
		t.Error("Expected custom stage type")
	}

	if pipeline.Stages[1].InputTransform == nil {
		t.Error("Expected input transform for retrieval stage")
	}

	// Check third stage - generation
	if pipeline.Stages[2].ID != "generate-answer" {
		t.Error("Expected third stage 'generate-answer'")
	}

	if pipeline.Stages[2].Type != StageTypeTextGen {
		t.Error("Expected text generation stage")
	}

	if pipeline.Stages[2].PreferredHardware != "nvidia" {
		t.Error("Expected NVIDIA preference")
	}

	if pipeline.Stages[2].Model != "llama3:70b" {
		t.Error("Expected llama3:70b model")
	}

	if pipeline.Stages[2].InputTransform == nil {
		t.Error("Expected input transform for generation stage")
	}

	// Check options
	if !pipeline.Options.EnableStreaming {
		t.Error("Expected streaming enabled")
	}

	if !pipeline.Options.PreserveContext {
		t.Error("Expected context preservation")
	}

	if !pipeline.Options.CollectMetrics {
		t.Error("Expected metrics collection")
	}
}

func TestSpeculativeExecutionPipeline(t *testing.T) {
	pipeline := SpeculativeExecutionPipeline()

	if pipeline == nil {
		t.Fatal("Expected non-nil pipeline")
	}

	if pipeline.ID != "speculative-execution" {
		t.Errorf("Expected ID 'speculative-execution', got %s", pipeline.ID)
	}

	if pipeline.Name != "Speculative Execution" {
		t.Errorf("Expected name 'Speculative Execution', got %s", pipeline.Name)
	}

	if len(pipeline.Stages) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(pipeline.Stages))
	}

	// Check first stage
	if pipeline.Stages[0].ID != "generate-candidates" {
		t.Error("Expected first stage 'generate-candidates'")
	}

	if pipeline.Stages[0].Type != StageTypeTextGen {
		t.Error("Expected text generation stage")
	}

	if pipeline.Stages[0].PreferredHardware != "npu" {
		t.Error("Expected NPU preference")
	}

	if pipeline.Stages[0].Model != "qwen2.5:0.5b" {
		t.Error("Expected qwen2.5:0.5b model")
	}

	// Check second stage
	if pipeline.Stages[1].ID != "select-best" {
		t.Error("Expected second stage 'select-best'")
	}

	if pipeline.Stages[1].Type != StageTypeTextGen {
		t.Error("Expected text generation stage")
	}

	if pipeline.Stages[1].PreferredHardware != "nvidia" {
		t.Error("Expected NVIDIA preference")
	}

	if pipeline.Stages[1].Model != "llama3:70b" {
		t.Error("Expected llama3:70b model")
	}

	if pipeline.Stages[1].InputTransform == nil {
		t.Error("Expected input transform for selection stage")
	}

	// Check options
	if !pipeline.Options.ParallelStages {
		t.Error("Expected parallel stages enabled")
	}

	if pipeline.Options.EnableStreaming {
		t.Error("Expected streaming disabled")
	}

	if !pipeline.Options.CollectMetrics {
		t.Error("Expected metrics collection")
	}
}

func TestBudgetAwarePipeline(t *testing.T) {
	pipeline := BudgetAwarePipeline()

	if pipeline == nil {
		t.Fatal("Expected non-nil pipeline")
	}

	if pipeline.ID != "budget-aware" {
		t.Errorf("Expected ID 'budget-aware', got %s", pipeline.ID)
	}

	if pipeline.Name != "Power Budget Aware" {
		t.Errorf("Expected name 'Power Budget Aware', got %s", pipeline.Name)
	}

	if len(pipeline.Stages) != 1 {
		t.Errorf("Expected 1 stage, got %d", len(pipeline.Stages))
	}

	stage := pipeline.Stages[0]
	if stage.ID != "generate-text" {
		t.Error("Expected stage 'generate-text'")
	}

	if stage.Type != StageTypeTextGen {
		t.Error("Expected text generation stage")
	}

	if stage.Model != "llama3:7b" {
		t.Error("Expected llama3:7b model")
	}

	if stage.ForwardingPolicy == nil {
		t.Fatal("Expected non-nil forwarding policy")
	}

	if !stage.ForwardingPolicy.EnableConfidenceCheck {
		t.Error("Expected confidence check")
	}

	if stage.ForwardingPolicy.MinConfidence != 0.75 {
		t.Errorf("Expected min confidence 0.75, got %f", stage.ForwardingPolicy.MinConfidence)
	}

	if len(stage.ForwardingPolicy.EscalationPath) != 2 {
		t.Errorf("Expected 2 backends in escalation path, got %d", len(stage.ForwardingPolicy.EscalationPath))
	}

	// Check options
	if !pipeline.Options.EnableStreaming {
		t.Error("Expected streaming enabled")
	}

	if !pipeline.Options.CollectMetrics {
		t.Error("Expected metrics collection")
	}
}

func TestAllExamplePipelinesAreValid(t *testing.T) {
	pipelines := []struct {
		name     string
		pipeline *Pipeline
	}{
		{"VoiceAssistant", VoiceAssistantPipeline()},
		{"AdaptiveText", AdaptiveTextPipeline()},
		{"CodeGeneration", CodeGenerationPipeline()},
		{"ThermalFailover", ThermalFailoverPipeline()},
		{"MultiModalRAG", MultiModalRAGPipeline()},
		{"SpeculativeExecution", SpeculativeExecutionPipeline()},
		{"BudgetAware", BudgetAwarePipeline()},
	}

	for _, p := range pipelines {
		if p.pipeline == nil {
			t.Errorf("Expected non-nil pipeline for %s", p.name)
			continue
		}

		if p.pipeline.ID == "" {
			t.Errorf("Expected non-empty ID for %s", p.name)
		}

		if p.pipeline.Name == "" {
			t.Errorf("Expected non-empty Name for %s", p.name)
		}

		if len(p.pipeline.Stages) == 0 {
			t.Errorf("Expected at least 1 stage for %s", p.name)
		}

		for i, stage := range p.pipeline.Stages {
			if stage.ID == "" {
				t.Errorf("Expected non-empty stage ID at index %d for %s", i, p.name)
			}

			if stage.Type == "" {
				t.Errorf("Expected non-empty stage type at index %d for %s", i, p.name)
			}

			// Model is optional for custom stages
			if stage.Type != StageTypeCustom && stage.Model == "" {
				t.Errorf("Expected non-empty model at index %d for %s (type %s)", i, p.name, stage.Type)
			}
		}

		if p.pipeline.Options == nil {
			t.Errorf("Expected non-nil options for %s", p.name)
		}
	}
}

func TestExamplePipelinesHaveDifferentIDs(t *testing.T) {
	pipelines := []struct {
		name string
		id   string
	}{
		{"VoiceAssistant", VoiceAssistantPipeline().ID},
		{"AdaptiveText", AdaptiveTextPipeline().ID},
		{"CodeGeneration", CodeGenerationPipeline().ID},
		{"ThermalFailover", ThermalFailoverPipeline().ID},
		{"MultiModalRAG", MultiModalRAGPipeline().ID},
		{"SpeculativeExecution", SpeculativeExecutionPipeline().ID},
		{"BudgetAware", BudgetAwarePipeline().ID},
	}

	seenIDs := make(map[string]string)
	for _, p := range pipelines {
		if prevName, exists := seenIDs[p.id]; exists {
			t.Errorf("Duplicate pipeline ID %s for %s and %s", p.id, prevName, p.name)
		}
		seenIDs[p.id] = p.name
	}
}

func TestVoiceAssistantStageSequence(t *testing.T) {
	pipeline := VoiceAssistantPipeline()

	// The stages should be in sequence: audio->text->LLM->text->audio
	expectedSequence := []struct {
		id   string
		typ  StageType
	}{
		{"voice-to-text", StageTypeAudioToText},
		{"process-text", StageTypeTextGen},
		{"text-to-voice", StageTypeTextToAudio},
	}

	for i, expected := range expectedSequence {
		if pipeline.Stages[i].ID != expected.id {
			t.Errorf("Expected stage %d ID %s, got %s", i, expected.id, pipeline.Stages[i].ID)
		}

		if pipeline.Stages[i].Type != expected.typ {
			t.Errorf("Expected stage %d type %s, got %s", i, expected.typ, pipeline.Stages[i].Type)
		}
	}
}

func TestCodeGenerationStageSequence(t *testing.T) {
	pipeline := CodeGenerationPipeline()

	// First stage should generate, second should review
	if pipeline.Stages[0].Model != "deepseek-coder:6.7b" {
		t.Error("Expected draft stage to use deepseek-coder")
	}

	if pipeline.Stages[1].Model != "llama3:70b" {
		t.Error("Expected review stage to use llama3:70b")
	}

	// Review stage should have input transform
	if pipeline.Stages[1].InputTransform == nil {
		t.Error("Expected review stage to have input transform")
	}
}

func TestRAGPipelineStageSequence(t *testing.T) {
	pipeline := MultiModalRAGPipeline()

	// Should be: embed -> retrieve -> generate
	stageTypes := []StageType{
		StageTypeEmbed,
		StageTypeCustom,
		StageTypeTextGen,
	}

	for i, expectedType := range stageTypes {
		if pipeline.Stages[i].Type != expectedType {
			t.Errorf("Expected stage %d type %s, got %s", i, expectedType, pipeline.Stages[i].Type)
		}
	}
}

func TestRetrieveContextHelper(t *testing.T) {
	embedding := []float64{0.1, 0.2, 0.3}
	context := retrieveContext(embedding)

	if context == "" {
		t.Error("Expected non-empty context")
	}

	if context != "Retrieved context from vector database" {
		t.Errorf("Expected specific context string, got %s", context)
	}
}

func TestExamplePipelinesContainValidForwarding(t *testing.T) {
	// Test that pipelines with forwarding policies have valid configurations
	pipeline := AdaptiveTextPipeline()

	stage := pipeline.Stages[0]
	if stage.ForwardingPolicy == nil {
		t.Fatal("Expected forwarding policy")
	}

	// Escalation path should have valid size
	if len(stage.ForwardingPolicy.EscalationPath) > stage.ForwardingPolicy.MaxRetries {
		t.Errorf("Escalation path (%d) should not exceed MaxRetries (%d)",
			len(stage.ForwardingPolicy.EscalationPath),
			stage.ForwardingPolicy.MaxRetries)
	}

	// MinConfidence should be reasonable
	if stage.ForwardingPolicy.MinConfidence < 0 || stage.ForwardingPolicy.MinConfidence > 1 {
		t.Errorf("MinConfidence should be between 0 and 1, got %f", stage.ForwardingPolicy.MinConfidence)
	}
}

func TestExamplePipelinesInputOutputTransforms(t *testing.T) {
	// Test RAG pipeline which has several input/output transforms
	pipeline := MultiModalRAGPipeline()

	// Retrieve stage should have input transform
	if pipeline.Stages[1].InputTransform == nil {
		t.Error("Expected retrieve stage to have input transform")
	}

	// Generate stage should have input transform
	if pipeline.Stages[2].InputTransform == nil {
		t.Error("Expected generate stage to have input transform")
	}

	// Speculative execution pipeline
	specPipeline := SpeculativeExecutionPipeline()

	// Select best stage should have input transform
	if specPipeline.Stages[1].InputTransform == nil {
		t.Error("Expected select-best stage to have input transform")
	}
}

func TestExamplePipelinesHardwarePreferences(t *testing.T) {
	// Test that examples use realistic hardware preferences
	validHardware := map[string]bool{
		"npu":    true,
		"igpu":   true,
		"nvidia": true,
		"cpu":    true,
		"":       true, // Not specified is also valid
	}

	pipelines := []struct {
		name string
		fn   func() *Pipeline
	}{
		{"VoiceAssistant", VoiceAssistantPipeline},
		{"AdaptiveText", AdaptiveTextPipeline},
		{"CodeGeneration", CodeGenerationPipeline},
		{"ThermalFailover", ThermalFailoverPipeline},
		{"MultiModalRAG", MultiModalRAGPipeline},
		{"SpeculativeExecution", SpeculativeExecutionPipeline},
		{"BudgetAware", BudgetAwarePipeline},
	}

	for _, p := range pipelines {
		pipeline := p.fn()
		for i, stage := range pipeline.Stages {
			if !validHardware[stage.PreferredHardware] {
				t.Errorf("%s stage %d has invalid hardware: %s", p.name, i, stage.PreferredHardware)
			}
		}
	}
}
