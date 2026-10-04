package study

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"ai-tutor/internal/db"
	llmpkg "ai-tutor/internal/llm"
	"ai-tutor/internal/models"
)

type dummyLLM struct {
	model  string
	limits llmpkg.ModelLimits
}

func (d *dummyLLM) GenerateAnswer(prompt string) (string, error) {
	return "dummy answer", nil
}

func (d *dummyLLM) ModelName() string {
	return d.model
}

func (d *dummyLLM) GetLimits() llmpkg.ModelLimits {
	if d.limits.MaxInputTokens == 0 {
		return llmpkg.ModelLimits{MaxInputTokens: 4000, MaxOutputTokens: 2500}
	}
	return d.limits
}

func TestFastTierCooldownRouting(t *testing.T) {
	fast := &dummyLLM{model: "gpt-4.1-mini"}
	heavy := &dummyLLM{model: "gemini-2.5-flash"}

	svc := &StudyService{
		fastLLMProvider:  fast,
		heavyLLMProvider: heavy,
	}

	// 1. Initial state: routes to fast
	provider, tier := svc.selectLLM("sample context")
	if tier != "fast" || provider.ModelName() != "gpt-4.1-mini" {
		t.Fatalf("expected initial routing to fast tier (gpt-4.1-mini), got tier=%s model=%s", tier, provider.ModelName())
	}

	// 2. Mark fast rate-limited
	svc.MarkFastRateLimited(45 * time.Second)
	if !svc.IsFastRateLimited() {
		t.Fatalf("expected IsFastRateLimited() to be true")
	}

	// 3. Routing during cooldown: escalates to heavy tier
	provider, tier = svc.selectLLM("sample context")
	if tier != "heavy" || provider.ModelName() != "gemini-2.5-flash" {
		t.Fatalf("expected routing to heavy tier (gemini-2.5-flash) during cooldown, got tier=%s model=%s", tier, provider.ModelName())
	}

	// 4. Cooldown expired
	svc.MarkFastRateLimited(-1 * time.Second)
	if svc.IsFastRateLimited() {
		t.Fatalf("expected IsFastRateLimited() to be false after expiry")
	}

	// 5. Routing after cooldown: back to fast tier
	provider, tier = svc.selectLLM("sample context")
	if tier != "fast" || provider.ModelName() != "gpt-4.1-mini" {
		t.Fatalf("expected routing back to fast tier after cooldown expiry, got tier=%s model=%s", tier, provider.ModelName())
	}
}

func TestFormatLLMErrorMessages(t *testing.T) {
	fast := &dummyLLM{model: "gpt-4.1-mini"}
	heavy := &dummyLLM{model: "gemini-2.5-flash"}

	rateLimitErr := &llmpkg.RateLimitError{StatusCode: 429, Message: "TPM exceeded"}

	// Scenario A: Heavy provider configured and distinct
	svcA := &StudyService{
		repo:             &db.Repository{},
		fastLLMProvider:  fast,
		heavyLLMProvider: heavy,
	}
	errA := svcA.FormatLLMError(rateLimitErr, "fast")
	expectedA := "Fast provider rate-limited (status 429). Please try again — next attempt will use Heavy provider (gemini-2.5-flash)."
	if errA.Error() != expectedA {
		t.Errorf("Scenario A expected error %q, got %q", expectedA, errA.Error())
	}
	if !svcA.IsFastRateLimited() {
		t.Errorf("Scenario A expected fast tier to be marked rate limited")
	}

	// Scenario B: Heavy provider NOT configured (nil)
	svcB := &StudyService{
		fastLLMProvider: fast,
	}
	errB := svcB.FormatLLMError(rateLimitErr, "fast")
	expectedB := "Fast provider rate-limited (status 429: TPM limit reached). Please try again in a few seconds, or configure a Heavy provider (e.g. Gemini AI Studio) in Settings."
	if errB.Error() != expectedB {
		t.Errorf("Scenario B expected error %q, got %q", expectedB, errB.Error())
	}

	// Scenario C: Heavy provider is same model as Fast
	svcC := &StudyService{
		fastLLMProvider:  fast,
		heavyLLMProvider: &dummyLLM{model: "gpt-4.1-mini"},
	}
	errC := svcC.FormatLLMError(rateLimitErr, "fast")
	if errC.Error() != expectedB {
		t.Errorf("Scenario C expected error %q, got %q", expectedB, errC.Error())
	}

	// Scenario D: Non-rate-limit error passed through unchanged
	standardErr := fmt.Errorf("network timeout")
	errD := svcA.FormatLLMError(standardErr, "fast")
	if errD.Error() != "network timeout" {
		t.Errorf("Scenario D expected unchanged error 'network timeout', got %q", errD.Error())
	}
}

func TestPacedRateLimitStrategy(t *testing.T) {
	fast := &dummyLLM{model: "gpt-4.1-mini"}
	heavy := &dummyLLM{model: "gemini-2.5-flash"}

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := db.Init(dbPath, "")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer func() { _ = repo.Close() }()

	// Update user settings to PACED
	err = repo.UpdateUserSettings(models.UserSettings{
		RateLimitStrategy: models.RateLimitStrategyPaced,
	})
	if err != nil {
		t.Fatalf("failed to update user settings: %v", err)
	}

	svc := NewStudyService(Config{
		Repo:             repo,
		FastLLMProvider:  fast,
		HeavyLLMProvider: heavy,
	})

	// 1. Initial state: fast has not been called yet (timeSinceLastFastCall is large), should route to fast
	provider, tier := svc.selectLLM("sample context")
	if tier != "fast" || provider.ModelName() != "gpt-4.1-mini" {
		t.Fatalf("expected initial routing to fast, got tier=%s model=%s", tier, provider.ModelName())
	}

	// 2. Call fast provider (simulating any feature calling GenerateAnswer)
	_, err = svc.fastLLMProvider.GenerateAnswer("test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3. Immediately select LLM: since fast was called <60s ago and PACED is active, should route to heavy
	provider, tier = svc.selectLLM("sample context")
	if tier != "heavy" || provider.ModelName() != "gemini-2.5-flash" {
		t.Fatalf("expected routing to heavy tier under PACED strategy after recent fast call, got tier=%s model=%s", tier, provider.ModelName())
	}

	// 4. Manually advance lastFastCallTime past 60s
	svc.rateLimitMu.Lock()
	svc.lastFastCallTime = time.Now().Add(-65 * time.Second)
	svc.rateLimitMu.Unlock()

	// 5. Select LLM: window passed, should route back to fast
	provider, tier = svc.selectLLM("sample context")
	if tier != "fast" || provider.ModelName() != "gpt-4.1-mini" {
		t.Fatalf("expected routing back to fast tier after 60s window elapsed, got tier=%s model=%s", tier, provider.ModelName())
	}

	// 6. Switch settings back to STANDARD
	err = repo.UpdateUserSettings(models.UserSettings{
		RateLimitStrategy: models.RateLimitStrategyStandard,
	})
	if err != nil {
		t.Fatalf("failed to set standard settings: %v", err)
	}

	// Call fast again
	_, _ = svc.fastLLMProvider.GenerateAnswer("test prompt 2")

	// In STANDARD mode, recent fast call does NOT force heavy
	provider, tier = svc.selectLLM("sample context")
	if tier != "fast" || provider.ModelName() != "gpt-4.1-mini" {
		t.Fatalf("expected STANDARD strategy to route to fast even when called recently, got tier=%s model=%s", tier, provider.ModelName())
	}
}

