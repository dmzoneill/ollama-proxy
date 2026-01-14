package server

import (
	"context"
	"fmt"
	"time"

	pb "github.com/daoneill/ollama-proxy/api/gen/go"
	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/logging"
	"github.com/daoneill/ollama-proxy/pkg/pipeline"
	"github.com/daoneill/ollama-proxy/pkg/router"
	"go.uber.org/zap"
)

// ComputeServer implements the gRPC ComputeService
type ComputeServer struct {
	pb.UnimplementedComputeServiceServer
	router           *router.Router
	forwardingRouter *router.ForwardingRouter
	pipelineExecutor *pipeline.PipelineExecutor
	pipelineLoader   *pipeline.PipelineLoader
}

// NewComputeServer creates a new gRPC server
func NewComputeServer(r *router.Router) *ComputeServer {
	return &ComputeServer{
		router: r,
	}
}

// SetForwardingRouter sets the forwarding router (optional)
func (s *ComputeServer) SetForwardingRouter(fr *router.ForwardingRouter) {
	s.forwardingRouter = fr
}

// SetPipelineExecutor sets the pipeline executor (optional)
func (s *ComputeServer) SetPipelineExecutor(pe *pipeline.PipelineExecutor, pl *pipeline.PipelineLoader) {
	s.pipelineExecutor = pe
	s.pipelineLoader = pl
}

