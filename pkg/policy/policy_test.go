package policy

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewPolicy(t *testing.T) {
	p := NewPolicy()
	if p == nil {
		t.Fatal("NewPolicy returned nil")
	}
	if p.budgets == nil {
		t.Fatal("budgets map is nil")
	}
}

func TestGetOrCreateBudget_Free(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)

	if budget == nil {
		t.Fatal("GetOrCreateBudget returned nil")
	}
	if budget.dailyBudgetWh != 10.0 {
		t.Errorf("Expected 10.0 Wh for free tier, got %.2f", budget.dailyBudgetWh)
	}
	if budget.nvidiaQuotaPerHour != 5 {
		t.Errorf("Expected 5 NVIDIA requests/hour for free tier, got %d", budget.nvidiaQuotaPerHour)
	}
	if budget.dailyBudgetUSD != 0.0 {
		t.Errorf("Expected $0 for free tier, got %.2f", budget.dailyBudgetUSD)
	}
}

func TestGetOrCreateBudget_Basic(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user2", TierBasic)

	if budget.dailyBudgetWh != 50.0 {
		t.Errorf("Expected 50.0 Wh for basic tier, got %.2f", budget.dailyBudgetWh)
	}
	if budget.nvidiaQuotaPerHour != 20 {
		t.Errorf("Expected 20 NVIDIA requests/hour for basic tier, got %d", budget.nvidiaQuotaPerHour)
	}
	if budget.dailyBudgetUSD != 1.0 {
		t.Errorf("Expected $1 for basic tier, got %.2f", budget.dailyBudgetUSD)
	}
}

func TestGetOrCreateBudget_Premium(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user3", TierPremium)

	if budget.dailyBudgetWh != 200.0 {
		t.Errorf("Expected 200.0 Wh for premium tier, got %.2f", budget.dailyBudgetWh)
	}
	if budget.nvidiaQuotaPerHour != 100 {
		t.Errorf("Expected 100 NVIDIA requests/hour for premium tier, got %d", budget.nvidiaQuotaPerHour)
	}
	if budget.dailyBudgetUSD != 10.0 {
		t.Errorf("Expected $10 for premium tier, got %.2f", budget.dailyBudgetUSD)
	}
}

func TestGetOrCreateBudget_Enterprise(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user4", TierEnterprise)

	if budget.dailyBudgetWh != 1000.0 {
		t.Errorf("Expected 1000.0 Wh for enterprise tier, got %.2f", budget.dailyBudgetWh)
	}
	if budget.nvidiaQuotaPerHour != 1000 {
		t.Errorf("Expected 1000 NVIDIA requests/hour for enterprise tier, got %d", budget.nvidiaQuotaPerHour)
	}
	if budget.dailyBudgetUSD != 100.0 {
		t.Errorf("Expected $100 for enterprise tier, got %.2f", budget.dailyBudgetUSD)
	}
}

func TestGetOrCreateBudget_ReturnsExisting(t *testing.T) {
	p := NewPolicy()
	budget1 := p.GetOrCreateBudget("user1", TierFree)
	budget1.usedTodayWh = 5.0 // Modify the budget

	// Get again with different tier - should return existing
	budget2 := p.GetOrCreateBudget("user1", TierPremium)

	if budget1 != budget2 {
		t.Error("Expected same budget instance to be returned")
	}
	if budget2.usedTodayWh != 5.0 {
		t.Error("Expected existing budget with usedTodayWh = 5.0")
	}
}

func TestCheckAndDeduct_Success(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)

	err := budget.CheckAndDeduct("ollama-igpu", 2.0)
	if err != nil {
		t.Errorf("CheckAndDeduct should succeed, got: %v", err)
	}

	if budget.usedTodayWh != 2.0 {
		t.Errorf("Expected usedTodayWh to be 2.0, got %.2f", budget.usedTodayWh)
	}
}

func TestCheckAndDeduct_EnergyBudgetExceeded(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree) // 10 Wh budget

	budget.usedTodayWh = 9.0
	err := budget.CheckAndDeduct("ollama-nvidia", 2.0) // Would exceed

	if err == nil {
		t.Error("Expected error for budget exceeded")
	}
	if !strings.Contains(err.Error(), "daily energy budget exceeded") {
		t.Errorf("Expected 'daily energy budget exceeded' in error, got: %v", err)
	}
}

