// Package pipeline runs multi-stage model pipelines.
package pipeline

import "fmt"

// Example pipeline configurations

// VoiceAssistantPipeline creates a voice → text → LLM → text → voice pipeline
func VoiceAssistantPipeline() *Pipeline {
	return &Pipeline{
		ID:          strVoiceAssistant2,
		Name:        "Voice Assistant",
		Description: "Voice input → Speech-to-text (NPU) → LLM processing (iGPU/GPU) → Text-to-speech (NPU)",
		Stages: []*Stage{
			{
				ID:                strVoiceToText,
				Type:              StageTypeAudioToText,
				Description:       "Convert voice to text using NPU",
				PreferredHardware: strNpu,
				Model:             strWhisperTiny, // Fast speech recognition on NPU
				ForwardingPolicy: &ForwardingPolicy{
					EnableConfidenceCheck: true,
					MinConfidence:         0.7,
					EscalationPath:        []string{strOllamaNpu, strOllamaIntel},
				},
			},
			{
				ID:                strProcessText,
				Type:              StageTypeTextGen,
				Description:       "Process text with LLM on iGPU or GPU",
				PreferredHardware: strIgpu, // Prefer iGPU (balanced), fallback to GPU if needed
				Model:             strLlama37b,
				ForwardingPolicy: &ForwardingPolicy{
					EnableConfidenceCheck: true,
					MinConfidence:         0.8,
					EscalationPath:        []string{strOllamaIntel, strOllamaNvidia},
					EnableThermalCheck:    true,
					MaxTemperature:        85.0,
				},
			},
			{
				ID:                strTextToVoice,
				Type:              StageTypeTextToAudio,
				Description:       "Convert response to voice using NPU",
				PreferredHardware: strNpu,
				Model:             strPiperTts, // Fast TTS on NPU
				ForwardingPolicy: &ForwardingPolicy{
					EnableConfidenceCheck: false, // TTS is deterministic
				},
			},
		},
		Options: &Options{
			EnableStreaming: true,
			PreserveContext: true,
			ContinueOnError: false,
			CollectMetrics:  true,
		},
	}
}

// AdaptiveTextPipeline creates a confidence-based escalation pipeline
func AdaptiveTextPipeline() *Pipeline {
	return &Pipeline{
		ID:          "adaptive-text",
		Name:        "Adaptive Text Generation",
		Description: "Start with NPU, escalate to iGPU or GPU based on quality",
		Stages: []*Stage{
			{
				ID:                strGenerateText,
				Type:              StageTypeTextGen,
				Description:       "Generate text with automatic quality escalation",
				PreferredHardware: strNpu, // Start cheap
				Model:             strQwen2505b,
				ForwardingPolicy: &ForwardingPolicy{
					EnableConfidenceCheck: true,
					MinConfidence:         0.75,
					MaxRetries:            3,
					EscalationPath: []string{
						strOllamaNpu,    // Try NPU first (3W)
						strOllamaIntel,  // Escalate to iGPU (12W)
						strOllamaNvidia, // Final escalation to GPU (55W)
					},
					EnableThermalCheck: true,
					MaxTemperature:     85.0,
				},
			},
		},
		Options: &Options{
			EnableStreaming: false,
			CollectMetrics:  true,
		},
	}
}

// CodeGenerationPipeline creates a pipeline optimized for code generation
func CodeGenerationPipeline() *Pipeline {
	return &Pipeline{
		ID:          "code-generation",
		Name:        "Code Generation with Review",
		Description: "Generate code draft on iGPU, review/refine on GPU if needed",
		Stages: []*Stage{
			{
				ID:                "draft-code",
				Type:              StageTypeTextGen,
				Description:       "Generate initial code draft",
				PreferredHardware: strIgpu,
				Model:             strDeepseekCoder67b,
				ForwardingPolicy: &ForwardingPolicy{
					EnableQualityCheck: true,
					QualityThreshold:   0.8,
				},
			},
			{
				ID:                "review-code",
				Type:              StageTypeTextGen,
				Description:       "Review and refine code if quality threshold not met",
				PreferredHardware: strNvidia,
				Model:             strLlama370b,
				InputTransform: func(input interface{}) (interface{}, error) {
					code, ok := input.(string)
					if !ok {
						return nil, fmt.Errorf("expected string input")
					}
					return "Review and improve this code:\n\n" + code, nil
				},
			},
		},
		Options: &Options{
			EnableStreaming: false,
			PreserveContext: true,
			CollectMetrics:  true,
		},
	}
}

