package openai

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/daoneill/ollama-proxy/pkg/backends"
	"github.com/daoneill/ollama-proxy/pkg/router"
	"github.com/daoneill/ollama-proxy/pkg/safeconv"
)

// ParseRoutingHeaders extracts routing annotations from HTTP request headers
func ParseRoutingHeaders(r *http.Request) *backends.Annotations {
	annotations := &backends.Annotations{
		Custom: make(map[string]string),
	}

	// X-Target-Backend: Explicit backend selection (e.g., "ollama-nvidia", "ollama-npu")
	if target := r.Header.Get("X-Target-Backend"); target != "" {
		annotations.Target = target
	}

	// X-Latency-Critical: Route to fastest backend (true/false)
	if latency := r.Header.Get("X-Latency-Critical"); latency != "" {
		annotations.LatencyCritical = parseBool(latency)
	}

	// X-Power-Efficient: Route to lowest power backend (true/false)
	if power := r.Header.Get("X-Power-Efficient"); power != "" {
		annotations.PreferPowerEfficiency = parseBool(power)
	}

	// X-Max-Latency-Ms: Maximum acceptable latency constraint (integer)
	if maxLatency := r.Header.Get("X-Max-Latency-Ms"); maxLatency != "" {
		if val, err := strconv.Atoi(maxLatency); err == nil {
			annotations.MaxLatencyMs = safeconv.Int32FromInt(val)
		}
	}

	// X-Max-Power-Watts: Maximum power budget constraint (integer)
	if maxPower := r.Header.Get("X-Max-Power-Watts"); maxPower != "" {
		if val, err := strconv.Atoi(maxPower); err == nil {
			annotations.MaxPowerWatts = safeconv.Int32FromInt(val)
		}
	}

	// X-Cache-Enabled: Enable response caching (true/false)
	if cache := r.Header.Get("X-Cache-Enabled"); cache != "" {
		annotations.CacheEnabled = parseBool(cache)
	}

	// X-Media-Type: Media type hint for routing (text, code, image, audio, realtime, auto)
	if mediaType := r.Header.Get("X-Media-Type"); mediaType != "" {
		annotations.MediaType = mediaTypeFromHeader(mediaType)
	}
	annotations.Priority = priorityFromHeaders(r.Header.Get("X-Priority"), annotations)

	applyTrackingHeaders(r, annotations)
	return annotations
}

func applyTrackingHeaders(r *http.Request, annotations *backends.Annotations) {
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		annotations.RequestID = requestID
	}
	if deadline := r.Header.Get("X-Deadline-Ms"); deadline != "" {
		if val, err := strconv.ParseInt(deadline, 10, 64); err == nil {
			annotations.DeadlineMs = val
		}
	}
	for key, values := range r.Header {
		if strings.HasPrefix(key, "X-Custom-") && len(values) > 0 {
			annotations.Custom[strings.TrimPrefix(key, "X-Custom-")] = values[0]
		}
	}
}

// WriteRoutingHeaders writes routing metadata to HTTP response headers
func WriteRoutingHeaders(w http.ResponseWriter, decision *router.RoutingDecision) {
	if decision == nil {
		return
	}

	// X-Backend-Used: Which backend processed the request
	if decision.Backend != nil {
		w.Header().Set("X-Backend-Used", decision.Backend.ID())
	}

	// X-Routing-Reason: Why this backend was selected
	if decision.Reason != "" {
		w.Header().Set("X-Routing-Reason", decision.Reason)
	}

	// X-Estimated-Power-Watts: Estimated power consumption
	if decision.EstimatedPowerW > 0 {
		w.Header().Set("X-Estimated-Power-Watts", fmt.Sprintf("%.1f", decision.EstimatedPowerW))
	}

	// X-Estimated-Latency-Ms: Estimated latency
	if decision.EstimatedLatencyMs > 0 {
		w.Header().Set("X-Estimated-Latency-Ms", fmt.Sprintf("%d", decision.EstimatedLatencyMs))
	}

	// X-Alternatives: Alternative backends that could handle this request
	if len(decision.Alternatives) > 0 {
		w.Header().Set("X-Alternatives", strings.Join(decision.Alternatives, ","))
	}
}

func mediaTypeFromHeader(mediaType string) backends.MediaType {
	switch strings.ToLower(mediaType) {
	case "text":
		return backends.MediaTypeText
	case "code":
		return backends.MediaTypeCode
	case "image":
		return backends.MediaTypeImage
	case "audio":
		return backends.MediaTypeAudio
	case strRealtime:
		return backends.MediaTypeRealtime
	case "auto":
		return backends.MediaTypeAuto
	default:
		return ""
	}
}

func priorityFromHeaders(priority string, annotations *backends.Annotations) backends.Priority {
	switch strings.ToLower(priority) {
	case "":
		if annotations.LatencyCritical || annotations.MediaType == backends.MediaTypeRealtime {
			return backends.PriorityCritical
		}
		if annotations.MediaType == backends.MediaTypeAudio {
			return backends.PriorityHigh
		}
		return backends.PriorityNormal
	case "best-effort", "low":
		return backends.PriorityBestEffort
	case "normal":
		return backends.PriorityNormal
	case "high":
		return backends.PriorityHigh
	case "critical", strRealtime:
		return backends.PriorityCritical
	default:
		return 0
	}
}

// parseBool converts string to bool, accepting various formats
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}
