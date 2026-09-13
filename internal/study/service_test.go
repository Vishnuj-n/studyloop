package study

import (
	"fmt"
	"testing"
	"time"

	"ai-tutor/internal/db"
	llmpkg "ai-tutor/internal/llm"
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
	provider, tier := svc.selectLLM("sample context", 0)
	if tier != "fast" || provider.ModelName() != "gpt-4.1-mini" {
		t.Fatalf("expected initial routing to fast tier (gpt-4.1-mini), got tier=%s model=%s", tier, provider.ModelName())
	}

	// 2. Mark fast rate-limited
	svc.MarkFastRateLimited(45 * time.Second)
	if !svc.IsFastRateLimited() {
		t.Fatalf("expected IsFastRateLimited() to be true")
	}

	// 3. Routing during cooldown: escalates to heavy tier
	provider, tier = svc.selectLLM("sample context", 0)
	if tier != "heavy" || provider.ModelName() != "gemini-2.5-flash" {
		t.Fatalf("expected routing to heavy tier (gemini-2.5-flash) during cooldown, got tier=%s model=%s", tier, provider.ModelName())
	}

	// 4. Cooldown expired
	svc.MarkFastRateLimited(-1 * time.Second)
	if svc.IsFastRateLimited() {
		t.Fatalf("expected IsFastRateLimited() to be false after expiry")
	}

	// 5. Routing after cooldown: back to fast tier
	provider, tier = svc.selectLLM("sample context", 0)
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