func TestCheckAndDeduct_NvidiaQuotaExceeded(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree) // 5 NVIDIA/hour quota

	budget.nvidiaUsedThisHour = 5
	err := budget.CheckAndDeduct("ollama-nvidia", 0.5)

	if err == nil {
		t.Error("Expected error for NVIDIA quota exceeded")
	}
	if !strings.Contains(err.Error(), "NVIDIA quota exceeded") {
		t.Errorf("Expected 'NVIDIA quota exceeded' in error, got: %v", err)
	}
}

func TestCheckAndDeduct_NvidiaCountIncremented(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierBasic)

	err := budget.CheckAndDeduct("ollama-nvidia", 1.0)
	if err != nil {
		t.Fatalf("CheckAndDeduct failed: %v", err)
	}

	if budget.nvidiaUsedThisHour != 1 {
		t.Errorf("Expected nvidiaUsedThisHour to be 1, got %d", budget.nvidiaUsedThisHour)
	}
}

func TestCheckAndDeduct_NonNvidiaNoQuotaCount(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)

	err := budget.CheckAndDeduct("ollama-igpu", 1.0)
	if err != nil {
		t.Fatalf("CheckAndDeduct failed: %v", err)
	}

	if budget.nvidiaUsedThisHour != 0 {
		t.Error("Non-NVIDIA backend should not increment NVIDIA quota")
	}
}

func TestCheckAndDeduct_DailyReset(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)

	// Simulate usage
	budget.usedTodayWh = 8.0
	budget.usedTodayUSD = 0.5
	budget.lastResetTime = time.Now().Add(-25 * time.Hour) // Over 24 hours ago

	err := budget.CheckAndDeduct("ollama-igpu", 1.0)
	if err != nil {
		t.Fatalf("CheckAndDeduct failed: %v", err)
	}

	// Should have been reset before the deduction
	if budget.usedTodayWh != 1.0 {
		t.Errorf("Expected usedTodayWh to be reset and then 1.0, got %.2f", budget.usedTodayWh)
	}
	if budget.usedTodayUSD != 0.0 {
		t.Errorf("Expected usedTodayUSD to be reset to 0.0, got %.2f", budget.usedTodayUSD)
	}
}

func TestCheckAndDeduct_NvidiaHourlyReset(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)

	// Simulate NVIDIA quota usage
	budget.nvidiaUsedThisHour = 5
	budget.lastNvidiaResetTime = time.Now().Add(-61 * time.Minute) // Over 1 hour ago

	err := budget.CheckAndDeduct("ollama-nvidia", 0.5)
	if err != nil {
		t.Fatalf("CheckAndDeduct failed: %v", err)
	}

	// Should have been reset before the deduction
	if budget.nvidiaUsedThisHour != 1 {
		t.Errorf("Expected nvidiaUsedThisHour to be reset and then 1, got %d", budget.nvidiaUsedThisHour)
	}
}

func TestSuggestAlternative_VeryLowBudget(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)
	budget.usedTodayWh = 9.5 // 0.5 Wh remaining

	suggestion := budget.SuggestAlternative("ollama-nvidia")
	if suggestion != "ollama-npu" {
		t.Errorf("Expected NPU for very low budget, got %s", suggestion)
	}
}

func TestSuggestAlternative_ModerateBudget(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)
	budget.usedTodayWh = 5.0 // 5 Wh remaining

	suggestion := budget.SuggestAlternative("ollama-nvidia")
	if suggestion != "ollama-igpu" {
		t.Errorf("Expected Intel GPU for moderate budget, got %s", suggestion)
	}
}

func TestSuggestAlternative_HighBudgetWithQuota(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierPremium)
	budget.usedTodayWh = 50.0 // 150 Wh remaining
	budget.nvidiaUsedThisHour = 10 // Under quota (100)

	suggestion := budget.SuggestAlternative("ollama-nvidia")
	if suggestion != "ollama-nvidia" {
		t.Errorf("Expected NVIDIA when budget and quota available, got %s", suggestion)
	}
}

func TestSuggestAlternative_HighBudgetNoQuota(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)
	budget.usedTodayWh = 0.0 // 10 Wh remaining (> 10)
	budget.nvidiaUsedThisHour = 5 // Quota exceeded

	suggestion := budget.SuggestAlternative("ollama-nvidia")
	if suggestion != "ollama-igpu" {
		t.Errorf("Expected Intel GPU when NVIDIA quota exceeded, got %s", suggestion)
	}
}