// Generate performs text generation with intelligent routing
func (s *ComputeServer) Generate(ctx context.Context, req *pb.GenerateRequest) (*pb.GenerateResponse, error) {
	logging.Logger.Info("Generate request received",
		zap.String("prompt", truncate(req.Prompt, 50)),
		zap.String("model", req.Model),
		zap.String("target", req.Annotations.GetTarget()),
	)

	start := time.Now()

	// Convert annotations
	annotations := convertAnnotations(req.Annotations)

	// Use forwarding router if available
	if s.forwardingRouter != nil {
		logging.Logger.Info("Using confidence-based forwarding")

		forwardingResult, err := s.forwardingRouter.GenerateWithForwarding(
			ctx,
			req.Prompt,
			req.Model,
			annotations,
		)

		if err != nil {
			logging.Logger.Error("Forwarding failed", zap.Error(err))
			return nil, fmt.Errorf("forwarding failed: %w", err)
		}

		// Log forwarding details
		if forwardingResult.Forwarded {
			logging.Logger.Info("Request forwarded",
				zap.Int("total_attempts", forwardingResult.TotalAttempts),
				zap.String("final_backend", forwardingResult.FinalBackend.ID()),
				zap.Float64("confidence", forwardingResult.FinalConfidence.Overall),
			)
		} else {
			logging.Logger.Info("No forwarding needed",
				zap.String("backend", forwardingResult.FinalBackend.ID()),
				zap.Float64("confidence", forwardingResult.FinalConfidence.Overall),
			)
		}

		// Build response with forwarding metadata
		resp := &pb.GenerateResponse{
			Response:    forwardingResult.FinalResponse,
			BackendUsed: forwardingResult.FinalBackend.ID(),
			FromCache:   false,
			Routing: &pb.RoutingMetadata{
				Backend:             forwardingResult.FinalBackend.ID(),
				Reason:              fmt.Sprintf("Confidence: %.2f", forwardingResult.FinalConfidence.Overall),
				EstimatedPowerWatts: float32(forwardingResult.FinalBackend.PowerWatts()),
			},
			Stats: &pb.GenerationStats{
				TotalTimeMs: forwardingResult.TotalLatencyMs,
			},
			// TODO: Add forwarding metadata to protobuf
		}

		elapsed := time.Since(start)
		logging.Logger.Info("Generate completed",
			zap.Duration("elapsed", elapsed),
			zap.String("backend", forwardingResult.FinalBackend.ID()),
		)

		return resp, nil
	}

	// Fallback to standard routing (no forwarding)
	logging.Logger.Info("Using standard routing", zap.String("reason", "forwarding disabled"))

	decision, err := s.router.RouteRequest(ctx, annotations)
	if err != nil {
		logging.Logger.Error("Routing failed", zap.Error(err))
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	logging.Logger.Info("Request routed",
		zap.String("backend", decision.Backend.ID()),
		zap.String("reason", decision.Reason),
	)

	// Build backend request
	backendReq := &backends.GenerateRequest{
		Prompt:  req.Prompt,
		Model:   req.Model,
		Options: convertGenerationOptions(req.Options),
	}

	// Execute on backend
	backendResp, err := decision.Backend.Generate(ctx, backendReq)
	if err != nil {
		logging.Logger.Error("Backend generation failed",
			zap.String("backend", decision.Backend.ID()),
			zap.Error(err),
		)

		// Try fallback
		if fallbackDecision, fallbackErr := s.router.FallbackRequest(ctx, []string{decision.Backend.ID()}, annotations); fallbackErr == nil {
			logging.Logger.Info("Falling back to alternative backend",
				zap.String("fallback_backend", fallbackDecision.Backend.ID()),
			)
			backendResp, err = fallbackDecision.Backend.Generate(ctx, backendReq)
			if err == nil {
				decision = fallbackDecision
			}
		}

		if err != nil {
			return nil, fmt.Errorf("generation failed: %w", err)
		}
	}

	// Build response
	resp := &pb.GenerateResponse{
		Response:    backendResp.Response,
		BackendUsed: decision.Backend.ID(),
		FromCache:   false, // TODO: implement caching
		Routing: &pb.RoutingMetadata{
			Backend:             decision.Backend.ID(),
			Reason:              decision.Reason,
			EstimatedPowerWatts: float32(decision.EstimatedPowerW),
			EstimatedLatencyMs:  decision.EstimatedLatencyMs,
			Alternatives:        decision.Alternatives,
		},
		Stats: convertStats(backendResp.Stats),
	}

	elapsed := time.Since(start)
	logging.Logger.Info("Generate completed",
		zap.Duration("elapsed", elapsed),
		zap.String("backend", decision.Backend.ID()),
		zap.Float64("tokens_per_second", float64(backendResp.Stats.TokensPerSecond)),
	)

	return resp, nil
}

// GenerateStream performs streaming text generation
func (s *ComputeServer) GenerateStream(req *pb.GenerateRequest, stream pb.ComputeService_GenerateStreamServer) error {
	logging.Logger.Info("GenerateStream request received",
		zap.String("prompt", truncate(req.Prompt, 50)),
		zap.String("model", req.Model),
		zap.String("target", req.Annotations.GetTarget()),
	)

	// Convert annotations
	annotations := convertAnnotations(req.Annotations)

	// Route request
	decision, err := s.router.RouteRequest(stream.Context(), annotations)
	if err != nil {
		logging.Logger.Error("GenerateStream routing failed", zap.Error(err))
		return fmt.Errorf("routing failed: %w", err)
	}

	logging.Logger.Info("GenerateStream routed",
		zap.String("backend", decision.Backend.ID()),
		zap.String("reason", decision.Reason),
	)

	// Build backend request
	backendReq := &backends.GenerateRequest{
		Prompt:  req.Prompt,
		Model:   req.Model,
		Options: convertGenerationOptions(req.Options),
	}

	// Start streaming from backend
	reader, err := decision.Backend.GenerateStream(stream.Context(), backendReq)
	if err != nil {
		logging.Logger.Error("GenerateStream backend failed",
			zap.String("backend", decision.Backend.ID()),
			zap.Error(err),
		)
		return fmt.Errorf("streaming failed: %w", err)
	}
	defer reader.Close()

	// Send first message with backend info
	firstChunk := true

	// Stream chunks
	for {
		chunk, err := reader.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}

		resp := &pb.GenerateStreamResponse{
			Token: chunk.Token,
			Done:  chunk.Done,
		}

		if firstChunk {
			resp.BackendUsed = decision.Backend.ID()
			firstChunk = false
		}

		if chunk.Done && chunk.Stats != nil {
			resp.Stats = convertStats(chunk.Stats)
			logging.Logger.Info("GenerateStream completed",
				zap.String("backend", decision.Backend.ID()),
				zap.Float64("tokens_per_second", float64(chunk.Stats.TokensPerSecond)),
			)
		}

		if err := stream.Send(resp); err != nil {
			return err
		}

		if chunk.Done {
			break
		}
	}

	return nil
}

