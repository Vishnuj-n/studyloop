package embeddings

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"unicode"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
)

var (
	promptTokenizerMu    sync.RWMutex
	promptTokenizer      *tokenizer.Tokenizer
	tokenizerUnavailable bool
)

// InitPromptTokenizer initializes shared tokenizer used for prompt budgeting.
func InitPromptTokenizer(tokenizerPath string) error {
	tok, err := pretrained.FromFile(tokenizerPath)
	if err != nil {
		return err
	}

	promptTokenizerMu.Lock()
	promptTokenizer = tok
	tokenizerUnavailable = false
	promptTokenizerMu.Unlock()

	return nil
}

// CountTokens counts tokens using configured tokenizer or an approximation fallback if uninitialized.
func CountTokens(text string) (int, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}

	tok := getPromptTokenizer()
	if tok == nil {
		words := len(strings.Fields(text))
		approx := int(float64(words) * 1.3)
		if approx < 1 {
			approx = (len(text) + 3) / 4
		}
		if approx < 1 {
			approx = 1
		}
		return approx, nil
	}

	enc, err := tok.EncodeSingle(text, true)
	if err != nil {
		return 0, fmt.Errorf("tokenizer encode failed in CountTokens: %w", err)
	}

	return len(enc.Ids), nil
}

// TruncateToTokens trims text to token limit, preferring clean sentence boundaries.
func TruncateToTokens(text string, limit int) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" || limit <= 0 {
		return "", nil
	}

	tok := getPromptTokenizer()
	if tok == nil {
		words := strings.Fields(text)
		if len(words) == 0 {
			return "", nil
		}
		maxWords := int(float64(limit) / 1.3)
		if maxWords <= 0 {
			maxWords = 1
		}
		if maxWords >= len(words) {
			return text, nil
		}
		truncated := strings.Join(words[:maxWords], " ")
		trimmed := trimToSentenceBoundary(truncated)
		if trimmed == "" {
			return truncated, nil
		}
		return trimmed, nil
	}

	enc, err := tok.EncodeSingle(text, true)
	if err != nil {
		return "", fmt.Errorf("tokenizer encode failed in TruncateToTokens: %w", err)
	}

	if len(enc.Ids) <= limit {
		return text, nil
	}

	decoded := tok.Decode(enc.Ids[:limit], true)

	return trimToSentenceBoundary(decoded), nil
}

func getPromptTokenizer() *tokenizer.Tokenizer {
	promptTokenizerMu.RLock()
	tok := promptTokenizer
	unavailable := tokenizerUnavailable
	promptTokenizerMu.RUnlock()
	if tok != nil {
		return tok
	}
	if unavailable {
		return nil
	}

	promptTokenizerMu.Lock()
	defer promptTokenizerMu.Unlock()

	if promptTokenizer != nil {
		return promptTokenizer
	}
	if tokenizerUnavailable {
		return nil
	}

	for _, candidate := range tokenizerPathCandidates() {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}

		tok, err := pretrained.FromFile(candidate)
		if err != nil {
			log.Printf("failed to initialize prompt tokenizer from %s: %v", candidate, err)
			continue
		}

		promptTokenizer = tok
		tokenizerUnavailable = false
		return promptTokenizer
	}

	tokenizerUnavailable = true

	return nil
}

func tokenizerPathCandidates() []string {
	candidates := make([]string, 0, 5)
	if fromEnv := strings.TrimSpace(os.Getenv("TOKENIZER_PATH")); fromEnv != "" {
		candidates = append(candidates, fromEnv)
	}
	candidates = append(candidates,
		"asset/tokenizer.json",
		"../asset/tokenizer.json",
		"../../asset/tokenizer.json",
		"../../../asset/tokenizer.json",
	)
	return candidates
}

func trimToSentenceBoundary(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	lastEnd := -1
	for i, r := range text {
		if r == '.' || r == '!' || r == '?' {
			lastEnd = i
		}
	}
	if lastEnd >= 0 {
		return strings.TrimSpace(text[:lastEnd+1])
	}

	lastSpace := -1
	for i, r := range text {
		if unicode.IsSpace(r) {
			lastSpace = i
		}
	}
	if lastSpace > 0 {
		return strings.TrimSpace(text[:lastSpace])
	}

	return text
}