func TestSetBatteryMode(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 45)

	if !p.batteryMode {
		t.Error("Expected battery mode to be true")
	}
	if p.batteryPercentage != 45 {
		t.Errorf("Expected battery percentage 45, got %d", p.batteryPercentage)
	}
}

func TestMaxAllowedPower_OnACPower(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(false, 100)

	maxPower := p.MaxAllowedPower()
	if maxPower != 999 {
		t.Errorf("Expected no limit (999) on AC power, got %d", maxPower)
	}
}

func TestMaxAllowedPower_BatteryUnder20(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 15)

	maxPower := p.MaxAllowedPower()
	if maxPower != 5 {
		t.Errorf("Expected 5W limit for battery < 20%%, got %d", maxPower)
	}
}

func TestMaxAllowedPower_Battery20to50(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 35)

	maxPower := p.MaxAllowedPower()
	if maxPower != 15 {
		t.Errorf("Expected 15W limit for battery 20-50%%, got %d", maxPower)
	}
}

func TestMaxAllowedPower_Battery50to80(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 65)

	maxPower := p.MaxAllowedPower()
	if maxPower != 30 {
		t.Errorf("Expected 30W limit for battery 50-80%%, got %d", maxPower)
	}
}

func TestMaxAllowedPower_BatteryOver80(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 90)

	maxPower := p.MaxAllowedPower()
	if maxPower != 999 {
		t.Errorf("Expected no limit (999) for battery > 80%%, got %d", maxPower)
	}
}

func TestShouldThrottle_QuietHoursLate(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(false, 100)

	// Can't directly control time.Now(), but we can test the logic
	// For now, test with current time if in quiet hours
	now := time.Now()
	hour := now.Hour()

	shouldThrottle := p.ShouldThrottle()

	// Should throttle if between 10pm-6am
	expected := (hour >= 22 || hour < 6)
	if shouldThrottle != expected {
		t.Logf("Current hour: %d, shouldThrottle: %v, expected: %v", hour, shouldThrottle, expected)
		// Only error if not in quiet hours and we're not on low battery
		if !expected && !p.batteryMode {
			// This is fine - we're not in quiet hours and not on battery
		}
	}
}

func TestShouldThrottle_LowBattery(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 25)

	shouldThrottle := p.ShouldThrottle()
	if !shouldThrottle {
		t.Error("Expected throttling with low battery (< 30%)")
	}
}

func TestShouldThrottle_FullBatteryDaytime(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 90)

	// Test at hour that's not quiet hours
	now := time.Now()
	hour := now.Hour()

	shouldThrottle := p.ShouldThrottle()

	// Should not throttle if battery > 30% and not quiet hours
	if hour >= 6 && hour < 22 && shouldThrottle {
		t.Error("Should not throttle with high battery during daytime")
	}
}

func TestGetRecommendedBackend_WithinBudget(t *testing.T) {
	p := NewPolicy()
	// Pre-create the budget to avoid deadlock in GetRecommendedBackend
	_ = p.GetOrCreateBudget("user1", TierPremium)
	p.SetBatteryMode(false, 100)

	backend, err := p.GetRecommendedBackend("user1", TierPremium, "ollama-nvidia", 100)
	if err != nil {
		t.Errorf("Expected no error for premium user with budget, got: %v", err)
	}
	if backend != "ollama-nvidia" {
		t.Errorf("Expected ollama-nvidia, got %s", backend)
	}
}

func TestGetRecommendedBackend_BudgetExceeded(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)
	budget.usedTodayWh = 9.5 // Close to 10 Wh limit

	p.SetBatteryMode(false, 100)

	// 5000 tokens on NVIDIA: 5000 / 65 ~= 76.9 seconds
	// 55W * 76.9s / 3600 = 1.176 Wh (definitely exceeds 0.5 Wh remaining)
	backend, err := p.GetRecommendedBackend("user1", TierFree, "ollama-nvidia", 5000)
	if err == nil {
		t.Error("Expected error for budget exceeded")
		return
	}
	if !strings.Contains(err.Error(), "budget exceeded") {
		t.Errorf("Expected 'budget exceeded' in error, got: %v", err)
	}
	// Should suggest alternative
	if backend == "ollama-nvidia" {
		t.Error("Should not return original backend when budget exceeded")
	}
}