// ThermalFailoverPipeline creates a pipeline with thermal protection
func ThermalFailoverPipeline() *Pipeline {
	return &Pipeline{
		ID:          "thermal-failover",
		Name:        "Long Generation with Thermal Protection",
		Description: "Long document generation that switches backends if overheating",
		Stages: []*Stage{
			{
				ID:                "long-generation",
				Type:              StageTypeTextGen,
				Description:       "Generate long document with thermal monitoring",
				PreferredHardware: strNvidia, // Start with best quality
				Model:             strLlama370b,
				ForwardingPolicy: &ForwardingPolicy{
					EnableThermalCheck: true,
					MaxTemperature:     87.0,
					MaxFanPercent:      85,
					EscalationPath: []string{
						strOllamaNvidia, // Start here
						strOllamaIntel,  // Switch to iGPU if overheating
						"ollama-cpu",    // Final fallback to CPU
					},
				},
			},
		},
		Options: &Options{
			EnableStreaming: true,
			PreserveContext: true, // Preserve generated tokens when switching
			ContinueOnError: false,
			CollectMetrics:  true,
		},
	}
}

// MultiModalRAGPipeline creates a pipeline for RAG with embeddings
func MultiModalRAGPipeline() *Pipeline {
	return &Pipeline{
		ID:          "rag-pipeline",
		Name:        "RAG with Embeddings",
		Description: "Generate embeddings (NPU) → Retrieve context → Generate answer (GPU)",
		Stages: []*Stage{
			{
				ID:                "generate-embedding",
				Type:              StageTypeEmbed,
				Description:       "Generate query embedding on NPU",
				PreferredHardware: strNpu,
				Model:             strNomicEmbedText,
			},
			{
				ID:          "retrieve-context",
				Type:        StageTypeCustom,
				Description: "Retrieve relevant context from vector DB",
				InputTransform: func(input interface{}) (interface{}, error) {
					// TODO: Vector DB lookup
					embedding, ok := input.([]float64)
					if !ok {
						return nil, fmt.Errorf("expected []float64 input")
					}
					context := retrieveContext(embedding) // Placeholder
					return context, nil
				},
			},
			{
				ID:                "generate-answer",
				Type:              StageTypeTextGen,
				Description:       "Generate answer with retrieved context",
				PreferredHardware: strNvidia,
				Model:             strLlama370b,
				InputTransform: func(input interface{}) (interface{}, error) {
					context, ok := input.(string)
					if !ok {
						return nil, fmt.Errorf("expected string input")
					}
					return "Context:\n" + context + "\n\nQuestion: [user query]", nil
				},
			},
		},
		Options: &Options{
			EnableStreaming: true,
			PreserveContext: true,
			CollectMetrics:  true,
		},
	}
}

// SpeculativeExecutionPipeline creates a pipeline for speculative execution
func SpeculativeExecutionPipeline() *Pipeline {
	return &Pipeline{
		ID:          "speculative-execution",
		Name:        "Speculative Execution",
		Description: "NPU generates multiple candidates, GPU picks best",
		Stages: []*Stage{
			{
				ID:                "generate-candidates",
				Type:              StageTypeTextGen,
				Description:       "Generate multiple solution candidates on NPU (parallel)",
				PreferredHardware: strNpu,
				Model:             strQwen2505b,
				// TODO: Support parallel execution of same stage N times
			},
			{
				ID:                "select-best",
				Type:              StageTypeTextGen,
				Description:       "Evaluate candidates and select best",
				PreferredHardware: strNvidia,
				Model:             strLlama370b,
				InputTransform: func(input interface{}) (interface{}, error) {
					candidates, ok := input.([]string)
					if !ok {
						return nil, fmt.Errorf("expected []string candidates, got %T", input)
					}
					prompt := "Evaluate these solutions and select the best:\n\n"
					for i, candidate := range candidates {
						prompt += fmt.Sprintf("Solution %d:\n%s\n\n", i+1, candidate)
					}
					return prompt, nil
				},
			},
		},
		Options: &Options{
			ParallelStages:  true, // Enable parallel stage execution
			EnableStreaming: false,
			CollectMetrics:  true,
		},
	}
}

// BudgetAwarePipeline creates a pipeline that respects power budgets
func BudgetAwarePipeline() *Pipeline {
	return &Pipeline{
		ID:          "budget-aware",
		Name:        "Power Budget Aware",
		Description: "Adapt backend selection based on power budget",
		Stages: []*Stage{
			{
				ID:          strGenerateText,
				Type:        StageTypeTextGen,
				Description: "Generate text within power budget",
				Model:       strLlama37b,
				ForwardingPolicy: &ForwardingPolicy{
					EnableConfidenceCheck: true,
					MinConfidence:         0.75,
					EscalationPath: []string{
						strOllamaNpu,   // 3W
						strOllamaIntel, // 12W
						// Skip NVIDIA if on battery (55W too high)
					},
				},
			},
		},
		Options: &Options{
			EnableStreaming: true,
			CollectMetrics:  true,
		},
	}
}

// Helper function placeholder
func retrieveContext(_ []float64) string {
	// TODO: Implement vector DB lookup
	return "Retrieved context from vector database"
}
