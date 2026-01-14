package policy

import (
	"fmt"
	"sync"
	"time"
)

// UserTier defines user access level
type UserTier int

const (
	TierFree UserTier = iota
	TierBasic
	TierPremium
	TierEnterprise
)

// PowerBudget tracks power/compute usage
type PowerBudget struct {
	mu sync.RWMutex

	// Energy budget (Watt-hours per day)
	dailyBudgetWh    float64
	usedTodayWh      float64
	lastResetTime    time.Time

	// Request quotas per backend
	nvidiaQuotaPerHour   int
	nvidiaUsedThisHour   int
	lastNvidiaResetTime  time.Time

	// Cost tracking (if using cloud backends)
	dailyBudgetUSD float64
	usedTodayUSD   float64
}

// Policy enforces usage policies
type Policy struct {
	mu sync.RWMutex

	// User budgets
	budgets map[string]*PowerBudget // userID -> budget

	// System-wide limits
	batteryMode       bool
	batteryPercentage int

	// Time-based policies
	quietHours      bool // 10pm-6am
	peakHours       bool // 9am-5pm weekdays
}

func NewPolicy() *Policy {
	return &Policy{
		budgets: make(map[string]*PowerBudget),
	}
}

// GetOrCreateBudget gets user budget, creating if needed
func (p *Policy) GetOrCreateBudget(userID string, tier UserTier) *PowerBudget {
	p.mu.Lock()
	defer p.mu.Unlock()

	if budget, exists := p.budgets[userID]; exists {
		return budget
	}

	// Create budget based on tier
	budget := &PowerBudget{
		lastResetTime:       time.Now(),
		lastNvidiaResetTime: time.Now(),
	}

	switch tier {
	case TierFree:
		budget.dailyBudgetWh = 10.0         // ~10 Wh/day (333 NPU requests or 18 NVIDIA)
		budget.nvidiaQuotaPerHour = 5       // 5 NVIDIA requests/hour
		budget.dailyBudgetUSD = 0.0
	case TierBasic:
		budget.dailyBudgetWh = 50.0         // 50 Wh/day
		budget.nvidiaQuotaPerHour = 20
		budget.dailyBudgetUSD = 1.0
	case TierPremium:
		budget.dailyBudgetWh = 200.0        // 200 Wh/day
		budget.nvidiaQuotaPerHour = 100
		budget.dailyBudgetUSD = 10.0
	case TierEnterprise:
		budget.dailyBudgetWh = 1000.0       // Unlimited effectively
		budget.nvidiaQuotaPerHour = 1000
		budget.dailyBudgetUSD = 100.0
	}

	p.budgets[userID] = budget
	return budget
}

// CheckAndDeduct checks if request is allowed and deducts from budget
func (b *PowerBudget) CheckAndDeduct(backendID string, estimatedEnergyWh float64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Reset daily budget if needed
	if time.Since(b.lastResetTime) > 24*time.Hour {
		b.usedTodayWh = 0
		b.usedTodayUSD = 0
		b.lastResetTime = time.Now()
	}

	// Reset NVIDIA quota if needed
	if time.Since(b.lastNvidiaResetTime) > time.Hour {
		b.nvidiaUsedThisHour = 0
		b.lastNvidiaResetTime = time.Now()
	}

	// Check NVIDIA quota
	if backendID == "ollama-nvidia" {
		if b.nvidiaUsedThisHour >= b.nvidiaQuotaPerHour {
			return fmt.Errorf("NVIDIA quota exceeded (%d/%d per hour). Try Intel GPU or NPU, or wait %v",
				b.nvidiaUsedThisHour, b.nvidiaQuotaPerHour,
				time.Until(b.lastNvidiaResetTime.Add(time.Hour)))
		}
	}

	// Check energy budget
	if b.usedTodayWh+estimatedEnergyWh > b.dailyBudgetWh {
		return fmt.Errorf("daily energy budget exceeded (%.2f/%.2f Wh used, need %.2f Wh). Try lower-power backend",
			b.usedTodayWh, b.dailyBudgetWh, estimatedEnergyWh)
	}

	// Deduct from quotas
	if backendID == "ollama-nvidia" {
		b.nvidiaUsedThisHour++
	}
	b.usedTodayWh += estimatedEnergyWh

	return nil
}