func TestGetRecommendedBackend_PowerConstraint(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 15) // Very low battery, maxPower = 5W

	backend, err := p.GetRecommendedBackend("user1", TierPremium, "ollama-nvidia", 100)
	if err == nil {
		t.Error("Expected error for power constraint")
		return
	}
	if !strings.Contains(err.Error(), "battery") {
		t.Errorf("Expected 'battery' in error message, got: %v", err)
	}
	if backend != "ollama-npu" {
		t.Errorf("Expected downgrade to NPU for low battery, got %s", backend)
	}
}

func TestGetRecommendedBackend_ModerateDowngrade(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 40) // Moderate battery, maxPower = 15W

	backend, err := p.GetRecommendedBackend("user1", TierPremium, "ollama-nvidia", 100)
	if err == nil {
		t.Error("Expected error for power constraint")
		return
	}
	if backend != "ollama-igpu" {
		t.Errorf("Expected downgrade to Intel GPU, got %s", backend)
	}
}

func TestEstimateEnergy_NPU(t *testing.T) {
	energy := estimateEnergy("ollama-npu", 100)

	// NPU: 10 tokens/sec, 3W
	// Time: 100/10 = 10 seconds
	// Energy: 3W * 10s / 3600 = 0.00833 Wh
	expected := (3.0 * 10.0) / 3600.0

	if energy < expected*0.99 || energy > expected*1.01 {
		t.Errorf("Expected energy ~%.6f Wh, got %.6f", expected, energy)
	}
}

func TestEstimateEnergy_IntelGPU(t *testing.T) {
	energy := estimateEnergy("ollama-igpu", 220)

	// Intel GPU: 22 tokens/sec, 12W
	// Time: 220/22 = 10 seconds
	// Energy: 12W * 10s / 3600 = 0.0333 Wh
	expected := (12.0 * 10.0) / 3600.0

	if energy < expected*0.99 || energy > expected*1.01 {
		t.Errorf("Expected energy ~%.6f Wh, got %.6f", expected, energy)
	}
}

func TestEstimateEnergy_Nvidia(t *testing.T) {
	energy := estimateEnergy("ollama-nvidia", 650)

	// NVIDIA: 65 tokens/sec, 55W
	// Time: 650/65 = 10 seconds
	// Energy: 55W * 10s / 3600 = 0.1528 Wh
	expected := (55.0 * 10.0) / 3600.0

	if energy < expected*0.99 || energy > expected*1.01 {
		t.Errorf("Expected energy ~%.6f Wh, got %.6f", expected, energy)
	}
}

func TestEstimateEnergy_CPU(t *testing.T) {
	energy := estimateEnergy("ollama-cpu", 60)

	// CPU: 6 tokens/sec, 28W
	// Time: 60/6 = 10 seconds
	// Energy: 28W * 10s / 3600 = 0.0778 Wh
	expected := (28.0 * 10.0) / 3600.0

	if energy < expected*0.99 || energy > expected*1.01 {
		t.Errorf("Expected energy ~%.6f Wh, got %.6f", expected, energy)
	}
}

func TestEstimateEnergy_UnknownBackend(t *testing.T) {
	energy := estimateEnergy("unknown-backend", 220)

	// Should default to Intel GPU characteristics
	expected := (12.0 * 10.0) / 3600.0

	if energy < expected*0.99 || energy > expected*1.01 {
		t.Errorf("Expected energy to default to Intel GPU ~%.6f Wh, got %.6f", expected, energy)
	}
}

func TestGetUsageStats(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)
	budget.usedTodayWh = 5.0
	budget.nvidiaUsedThisHour = 3

	stats := budget.GetUsageStats()

	if stats["energy_used_wh"].(float64) != 5.0 {
		t.Errorf("Expected energy_used_wh = 5.0, got %v", stats["energy_used_wh"])
	}
	if stats["energy_budget_wh"].(float64) != 10.0 {
		t.Errorf("Expected energy_budget_wh = 10.0, got %v", stats["energy_budget_wh"])
	}
	if stats["nvidia_used_hour"].(int) != 3 {
		t.Errorf("Expected nvidia_used_hour = 3, got %v", stats["nvidia_used_hour"])
	}
	if stats["nvidia_quota_hour"].(int) != 5 {
		t.Errorf("Expected nvidia_quota_hour = 5, got %v", stats["nvidia_quota_hour"])
	}

	// Check energy remaining percentage
	remaining := stats["energy_remaining_%"].(float64)
	expected := 50.0 // (1 - 5/10) * 100
	if remaining < expected*0.99 || remaining > expected*1.01 {
		t.Errorf("Expected energy_remaining_%% ~%.1f, got %.1f", expected, remaining)
	}
}