// Embed generates embeddings
func (s *ComputeServer) Embed(ctx context.Context, req *pb.EmbedRequest) (*pb.EmbedResponse, error) {
	logging.Logger.Info("Embed request received",
		zap.String("text", truncate(req.Text, 50)),
		zap.String("model", req.Model),
	)

	annotations := convertAnnotations(req.Annotations)

	decision, err := s.router.RouteRequest(ctx, annotations)
	if err != nil {
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	backendReq := &backends.EmbedRequest{
		Text:  req.Text,
		Model: req.Model,
	}

	backendResp, err := decision.Backend.Embed(ctx, backendReq)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	return &pb.EmbedResponse{
		Embedding: backendResp.Embedding,
		BackendUsed: decision.Backend.ID(),
		Routing: &pb.RoutingMetadata{
			Backend:               decision.Backend.ID(),
			Reason:                decision.Reason,
			EstimatedPowerWatts:   float32(decision.EstimatedPowerW),
			EstimatedLatencyMs:    decision.EstimatedLatencyMs,
			Alternatives:          decision.Alternatives,
		},
	}, nil
}

// ListBackends returns available backends and their status
func (s *ComputeServer) ListBackends(ctx context.Context, req *pb.ListBackendsRequest) (*pb.ListBackendsResponse, error) {
	backends := s.router.ListBackends()

	pbBackends := make([]*pb.BackendInfo, 0, len(backends))
	for _, backend := range backends {
		models, _ := backend.ListModels(ctx)
		metrics := backend.GetMetrics()

		pbBackends = append(pbBackends, &pb.BackendInfo{
			Id:       backend.ID(),
			Type:     backend.Type(),
			Name:     backend.Name(),
			Hardware: backend.Hardware(),
			Status: &pb.BackendStatus{
				State:   healthState(backend.IsHealthy()),
				Message: healthMessage(backend.IsHealthy()),
			},
			Capabilities: &pb.BackendCapabilities{
				Generate: backend.SupportsGenerate(),
				Embed:    backend.SupportsEmbed(),
				Stream:   backend.SupportsStream(),
				Models:   models,
			},
			Metrics: &pb.BackendMetrics{
				AvgLatencyMs:      metrics.AvgLatencyMs,
				ErrorRate:         metrics.ErrorRate,
				PowerWatts:        float32(backend.PowerWatts()),
				LoadedModels:      metrics.LoadedModels,
			},
		})
	}

	return &pb.ListBackendsResponse{
		Backends: pbBackends,
	}, nil
}

// HealthCheck checks proxy and backend health
func (s *ComputeServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	backendHealth := s.router.HealthCheckAll(ctx)

	overallHealthy := true
	backendHealthMap := make(map[string]string)

	for id, healthy := range backendHealth {
		status := "healthy"
		if !healthy {
			status = "unhealthy"
			overallHealthy = false
		}
		backendHealthMap[id] = status
	}

	overallStatus := "healthy"
	if !overallHealthy {
		overallStatus = "degraded"
	}

	return &pb.HealthCheckResponse{
		Status:        overallStatus,
		BackendHealth: backendHealthMap,
		TimestampUnix: time.Now().Unix(),
	}, nil
}

// Helper functions

func convertAnnotations(pb *pb.JobAnnotations) *backends.Annotations {
	if pb == nil {
		return &backends.Annotations{}
	}

	return &backends.Annotations{
		Target:                pb.Target,
		LatencyCritical:       pb.LatencyCritical,
		PreferPowerEfficiency: pb.PreferPowerEfficiency,
		CacheEnabled:          pb.CacheEnabled,
		MaxLatencyMs:          pb.MaxLatencyMs,
		MaxPowerWatts:         pb.MaxPowerWatts,
		Custom:                pb.Custom,
	}
}

func convertGenerationOptions(pb *pb.GenerationOptions) *backends.GenerationOptions {
	if pb == nil {
		return nil
	}

	return &backends.GenerationOptions{
		MaxTokens:     pb.MaxTokens,
		Temperature:   pb.Temperature,
		TopP:          pb.TopP,
		TopK:          pb.TopK,
		Stop:          pb.Stop,
		ContextLength: pb.ContextLength,
	}
}

func convertStats(stats *backends.GenerationStats) *pb.GenerationStats {
	if stats == nil {
		return nil
	}

	return &pb.GenerationStats{
		TimeToFirstTokenMs: stats.TimeToFirstTokenMs,
		TotalTimeMs:        stats.TotalTimeMs,
		TokensGenerated:    stats.TokensGenerated,
		TokensPerSecond:    stats.TokensPerSecond,
		EnergyWh:           stats.EnergyWh,
	}
}

func healthState(healthy bool) string {
	if healthy {
		return "healthy"
	}
	return "unhealthy"
}

func healthMessage(healthy bool) string {
	if healthy {
		return "Backend is responding normally"
	}
	return "Backend is not responding"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// ExecutePipeline executes a multi-stage processing pipeline
func (s *ComputeServer) ExecutePipeline(ctx context.Context, req *pb.ExecutePipelineRequest) (*pb.ExecutePipelineResponse, error) {
	logging.Logger.Info("ExecutePipeline started",
		zap.String("pipeline_id", req.PipelineId),
	)

	if s.pipelineExecutor == nil || s.pipelineLoader == nil {
		return nil, fmt.Errorf("pipelines not enabled")
	}

	// Load pipeline
	pipelineDef, err := s.pipelineLoader.GetPipeline(req.PipelineId)
	if err != nil {
		return nil, fmt.Errorf("pipeline not found: %w", err)
	}

	// Execute pipeline
	result, err := s.pipelineExecutor.Execute(ctx, pipelineDef, req.Input)
	if err != nil {
		return &pb.ExecutePipelineResponse{
			PipelineId: req.PipelineId,
			Success:    false,
			Error:      err.Error(),
		}, nil
	}

	// Convert result to protobuf
	stageResults := make([]*pb.StageResult, 0, len(result.StageResults))
	for _, stageResult := range result.StageResults {
		pbStageResult := &pb.StageResult{
			StageId: stageResult.StageID,
			Backend: stageResult.Backend,
			Success: stageResult.Success,
			Error:   "",
		}

		if stageResult.Error != nil {
			pbStageResult.Error = stageResult.Error.Error()
		}

		// Convert output to string
		if stageResult.Output != nil {
			if str, ok := stageResult.Output.(string); ok {
				pbStageResult.Output = str
			} else {
				pbStageResult.Output = fmt.Sprintf("%v", stageResult.Output)
			}
		}

		// Convert metadata
		if stageResult.Metadata != nil {
			pbStageResult.Metadata = &pb.StageMetadata{
				StartTimeUnix: stageResult.Metadata.StartTime.Unix(),
				EndTimeUnix:   stageResult.Metadata.EndTime.Unix(),
				DurationMs:    stageResult.Metadata.DurationMs,
				Model:         stageResult.Metadata.Model,
				TokensIn:      stageResult.Metadata.TokensIn,
				TokensOut:     stageResult.Metadata.TokensOut,
				Confidence:    float32(stageResult.Metadata.Confidence),
				Temperature:   float32(stageResult.Metadata.Temperature),
				FanSpeed:      int32(stageResult.Metadata.FanSpeed),
				Forwarded:     stageResult.Metadata.Forwarded,
				ForwardReason: stageResult.Metadata.ForwardReason,
				AttemptCount:  int32(stageResult.Metadata.AttemptCount),
			}
		}

		stageResults = append(stageResults, pbStageResult)
	}

	// Convert final output
	finalOutput := make(map[string]string)
	if result.FinalOutput != nil {
		if outputMap, ok := result.FinalOutput.(map[string]interface{}); ok {
			for k, v := range outputMap {
				finalOutput[k] = fmt.Sprintf("%v", v)
			}
		} else if outputStr, ok := result.FinalOutput.(string); ok {
			finalOutput["text"] = outputStr
		} else {
			finalOutput["result"] = fmt.Sprintf("%v", result.FinalOutput)
		}
	}

	response := &pb.ExecutePipelineResponse{
		PipelineId:   req.PipelineId,
		Success:      result.Success,
		FinalOutput:  finalOutput,
		StageResults: stageResults,
		TotalTimeMs:  int32(result.TotalTimeMs),
		TotalEnergyWh: float32(result.TotalEnergyWh),
		Error:        "",
	}

	if result.Error != nil {
		response.Error = result.Error.Error()
	}

	logging.Logger.Info("ExecutePipeline completed",
		zap.String("pipeline_id", req.PipelineId),
		zap.Bool("success", result.Success),
		zap.Int("stage_count", len(result.StageResults)),
		zap.Int64("total_time_ms", result.TotalTimeMs),
	)

	return response, nil
}

// ExecutePipelineStream executes a pipeline with streaming output
func (s *ComputeServer) ExecutePipelineStream(req *pb.ExecutePipelineRequest, stream pb.ComputeService_ExecutePipelineStreamServer) error {
	logging.Logger.Info("ExecutePipelineStream started",
		zap.String("pipeline_id", req.PipelineId),
	)

	if s.pipelineExecutor == nil || s.pipelineLoader == nil {
		return fmt.Errorf("pipelines not enabled")
	}

	// For now, execute non-streaming and send results
	// TODO: Implement true streaming execution
	result, err := s.ExecutePipeline(stream.Context(), req)
	if err != nil {
		return err
	}

	// Send stage results as they complete
	for _, stageResult := range result.StageResults {
		streamResp := &pb.PipelineStreamResponse{
			StageId:      stageResult.StageId,
			Done:         false,
			StageResult:  stageResult,
		}

		if err := stream.Send(streamResp); err != nil {
			return err
		}
	}

	// Send final result
	finalResp := &pb.PipelineStreamResponse{
		Done:        true,
		FinalResult: result,
	}

	return stream.Send(finalResp)
}