// SuggestAlternative suggests cheaper alternative if quota exceeded
func (b *PowerBudget) SuggestAlternative(requestedBackend string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	remaining := b.dailyBudgetWh - b.usedTodayWh

	// If very low budget remaining, suggest NPU
	if remaining < 1.0 {
		return "ollama-npu"
	}

	// If moderate budget, suggest Intel GPU
	if remaining < 10.0 {
		return "ollama-igpu"
	}

	// Otherwise allow NVIDIA if quota available
	if b.nvidiaUsedThisHour < b.nvidiaQuotaPerHour {
		return "ollama-nvidia"
	}

	return "ollama-igpu"
}

// SetBatteryMode sets system battery state
func (p *Policy) SetBatteryMode(onBattery bool, percentage int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.batteryMode = onBattery
	p.batteryPercentage = percentage
}

// MaxAllowedPower returns max power based on system state
func (p *Policy) MaxAllowedPower() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.batteryMode {
		return 999 // No limit on AC power
	}

	// Battery-based limits
	switch {
	case p.batteryPercentage < 20:
		return 5 // NPU only (3W)
	case p.batteryPercentage < 50:
		return 15 // NPU or Intel GPU (12W)
	case p.batteryPercentage < 80:
		return 30 // Allow NVIDIA if needed
	default:
		return 999 // Full battery, allow anything
	}
}

// ShouldThrottle checks if we should throttle high-power backends
func (p *Policy) ShouldThrottle() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()
	hour := now.Hour()

	// Quiet hours (10pm - 6am): prefer silent, efficient backends
	if hour >= 22 || hour < 6 {
		return true
	}

	// Low battery: throttle
	if p.batteryMode && p.batteryPercentage < 30 {
		return true
	}

	return false
}

// GetRecommendedBackend applies all policies
func (p *Policy) GetRecommendedBackend(
	userID string,
	tier UserTier,
	requestedBackend string,
	estimatedTokens int,
) (string, error) {
	// Get or create budget (needs write lock)
	budget := p.GetOrCreateBudget(userID, tier)

	// Estimate energy based on backend and token count
	estimatedEnergyWh := estimateEnergy(requestedBackend, estimatedTokens)

	// Check budget
	if err := budget.CheckAndDeduct(requestedBackend, estimatedEnergyWh); err != nil {
		// Budget exceeded, suggest alternative
		alternative := budget.SuggestAlternative(requestedBackend)
		return alternative, fmt.Errorf("budget exceeded, using %s instead: %w", alternative, err)
	}

	// Check system state (needs read lock)
	maxPower := p.MaxAllowedPower()

	// Map backends to power
	backendPower := map[string]int{
		"ollama-npu":    3,
		"ollama-igpu":   12,
		"ollama-nvidia": 55,
		"ollama-cpu":    28,
	}

	if backendPower[requestedBackend] > maxPower {
		// Downgrade to fit power budget
		if maxPower >= 12 {
			return "ollama-igpu", fmt.Errorf("battery low, downgraded from %s to igpu", requestedBackend)
		}
		return "ollama-npu", fmt.Errorf("battery critical, downgraded from %s to npu", requestedBackend)
	}

	return requestedBackend, nil
}

// estimateEnergy estimates energy consumption for a request
func estimateEnergy(backend string, estimatedTokens int) float64 {
	// Token generation rates and power consumption
	type backendChar struct {
		tokensPerSec float64
		powerWatts   float64
	}

	characteristics := map[string]backendChar{
		"ollama-npu":    {tokensPerSec: 10, powerWatts: 3},
		"ollama-igpu":   {tokensPerSec: 22, powerWatts: 12},
		"ollama-nvidia": {tokensPerSec: 65, powerWatts: 55},
		"ollama-cpu":    {tokensPerSec: 6, powerWatts: 28},
	}

	char := characteristics[backend]
	if char.tokensPerSec == 0 {
		char = characteristics["ollama-igpu"] // Default
	}

	// Calculate time needed
	timeSeconds := float64(estimatedTokens) / char.tokensPerSec

	// Energy = Power × Time (in Wh)
	energyWh := (char.powerWatts * timeSeconds) / 3600.0

	return energyWh
}

// GetUsageStats returns current usage statistics
func (b *PowerBudget) GetUsageStats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return map[string]interface{}{
		"energy_used_wh":     b.usedTodayWh,
		"energy_budget_wh":   b.dailyBudgetWh,
		"energy_remaining_%": (1 - b.usedTodayWh/b.dailyBudgetWh) * 100,
		"nvidia_used_hour":   b.nvidiaUsedThisHour,
		"nvidia_quota_hour":  b.nvidiaQuotaPerHour,
		"cost_used_usd":      b.usedTodayUSD,
		"cost_budget_usd":    b.dailyBudgetUSD,
	}
}