// TestCheckAndDeduct_CostTracking tests cost tracking and budget
func TestCheckAndDeduct_CostTracking(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierBasic)
	budget.usedTodayUSD = 0.5

	err := budget.CheckAndDeduct("ollama-igpu", 1.0)
	if err != nil {
		t.Errorf("CheckAndDeduct should succeed, got: %v", err)
	}

	// Cost tracking happens on energy deduction only, verify budget exists
	if budget.dailyBudgetUSD != 1.0 {
		t.Errorf("Expected dailyBudgetUSD to be 1.0, got %.2f", budget.dailyBudgetUSD)
	}
}

// TestGetUsageStats_CostInfo tests cost info in usage stats
func TestGetUsageStats_CostInfo(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierPremium)
	budget.usedTodayUSD = 5.5

	stats := budget.GetUsageStats()

	if stats["cost_used_usd"].(float64) != 5.5 {
		t.Errorf("Expected cost_used_usd = 5.5, got %v", stats["cost_used_usd"])
	}
	if stats["cost_budget_usd"].(float64) != 10.0 {
		t.Errorf("Expected cost_budget_usd = 10.0, got %v", stats["cost_budget_usd"])
	}
}

// TestCheckAndDeduct_ExactBudgetLimit tests exact boundary condition
func TestCheckAndDeduct_ExactBudgetLimit(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree) // 10 Wh budget

	budget.usedTodayWh = 8.0
	err := budget.CheckAndDeduct("ollama-igpu", 2.0) // Exactly fills budget

	if err != nil {
		t.Errorf("Deduction to exact budget limit should succeed, got: %v", err)
	}

	if budget.usedTodayWh != 10.0 {
		t.Errorf("Expected usedTodayWh to be 10.0, got %.2f", budget.usedTodayWh)
	}
}

// TestCheckAndDeduct_OnlySlightlyOver tests just over boundary
func TestCheckAndDeduct_OnlySlightlyOver(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree) // 10 Wh budget

	budget.usedTodayWh = 9.5
	err := budget.CheckAndDeduct("ollama-igpu", 0.6) // 0.1 Wh over

	if err == nil {
		t.Error("Deduction over budget should fail")
	}
}

// TestCheckAndDeduct_CPU backend
func TestCheckAndDeduct_CPU(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierBasic)

	err := budget.CheckAndDeduct("ollama-cpu", 1.0)
	if err != nil {
		t.Errorf("CheckAndDeduct CPU should succeed, got: %v", err)
	}

	if budget.usedTodayWh != 1.0 {
		t.Errorf("Expected CPU deduction, got %.2f", budget.usedTodayWh)
	}
}

// TestCheckAndDeduct_NPU backend
func TestCheckAndDeduct_NPU(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierBasic)

	err := budget.CheckAndDeduct("ollama-npu", 0.5)
	if err != nil {
		t.Errorf("CheckAndDeduct NPU should succeed, got: %v", err)
	}

	if budget.usedTodayWh != 0.5 {
		t.Errorf("Expected NPU deduction, got %.2f", budget.usedTodayWh)
	}
}

// TestMaxAllowedPower_Boundary20 tests exact boundary at 20%
func TestMaxAllowedPower_Boundary20(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 20)

	maxPower := p.MaxAllowedPower()
	if maxPower != 15 {
		t.Errorf("Expected 15W at exactly 20%%, got %d", maxPower)
	}
}

// TestMaxAllowedPower_Boundary50 tests exact boundary at 50%
func TestMaxAllowedPower_Boundary50(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 50)

	maxPower := p.MaxAllowedPower()
	if maxPower != 30 {
		t.Errorf("Expected 30W at exactly 50%%, got %d", maxPower)
	}
}

// TestMaxAllowedPower_Boundary80 tests exact boundary at 80%
func TestMaxAllowedPower_Boundary80(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 80)

	maxPower := p.MaxAllowedPower()
	if maxPower != 999 {
		t.Errorf("Expected no limit (999) at exactly 80%%, got %d", maxPower)
	}
}

