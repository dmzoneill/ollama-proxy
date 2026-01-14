package pipeline

import "fmt"

// Example pipeline configurations

// VoiceAssistantPipeline creates a voice → text → LLM → text → voice pipeline
func VoiceAssistantPipeline() *Pipeline {
	return &Pipeline{
		ID:          "voice-assistant",
		Name:        "Voice Assistant",
		Description: "Voice input → Speech-to-text (NPU) → LLM processing (iGPU/GPU) → Text-to-speech (NPU)",
		Stages: []*Stage{
			{
				ID:               "voice-to-text",
				Type:             StageTypeAudioToText,
				Description:      "Convert voice to text using NPU",
				PreferredHardware: "npu",
				Model:            "whisper-tiny", // Fast speech recognition on NPU
				ForwardingPolicy: &ForwardingPolicy{
					EnableConfidenceCheck: true,
					MinConfidence:         0.7,
					EscalationPath:       []string{"ollama-npu", "ollama-intel"},
				},
			},
			{
				ID:               "process-text",
				Type:             StageTypeTextGen,
				Description:      "Process text with LLM on iGPU or GPU",
				PreferredHardware: "igpu", // Prefer iGPU (balanced), fallback to GPU if needed
				Model:            "llama3:7b",
				ForwardingPolicy: &ForwardingPolicy{
					EnableConfidenceCheck: true,
					MinConfidence:         0.8,
					EscalationPath:       []string{"ollama-intel", "ollama-nvidia"},
					EnableThermalCheck:   true,
					MaxTemperature:       85.0,
				},
			},
			{
				ID:               "text-to-voice",
				Type:             StageTypeTextToAudio,
				Description:      "Convert response to voice using NPU",
				PreferredHardware: "npu",
				Model:            "piper-tts", // Fast TTS on NPU
				ForwardingPolicy: &ForwardingPolicy{
					EnableConfidenceCheck: false, // TTS is deterministic
				},
			},
		},
		Options: &PipelineOptions{
			EnableStreaming:  true,
			PreserveContext:  true,
			ContinueOnError:  false,
			CollectMetrics:   true,
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
				ID:               "generate-text",
				Type:             StageTypeTextGen,
				Description:      "Generate text with automatic quality escalation",
				PreferredHardware: "npu", // Start cheap
				Model:            "qwen2.5:0.5b",
				ForwardingPolicy: &ForwardingPolicy{
					EnableConfidenceCheck: true,
					MinConfidence:         0.75,
					MaxRetries:           3,
					EscalationPath: []string{
						"ollama-npu",    // Try NPU first (3W)
						"ollama-intel",  // Escalate to iGPU (12W)
						"ollama-nvidia", // Final escalation to GPU (55W)
					},
					EnableThermalCheck: true,
					MaxTemperature:     85.0,
				},
			},
		},
		Options: &PipelineOptions{
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
				ID:               "draft-code",
				Type:             StageTypeTextGen,
				Description:      "Generate initial code draft",
				PreferredHardware: "igpu",
				Model:            "deepseek-coder:6.7b",
				ForwardingPolicy: &ForwardingPolicy{
					EnableQualityCheck: true,
					QualityThreshold:   0.8,
				},
			},
			{
				ID:               "review-code",
				Type:             StageTypeTextGen,
				Description:      "Review and refine code if quality threshold not met",
				PreferredHardware: "nvidia",
				Model:            "llama3:70b",
				InputTransform: func(input interface{}) (interface{}, error) {
					code := input.(string)
					return "Review and improve this code:\n\n" + code, nil
				},
			},
		},
		Options: &PipelineOptions{
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
				ID:               "long-generation",
				Type:             StageTypeTextGen,
				Description:      "Generate long document with thermal monitoring",
				PreferredHardware: "nvidia", // Start with best quality
				Model:            "llama3:70b",
				ForwardingPolicy: &ForwardingPolicy{
					EnableThermalCheck: true,
					MaxTemperature:     87.0,
					MaxFanPercent:      85,
					EscalationPath: []string{
						"ollama-nvidia", // Start here
						"ollama-intel",  // Switch to iGPU if overheating
						"ollama-cpu",    // Final fallback to CPU
					},
				},
			},
		},
		Options: &PipelineOptions{
			EnableStreaming:  true,
			PreserveContext:  true, // Preserve generated tokens when switching
			ContinueOnError:  false,
			CollectMetrics:   true,
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
				ID:               "generate-embedding",
				Type:             StageTypeEmbed,
				Description:      "Generate query embedding on NPU",
				PreferredHardware: "npu",
				Model:            "nomic-embed-text",
			},
			{
				ID:               "retrieve-context",
				Type:             StageTypeCustom,
				Description:      "Retrieve relevant context from vector DB",
				InputTransform: func(input interface{}) (interface{}, error) {
					// TODO: Vector DB lookup
					embedding := input.([]float64)
					context := retrieveContext(embedding) // Placeholder
					return context, nil
				},
			},
			{
				ID:               "generate-answer",
				Type:             StageTypeTextGen,
				Description:      "Generate answer with retrieved context",
				PreferredHardware: "nvidia",
				Model:            "llama3:70b",
				InputTransform: func(input interface{}) (interface{}, error) {
					context := input.(string)
					return "Context:\n" + context + "\n\nQuestion: [user query]", nil
				},
			},
		},
		Options: &PipelineOptions{
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
				ID:               "generate-candidates",
				Type:             StageTypeTextGen,
				Description:      "Generate multiple solution candidates on NPU (parallel)",
				PreferredHardware: "npu",
				Model:            "qwen2.5:0.5b",
				// TODO: Support parallel execution of same stage N times
			},
			{
				ID:               "select-best",
				Type:             StageTypeTextGen,
				Description:      "Evaluate candidates and select best",
				PreferredHardware: "nvidia",
				Model:            "llama3:70b",
				InputTransform: func(input interface{}) (interface{}, error) {
					candidates := input.([]string)
					prompt := "Evaluate these solutions and select the best:\n\n"
					for i, candidate := range candidates {
						prompt += fmt.Sprintf("Solution %d:\n%s\n\n", i+1, candidate)
					}
					return prompt, nil
				},
			},
		},
		Options: &PipelineOptions{
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
				ID:               "generate-text",
				Type:             StageTypeTextGen,
				Description:      "Generate text within power budget",
				Model:            "llama3:7b",
				ForwardingPolicy: &ForwardingPolicy{
					EnableConfidenceCheck: true,
					MinConfidence:         0.75,
					EscalationPath: []string{
						"ollama-npu",   // 3W
						"ollama-intel", // 12W
						// Skip NVIDIA if on battery (55W too high)
					},
				},
			},
		},
		Options: &PipelineOptions{
			EnableStreaming: true,
			CollectMetrics:  true,
		},
	}
}

// Helper function placeholder
func retrieveContext(embedding []float64) string {
	// TODO: Implement vector DB lookup
	return "Retrieved context from vector database"
}