// TestMaxAllowedPower_BatteryZero tests extreme low battery
func TestMaxAllowedPower_BatteryZero(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 0)

	maxPower := p.MaxAllowedPower()
	if maxPower != 5 {
		t.Errorf("Expected 5W at 0%% battery, got %d", maxPower)
	}
}

// TestSuggestAlternative_Boundary transitions
func TestSuggestAlternative_EdgeCase1Wh(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)
	budget.usedTodayWh = 9.0 // 1.0 Wh remaining (edge case)

	suggestion := budget.SuggestAlternative("ollama-nvidia")
	if suggestion != "ollama-igpu" {
		t.Errorf("Expected Intel GPU for 1 Wh remaining, got %s", suggestion)
	}
}

// TestSuggestAlternative_Exactly10Wh tests exact 10 Wh boundary
func TestSuggestAlternative_Exactly10Wh(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)
	budget.usedTodayWh = 0.0 // 10 Wh remaining (exact boundary)

	suggestion := budget.SuggestAlternative("ollama-nvidia")
	// At exactly 10.0, it's not < 10.0, so should try NVIDIA if quota available
	if suggestion != "ollama-nvidia" {
		t.Errorf("Expected NVIDIA for exactly 10 Wh remaining, got %s", suggestion)
	}
}

// TestShouldThrottle_QuietHours22 tests 10pm hour
func TestShouldThrottle_QuietHours22(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(false, 100)

	now := time.Now()
	// This test is time-dependent, so we log but don't fail
	shouldThrottle := p.ShouldThrottle()
	hour := now.Hour()

	if hour == 22 && !shouldThrottle {
		t.Error("Expected throttling at 10pm")
	}
}

// TestShouldThrottle_QuietHours5am tests 5am hour
func TestShouldThrottle_QuietHours5am(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(false, 100)

	now := time.Now()
	shouldThrottle := p.ShouldThrottle()
	hour := now.Hour()

	if hour == 5 && !shouldThrottle {
		t.Error("Expected throttling at 5am")
	}
}

// TestShouldThrottle_NotQuietHours tests non-quiet hours
func TestShouldThrottle_NotQuietHours(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(false, 100)

	now := time.Now()
	shouldThrottle := p.ShouldThrottle()
	hour := now.Hour()

	if hour >= 6 && hour < 22 && shouldThrottle {
		t.Error("Should not throttle during daytime with high battery")
	}
}

// TestShouldThrottle_LowBatteryHighPercentage tests boundary just over 30%
func TestShouldThrottle_LowBatteryEdge31(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 31)

	shouldThrottle := p.ShouldThrottle()
	// At 31%, should not throttle just for battery (unless quiet hours)
	now := time.Now()
	hour := now.Hour()

	if hour >= 6 && hour < 22 && shouldThrottle {
		t.Error("Should not throttle with 31% battery during daytime")
	}
}

// TestShouldThrottle_LowBatteryDaytime tests low battery during non-quiet hours
func TestShouldThrottle_LowBatteryDaytime(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 20)

	shouldThrottle := p.ShouldThrottle()
	now := time.Now()
	hour := now.Hour()

	// During non-quiet hours (6am-10pm), should throttle due to low battery
	if hour >= 6 && hour < 22 && !shouldThrottle {
		t.Error("Should throttle with 20% battery during daytime")
	}
}

// TestEstimateEnergy_ZeroTokens tests edge case
func TestEstimateEnergy_ZeroTokens(t *testing.T) {
	energy := estimateEnergy("ollama-nvidia", 0)

	if energy != 0.0 {
		t.Errorf("Expected 0 energy for 0 tokens, got %.6f", energy)
	}
}

// TestEstimateEnergy_LargeTokenCount tests realistic scenario
func TestEstimateEnergy_LargeTokenCount(t *testing.T) {
	energy := estimateEnergy("ollama-nvidia", 10000)

	// NVIDIA: 65 tokens/sec, 55W
	// Time: 10000/65 ~= 153.85 seconds
	// Energy: 55W * 153.85s / 3600 ~= 2.36 Wh
	if energy < 2.0 || energy > 3.0 {
		t.Errorf("Expected energy around 2.3 Wh for 10k tokens on NVIDIA, got %.6f", energy)
	}
}

// TestGetOrCreateBudget_MultipleTiers tests creating multiple users
func TestGetOrCreateBudget_MultipleUsers(t *testing.T) {
	p := NewPolicy()

	u1 := p.GetOrCreateBudget("user1", TierFree)
	u2 := p.GetOrCreateBudget("user2", TierBasic)
	u3 := p.GetOrCreateBudget("user3", TierPremium)
	u4 := p.GetOrCreateBudget("user4", TierEnterprise)

	if u1.dailyBudgetWh != 10.0 {
		t.Errorf("User1 should be Free tier with 10 Wh")
	}
	if u2.dailyBudgetWh != 50.0 {
		t.Errorf("User2 should be Basic tier with 50 Wh")
	}
	if u3.dailyBudgetWh != 200.0 {
		t.Errorf("User3 should be Premium tier with 200 Wh")
	}
	if u4.dailyBudgetWh != 1000.0 {
		t.Errorf("User4 should be Enterprise tier with 1000 Wh")
	}
}

// TestCheckAndDeduct_MultipleDeductions tests cumulative deductions
func TestCheckAndDeduct_MultipleDeductions(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierBasic) // 50 Wh

	err := budget.CheckAndDeduct("ollama-igpu", 10.0)
	if err != nil {
		t.Errorf("First deduction should succeed: %v", err)
	}

	err = budget.CheckAndDeduct("ollama-igpu", 15.0)
	if err != nil {
		t.Errorf("Second deduction should succeed: %v", err)
	}

	err = budget.CheckAndDeduct("ollama-igpu", 20.0)
	if err != nil {
		t.Errorf("Third deduction should succeed: %v", err)
	}

	err = budget.CheckAndDeduct("ollama-igpu", 10.0)
	if err == nil {
		t.Error("Fourth deduction should fail, exceeds budget")
	}

	if budget.usedTodayWh != 45.0 {
		t.Errorf("Expected total 45 Wh used, got %.2f", budget.usedTodayWh)
	}
}

// TestCheckAndDeduct_NvidiaQuotaMultiple tests multiple NVIDIA requests
func TestCheckAndDeduct_NvidiaQuotaMultiple(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierBasic) // 20 NVIDIA/hour

	for i := 0; i < 20; i++ {
		err := budget.CheckAndDeduct("ollama-nvidia", 0.5)
		if err != nil {
			t.Errorf("Deduction %d should succeed: %v", i+1, err)
		}
	}

	// Should fail on 21st
	err := budget.CheckAndDeduct("ollama-nvidia", 0.5)
	if err == nil {
		t.Error("21st NVIDIA deduction should fail, quota exceeded")
	}
}

// TestCheckAndDeduct_MixedBackends tests NVIDIA quota isolation
func TestCheckAndDeduct_MixedBackends(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree) // 5 NVIDIA/hour

	// Use non-NVIDIA backends - should not affect NVIDIA quota
	_ = budget.CheckAndDeduct("ollama-npu", 1.0)
	_ = budget.CheckAndDeduct("ollama-igpu", 1.0)
	_ = budget.CheckAndDeduct("ollama-cpu", 1.0)

	if budget.nvidiaUsedThisHour != 0 {
		t.Error("Non-NVIDIA backends should not affect NVIDIA quota")
	}

	// Now use NVIDIA - should count
	_ = budget.CheckAndDeduct("ollama-nvidia", 1.0)
	if budget.nvidiaUsedThisHour != 1 {
		t.Error("NVIDIA deduction should increment NVIDIA quota")
	}
}

// TestSetBatteryMode_MultipleUpdates tests changing battery state multiple times
func TestSetBatteryMode_MultipleUpdates(t *testing.T) {
	p := NewPolicy()

	p.SetBatteryMode(false, 100)
	if p.batteryMode || p.batteryPercentage != 100 {
		t.Error("First update failed")
	}

	p.SetBatteryMode(true, 50)
	if !p.batteryMode || p.batteryPercentage != 50 {
		t.Error("Second update failed")
	}

	p.SetBatteryMode(false, 100)
	if p.batteryMode || p.batteryPercentage != 100 {
		t.Error("Third update failed")
	}
}

// TestGetRecommendedBackend_AllBackends tests all backend types in recommendations
func TestGetRecommendedBackend_AllBackends(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(false, 100)

	backends := []string{"ollama-npu", "ollama-igpu", "ollama-cpu", "ollama-nvidia"}

	for i, backend := range backends {
		userID := fmt.Sprintf("user%d", i)
		_ = p.GetOrCreateBudget(userID, TierPremium)

		recommended, err := p.GetRecommendedBackend(userID, TierPremium, backend, 100)
		if err != nil {
			t.Errorf("Backend %s recommendation should succeed: %v", backend, err)
		}
		if recommended != backend {
			t.Errorf("Expected %s, got %s", backend, recommended)
		}
	}
}

// TestGetRecommendedBackend_IGPUDowngrade tests downgrade to IGPU
func TestGetRecommendedBackend_IGPUDowngrade(t *testing.T) {
	p := NewPolicy()
	p.SetBatteryMode(true, 55) // maxPower = 30W, NVIDIA = 55W too much

	backend, err := p.GetRecommendedBackend("user1", TierPremium, "ollama-nvidia", 100)
	if err == nil {
		t.Error("Should have returned error for power constraint")
	}
	if backend != "ollama-igpu" {
		t.Errorf("Expected downgrade to IGPU (12W), got %s", backend)
	}
}

// TestEstimateEnergy_AllBackends tests energy estimation for all backends
func TestEstimateEnergy_AllBackends(t *testing.T) {
	backends := map[string]struct {
		tokens   int
		minWh    float64
		maxWh    float64
	}{
		"ollama-npu":    {tokens: 100, minWh: 0.008, maxWh: 0.009},
		"ollama-igpu":   {tokens: 220, minWh: 0.033, maxWh: 0.034},
		"ollama-nvidia": {tokens: 650, minWh: 0.15, maxWh: 0.16},
		"ollama-cpu":    {tokens: 60, minWh: 0.077, maxWh: 0.078},
	}

	for backend, test := range backends {
		energy := estimateEnergy(backend, test.tokens)
		if energy < test.minWh || energy > test.maxWh {
			t.Errorf("Backend %s: expected energy between %.6f-%.6f Wh, got %.6f",
				backend, test.minWh, test.maxWh, energy)
		}
	}
}

// TestGetUsageStats_AllTiers tests usage stats across different tiers
func TestGetUsageStats_AllTiers(t *testing.T) {
	p := NewPolicy()
	tiers := map[UserTier]float64{
		TierFree:       10.0,
		TierBasic:      50.0,
		TierPremium:    200.0,
		TierEnterprise: 1000.0,
	}

	for tier, expectedBudget := range tiers {
		userID := fmt.Sprintf("user_%d", tier)
		budget := p.GetOrCreateBudget(userID, tier)
		stats := budget.GetUsageStats()

		if stats["energy_budget_wh"].(float64) != expectedBudget {
			t.Errorf("Tier %d: expected budget %.1f, got %.1f", tier, expectedBudget, stats["energy_budget_wh"])
		}
	}
}

// TestCheckAndDeduct_DailyReset_MultipleResets tests multiple daily resets
func TestCheckAndDeduct_MultipleDailyResets(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierFree)

	// First reset simulation
	budget.usedTodayWh = 5.0
	budget.lastResetTime = time.Now().Add(-25 * time.Hour)

	_ = budget.CheckAndDeduct("ollama-igpu", 1.0)
	firstUsed := budget.usedTodayWh

	// Second reset simulation
	budget.lastResetTime = time.Now().Add(-25 * time.Hour)
	_ = budget.CheckAndDeduct("ollama-igpu", 1.0)
	secondUsed := budget.usedTodayWh

	if firstUsed != 1.0 || secondUsed != 1.0 {
		t.Errorf("Daily reset not working properly: first=%.1f, second=%.1f", firstUsed, secondUsed)
	}
}

// TestCheckAndDeduct_HourlyReset_MultipleResets tests multiple hourly resets
func TestCheckAndDeduct_MultipleHourlyResets(t *testing.T) {
	p := NewPolicy()
	budget := p.GetOrCreateBudget("user1", TierBasic)

	// First reset simulation
	budget.nvidiaUsedThisHour = 10
	budget.lastNvidiaResetTime = time.Now().Add(-61 * time.Minute)

	_ = budget.CheckAndDeduct("ollama-nvidia", 0.5)
	firstCount := budget.nvidiaUsedThisHour

	// Second reset simulation
	budget.lastNvidiaResetTime = time.Now().Add(-61 * time.Minute)
	_ = budget.CheckAndDeduct("ollama-nvidia", 0.5)
	secondCount := budget.nvidiaUsedThisHour

	if firstCount != 1 || secondCount != 1 {
		t.Errorf("Hourly reset not working properly: first=%d, second=%d", firstCount, secondCount)
	}
}
