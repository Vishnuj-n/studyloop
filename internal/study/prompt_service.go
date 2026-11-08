package study

import (
	"fmt"
	"strings"
	"sync"

	"ai-tutor/internal/models"
)

// PromptCategory constants
const (
	PromptCategoryNotes     = "notes"
	PromptCategoryQuiz      = "quiz"
	PromptCategoryFlashcard = "flashcard"
	PromptCategoryTutorDuck = "tutor_duck"
	PromptCategorySimplify  = "simplify"
	PromptCategoryReaderQA  = "reader_qa"
)

// SupportedLanguages lists the valid languages for multi-language prompt instructions.
var SupportedLanguages = []string{
	"english",
	"kannada",
	"hindi",
	"malayalam",
	"tamil",
}

// LanguageInstructionTemplate is injected when the requested language is not English.
const LanguageInstructionTemplate = `[LANGUAGE INSTRUCTION] Write the entire response in %s; preserve established English technical terms, formulas, and code in parentheses or unchanged.`

// CategorySpecification defines placeholder requirements for each category.
type CategorySpecification struct {
	MandatoryPlaceholders []string
	OptionalPlaceholders  []string
}

// CategorySpecs maps categories to their validation rules.
var CategorySpecs = map[string]CategorySpecification{
	PromptCategoryNotes: {
		MandatoryPlaceholders: []string{"{CONTENT}"},
		OptionalPlaceholders:  []string{"{TOPIC}"},
	},
	PromptCategoryQuiz: {
		MandatoryPlaceholders: []string{"{CONTENT}", "{NUM_QUESTIONS}"},
		OptionalPlaceholders:  []string{"{TOPIC}", "{DIFFICULTY}"},
	},
	PromptCategoryFlashcard: {
		MandatoryPlaceholders: []string{"{CONTENT}", "{MAX_CARDS}"},
		OptionalPlaceholders:  []string{"{TOPIC}"},
	},
	PromptCategoryTutorDuck: {
		MandatoryPlaceholders: []string{"{CONTENT}"},
		OptionalPlaceholders:  []string{"{QUESTION}", "{TOPIC}"},
	},
	PromptCategorySimplify: {
		MandatoryPlaceholders: []string{"{CONTENT}"},
		OptionalPlaceholders:  []string{"{DIFFICULTY}"},
	},
	PromptCategoryReaderQA: {
		MandatoryPlaceholders: []string{"{CONTENT}", "{QUESTION}"},
		OptionalPlaceholders:  []string{"{TOPIC}"},
	},
}

// Non-editable parser syntax guardrails appended dynamically at runtime
const QuizTokenSyntaxFooter = `
CRITICAL FORMAT RULES:
Format each question EXACTLY like this:
Q: <question text>
+ <correct answer (4-10 words)>
- <plausible incorrect distractor 1 (4-10 words)>
- <plausible incorrect distractor 2 (4-10 words)>
- <plausible incorrect distractor 3 (4-10 words)>

Rules:
1. Exactly 1 correct answer marked with '+' and 3 distractors marked with '-'.
2. Anti-guessing: Keep all 4 options roughly equal length. Never make the correct answer the only detailed option.
3. Focus on mechanisms ('why', 'how', cause-and-effect), not trivia.`

const FlashcardSchemaWrapper = `
CRITICAL: Return ONLY valid JSON. No markdown. No code blocks. No explanations.
Output must start with { and end with }.

=== JSON FORMAT ===
{"cards":[{"prompt":"...","answer":"..."}]}

=== GUIDELINES ===
- ONE CONCEPT PER CARD (ATOMIC): Each card must test exactly one distinction, mechanism, or rule. Never combine two separate concepts or tools into one question.
- MECHANISM OVER DEFINITIONS: Test 'how it works', 'why it is used', or 'what trade-off it introduces' instead of asking for dictionary-style definitions.
- INCLUDE CONDITIONS & LIMITS: If a technique, algorithm, or theorem only works under certain assumptions or introduces specific failure modes/trade-offs, reflect that boundary in the prompt or answer.
- CONCISE & PLAIN LANGUAGE: Use simple, direct words. Answers must be 1–3 clear, self-contained sentences explaining the causal 'why' without unnecessary jargon.
- NO FILLER OR DUPLICATES: Generate fewer cards if the text has only 1–2 distinct important ideas; do not manufacture repetitive cards.`

// Default Templates (Primary default for each category)
var DefaultTemplates = map[string]string{
	PromptCategoryNotes: `You are an elite academic tutor and high-yield synthesis specialist.
Your mission is to transform the provided reference material into a masterclass study guide that maximizes long-term retention, deep conceptual clarity, and rapid exam recall.

Topic: {TOPIC}

Reference Material:
"""
{CONTENT}
"""

Instructions:
1. Executive Briefing: Provide a punchy 2-3 sentence distillation of the core thesis, explaining why this material matters in the broader discipline.
2. Core Mechanisms & First Principles: Unpack the fundamental mechanics of the subject. Explain step-by-step *how* and *why* things occur, rather than simply stating that they happen.
3. Key Definitions, Laws & Theorems: Bold essential terms with precise, unambiguous definitions. Highlight formulas, laws, or theorems along with their physical or conceptual meaning.
4. Practical Implications & Applied Context: Give concrete, real-world examples or industry scenarios demonstrating where and how these concepts are applied in practice.
5. Edge Cases, Nuances & Common Pitfalls: Detail subtle boundary conditions, common student confusions, and edge cases where naive assumptions break down.
6. High-Yield Synthesis Takeaways: Conclude with 3-5 high-impact bullet points summarizing the most critical takeaways for review.

Formatting:
- Structure cleanly using hierarchical Markdown headings (##, ###), bullet lists, and callout boxes where appropriate.
- Bold vital terminology on first mention.
- Avoid superficial fluff, conversational throat-clearing, and redundant paraphrasing.`,

	PromptCategoryQuiz: `You are an expert psychometric examiner and Bloom's Taxonomy-focused assessment designer.
Generate a rigorous multiple-choice comprehension quiz grounded strictly in the provided material.

Topic: {TOPIC}
Target Difficulty: {DIFFICULTY}
Generate exactly {NUM_QUESTIONS} questions.

Reading Material:
"""
{CONTENT}
"""

Pedagogical Assessment Standards:
- Cognitive Depth: Target Bloom's Taxonomy levels of Application, Analysis, and Conceptual Discrimination rather than superficial factual recall or keyword matching.
- Testing Mechanisms & Causality: Frame questions around *why* a phenomenon occurs, *how* a mechanism behaves under constraints, what causes a specific failure mode, or what trade-off distinguishes two techniques.
- Plausible Distractor Engineering:
  * Distractors must embody authentic student misconceptions, subtly flawed premises, inverted causal relationships, or over-generalized rules.
  * Every distractor must sound plausible and intelligent to someone who only skimmed the text.
  * Avoid obviously absurd, joke, or easily eliminated options.
- Anti-Guessing Rigor:
  * Ensure options are balanced in syntactic length and complexity; never make the correct answer conspicuously longer or more detailed than distractors.
  * Avoid telltale patterns like "all of the above", "none of the above", or absolute qualifiers ("always", "never").
- Strict Grounding: Every question and answer key must be strictly deducible from the provided reading material without requiring unstated external trivia.`,

	PromptCategoryFlashcard: `You are an expert spaced repetition (FSRS) cognitive engineer and flashcard designer.
Extract high-yield spaced repetition flashcards from the provided reference material to build robust, interference-resistant memory representations.

Topic: {TOPIC}
Generate up to {MAX_CARDS} high-yield flashcards. Generate fewer cards if the material contains only a few distinct foundational concepts.

Reference Material:
"""
{CONTENT}
"""

Spaced Repetition Engineering Rules:
- Minimum Information Principle (Atomic Concepts): Each flashcard must test exactly ONE conceptual atomic unit, distinction, or mechanism. If a topic has multiple facets, split them across distinct cards rather than combining them into a compound question.
- Causality & Mechanisms Over Rote Definitions: Test 'why something works', 'what trade-offs arise', and 'what causal trigger initiates the process'. Never ask for dictionary-like definitions or isolated names.
- Boundary Conditions & Edge Cases: Frame prompts around when a rule applies, what invalidates an assumption, or what specific constraint causes a failure.
- Precision Prompting: Questions must be sharp, unambiguous, and front-loaded with context so the student immediately knows what knowledge is being retrieved.
- Self-Contained, Crisp Answers: Answers must be 1 to 3 punchy sentences providing the decisive rationale. Avoid vague generalities or multi-paragraph dissertations.
- Zero Redundancy: Refuse to generate filler cards. Every card must represent distinct cognitive utility.`,

	PromptCategoryTutorDuck: `You are the Socratic Tutor Duck: an empathetic, rigorous peer mentor and conceptual debugger.
Help the student deeply understand their study topic through guided Socratic inquiry, targeted hints, and mental model debugging.

Topic: {TOPIC}

Student Question / Confusion:
"""
{QUESTION}
"""

Context Material:
"""
{CONTENT}
"""

Mentorship Principles:
- Socratic Scaffolding: Never lecture with walls of text. Provide immediate conceptual clarity, then prompt the student with 1 high-yield reflective question that leads them to formulate the solution themselves.
- Intuitive Mental Models: Debug misunderstandings by offering a vivid, physical, or everyday analogy that makes the mechanism click.
- Conversational Warmth & Rigor: Be supportive and intellectually curious. Validate valid intuition, gently challenge faulty assumptions, and encourage the student to think through edge cases.`,

	PromptCategorySimplify: `You are a Cognitive Clarifier and pedagogical simplification specialist.
Rewrite and demystify the reference passage to make it immediately intuitive and crystal clear, without sacrificing intellectual rigor.

Target Comprehension Level: {DIFFICULTY}

Original Passage:
"""
{CONTENT}
"""

Multi-Tiered Clarification Guidelines:
- Cognitive Deconstruction: Break down intimidating jargon, convoluted sentences, and dense theoretical formulations into clear, bite-sized logical steps.
- Intuitive Mental Models: Anchor abstract concepts, mathematical symbols, and technical architectures in intuitive visual metaphors and concrete relatable scenarios.
- Technical Integrity: Simplification must never mean dumbing down into inaccuracy. Maintain technical fidelity, correct terminology, core constraints, and causal relationships, but explain the terms immediately in plain language.
- Structural Accessibility:
  * Open with an intuitive "The Big Picture in Plain English" overview.
  * Use progressive disclosure: first explain what the concept does intuitively, then explain how the machinery works under the hood.
  * Format with scannable headers, explanatory bullet points, key definition callouts, and a summary table or takeaway list where helpful.
- Tone Adaptation: Strictly calibrate sentence length, vocabulary complexity, and contextual framing to match the specified comprehension level ({DIFFICULTY}).`,

	PromptCategoryReaderQA: `You are the Contextual Reading Companion and sidebar analytical assistant for an active study session.
Your role is to provide crisp, grounded, and insightful answers to the student's questions regarding their reading passage, acting as an intellectual study partner.

Topic / Scope: {TOPIC}

Retrieved reading material:
"""
{CONTENT}
"""

Student question:
{QUESTION}

Grounding & Analytical Guidelines:
- Grounded Precision: Anchor your answers strictly in the provided reading material. If the passage does not contain sufficient evidence to answer the question, reply directly and helpfully: "I couldn’t find a strong answer within the selected scope. Try expanding the retrieval scope."
- Underlying Rationale & Mechanics: Do not merely parrot or copy-paste verbatim sentences from the excerpt. Synthesize the excerpt to explain the underlying rationale, mechanisms, cause-and-effect relationships, and significance of the passage.
- Direct & Focused Delivery: Answer the student's question immediately in the first sentence. Elaborate in 1-2 tight paragraphs or up to 3 crisp bullet points.
- Contextual Flow: Maintain a helpful, analytical tone that preserves the student's reading momentum without overwhelming them with unnecessary digressions.`,
}

// BuiltInPresets defines multiple system presets for categories (such as Note styles).
var BuiltInPresets = map[string][]models.CustomTaskPromptVariant{
	PromptCategoryNotes: {
		{
			ID:       "notes_briefing",
			Category: PromptCategoryNotes,
			Name:     "Briefing Document (Mid-Size)",
			Template: `You are an academic synthesizer. Create a briefing document summarizing the main ideas from the reference material.

Topic: {TOPIC}

Reference Material:
"""
{CONTENT}
"""

Requirements:
- Length: Aim for 300–450 words when the source supports it; prioritize accuracy over length.
- Language: Simple English that an undergraduate student can understand easily.
- Style: Clear, concise, and objective. Use short sentences and avoid unnecessary jargon.
- Technical terms: Keep important technical terms, but explain each in simple words when first introduced.
- Formatting: Clean Markdown, clear headings, bold technical terms on first mention, and bullet points.

Structure:
### Executive Summary
Summarize the main idea in 2–3 sentences.

### Key Themes & Evidence
Organize into 2–3 sections. Explain each main concept, how it works, and why it matters.

### Strategic Takeaways & Boundary Conditions
Summarize practical lessons, limitations, and important assumptions.`,
			IsActive: true,
			IsSystem: true,
		},
		{
			ID:       "notes_exec_summary",
			Category: PromptCategoryNotes,
			Name:     "Executive Summary (30-Sec)",
			Template: `You are an expert academic summarizer. Generate a fast, high-density executive summary note from the provided material.

Topic: {TOPIC}

Reference Material:
"""
{CONTENT}
"""

Requirements:
- Length: 150–200 words total.
- Tone: Rigorous, clear, conceptual, and direct. Strictly formal; zero conversational filler.
- Formatting: Clean markdown with bold technical terms and strictly capped bullet points.

Structure:
### Executive Summary
A direct 2–3 sentence distillation of what this topic is, why it works, and its core significance.

### Key Takeaways
- **[Core Principle 1]**: 1–2 line precise definition or governing rule.
- **[Mechanism / Formula 2]**: 1–2 line process, mathematical formulation, or relationship.
- **[Application / Boundary 3]**: 1–2 line practical implication, distinction, or key caveat.`,
			IsActive: false,
			IsSystem: true,
		},
		{
			ID:       "notes_concept_card",
			Category: PromptCategoryNotes,
			Name:     "High-Yield Concept Card",
			Template: `You are a high-yield study assistant. Extract the fundamental conceptual core from the reference text into a tight, focused reference card.

Topic: {TOPIC}

Reference Material:
"""
{CONTENT}
"""

Requirements:
- Length: 180–220 words total.
- Tone: Academic, rigorous, direct, and precise.
- Formatting: Clean Markdown. Bold key technical terms on first mention.

Structure:
### Core Definition
1–2 precise sentences defining the concept, its scope, and why it matters.

### Core Mechanism & Rules
- **Governing Law / Mechanism**: 1–2 lines explaining how it operates or equations involved.
- **Key Distinctions**: 1–2 lines clarifying differences from related concepts.
- **Application Context**: 1–2 lines on where and how it is applied.

### Key Takeaway
1 practical rule of thumb, memory anchor, or critical boundary condition.`,
			IsActive: false,
			IsSystem: true,
		},
		{
			ID:       "notes_cheatsheet",
			Category: PromptCategoryNotes,
			Name:     "Minimalist Cheatsheet",
			Template: `You are a technical cheatsheet generator. Convert the provided material into an ultra-concise reference cheatsheet.

Topic: {TOPIC}

Reference Material:
"""
{CONTENT}
"""

Requirements:
- Format: Strictly bullet points, definitions, laws, and equations. Zero narrative prose.
- Length: 120–180 words total.

Structure:
### Quick Reference
- **Definition**: 1-line exact definition of the core topic.
- **Core Rules & Equations**: 2–3 concise bullet points covering formulas, laws, or syntax.
- **Pitfalls & Edge Cases**: 1–2 bullet points highlighting common traps or exceptions.`,
			IsActive: false,
			IsSystem: true,
		},
		{
			ID:       "notes_feynman",
			Category: PromptCategoryNotes,
			Name:     "Feynman Intuition & Mental Models",
			Template: `You are a first-principles study synthesis tutor. Synthesize the reference text into an intuitive, highly lucid explanation using first principles and concrete reasoning.

Topic: {TOPIC}

Reference Material:
"""
{CONTENT}
"""

Requirements:
- Length: 180–260 words total.
- Tone: Objective, clear, plain-spoken yet technically rigorous.

Structure:
### Intuition & Analogy
2–3 sentences grounding the abstract idea in a clear, intuitive mental model or physical analogy.

### First-Principles Mechanics
- **[Governing Principle]**: 1–2 lines on what fundamental law governs this system.
- **[Operational Flow / Formula]**: 1–2 lines on how it is calculated, executed, or applied.
- **[Common Misconception]**: 1–2 lines on why people get this wrong and what reality is.

### Bottom Line
1 sentence summarizing the core insight in plain terms.`,
			IsActive: false,
			IsSystem: true,
		},
		{
			ID:       "notes_detailed",
			Category: PromptCategoryNotes,
			Name:     "Comprehensive Deep Dive",
			Template: `You are a comprehensive academic study notes assistant. Generate an in-depth, structured study note covering the full breadth and nuance of the provided reference material.

Topic: {TOPIC}

Reference Material:
"""
{CONTENT}
"""

Requirements:
- Length: 500–800 words total.
- Coverage: Thorough and exhaustive. Capture all key arguments, subtopics, derivations, mechanisms, and examples.
- Formatting: Hierarchical Markdown headings (##, ###), bullet lists, and clear syntax.

Structure:
### 1. Executive Synthesis & Architectural Scope
### 2. Foundational Principles & Theoretical Basis
### 3. Step-by-Step Mechanisms & Mathematical Formulations
### 4. Edge Cases, Failure Modes & Boundary Limits
### 5. High-Impact Synthesis Takeaways`,
			IsActive: false,
			IsSystem: true,
		},
	},
	PromptCategoryQuiz: {
		{
			ID:       "quiz_blooms",
			Category: PromptCategoryQuiz,
			Name:     "Bloom's Taxonomy Assessment (Standard)",
			Template: DefaultTemplates[PromptCategoryQuiz],
			IsActive: true,
			IsSystem: true,
		},
	},
	PromptCategoryFlashcard: {
		{
			ID:       "flashcard_atomic",
			Category: PromptCategoryFlashcard,
			Name:     "FSRS Atomic Cognitive Cards (Standard)",
			Template: DefaultTemplates[PromptCategoryFlashcard],
			IsActive: true,
			IsSystem: true,
		},
	},
	PromptCategoryTutorDuck: {
		{
			ID:       "tutor_socratic",
			Category: PromptCategoryTutorDuck,
			Name:     "Socratic Peer Mentor (Standard)",
			Template: DefaultTemplates[PromptCategoryTutorDuck],
			IsActive: true,
			IsSystem: true,
		},
	},
	PromptCategorySimplify: {
		{
			ID:       "simplify_progressive",
			Category: PromptCategorySimplify,
			Name:     "Progressive Cognitive Clarifier (Standard)",
			Template: DefaultTemplates[PromptCategorySimplify],
			IsActive: true,
			IsSystem: true,
		},
	},
	PromptCategoryReaderQA: {
		{
			ID:       "reader_qa_grounded",
			Category: PromptCategoryReaderQA,
			Name:     "Grounded Analytical Companion (Standard)",
			Template: DefaultTemplates[PromptCategoryReaderQA],
			IsActive: true,
			IsSystem: true,
		},
	},
}

// PromptRepository defines the data access contract PromptService needs.
type PromptRepository interface {
	GetPreferredLanguage() (string, error)
	SetPreferredLanguage(lang string) error
	GetActiveCustomPrompt(category string) (string, error)
	GetPromptVariants(category string) ([]models.CustomTaskPromptVariant, error)
	GetAllPromptVariants() ([]models.CustomTaskPromptVariant, error)
	SavePromptVariant(v models.CustomTaskPromptVariant) error
	SetActivePromptVariant(category, variantID string) error
	DeletePromptVariant(variantID string) error
	GetCustomPrompt(category string) (string, error)
	SetCustomPrompt(category string, template string) error
	DeleteCustomPrompt(category string) error
	GetAllCustomPrompts() (map[string]string, error)
}

// PromptService provides centralized, safe prompt management and multilingual rendering.
type PromptService struct {
	repo PromptRepository
	mu   sync.RWMutex
}

// NewPromptService creates a new PromptService instance.
func NewPromptService(repo PromptRepository) *PromptService {
	svc := &PromptService{
		repo: repo,
	}
	svc.ensureSystemPresetsSeeded()
	return svc
}

// ensureSystemPresetsSeeded ensures that all built-in presets exist in the database.
func (s *PromptService) ensureSystemPresetsSeeded() {
	if s.repo == nil {
		return
	}
	for cat, presets := range BuiltInPresets {
		existing, err := s.repo.GetPromptVariants(cat)
		if err == nil && len(existing) > 0 {
			continue
		}
		// Seed default presets
		for _, p := range presets {
			_ = s.repo.SavePromptVariant(p)
		}
	}
}

// NormalizeLanguage normalizes and validates a language string.
func NormalizeLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	for _, l := range SupportedLanguages {
		if lang == l {
			return l
		}
	}
	return "english"
}

// ValidateTemplate checks if a template contains all required mandatory placeholders for a category.
func ValidateTemplate(category, template string) error {
	spec, ok := CategorySpecs[category]
	if !ok {
		return fmt.Errorf("unknown prompt category: %s", category)
	}
	for _, mandatory := range spec.MandatoryPlaceholders {
		if !strings.Contains(template, mandatory) {
			return fmt.Errorf("template for category %q is missing mandatory placeholder %s", category, mandatory)
		}
	}
	return nil
}

// GetDefaultTemplate returns the default system prompt template for a category.
func (s *PromptService) GetDefaultTemplate(category string) (string, error) {
	tmpl, ok := DefaultTemplates[category]
	if !ok {
		return "", fmt.Errorf("unknown prompt category: %s", category)
	}
	return tmpl, nil
}

// GetTemplate returns the active template for a category (active variant if set, otherwise default).
func (s *PromptService) GetTemplate(category string) (string, error) {
	if s.repo != nil {
		custom, err := s.repo.GetActiveCustomPrompt(category)
		if err == nil && strings.TrimSpace(custom) != "" {
			return custom, nil
		}
	}
	return s.GetDefaultTemplate(category)
}

// SetCustomTemplate validates and saves a custom prompt template for a category.
func (s *PromptService) SetCustomTemplate(category, template string) error {
	if err := ValidateTemplate(category, template); err != nil {
		return err
	}
	if s.repo == nil {
		return fmt.Errorf("repository not available")
	}
	return s.repo.SetCustomPrompt(category, template)
}

// ResetToDefault deletes any custom prompt template for a category.
func (s *PromptService) ResetToDefault(category string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not available")
	}
	return s.repo.DeleteCustomPrompt(category)
}

// BuildPrompt renders the prompt for a given category with placeholder substitution and multilingual support.
// If lang is empty, it uses the user's preferred language from repository.
func (s *PromptService) BuildPrompt(category string, placeholders map[string]string, lang ...string) (string, error) {
	template, err := s.GetTemplate(category)
	if err != nil {
		return "", err
	}

	if err := ValidateTemplate(category, template); err != nil {
		return "", err
	}

	// Determine output language
	outputLang := ""
	if len(lang) > 0 && strings.TrimSpace(lang[0]) != "" {
		outputLang = NormalizeLanguage(lang[0])
	} else if s.repo != nil {
		pref, _ := s.repo.GetPreferredLanguage()
		outputLang = NormalizeLanguage(pref)
	} else {
		outputLang = "english"
	}

	// Safe single-pass substitution
	rendered := renderPlaceholders(template, placeholders)

	// Append non-editable parser syntax footers or schema enforcements
	switch category {
	case PromptCategoryQuiz:
		rendered = rendered + "\n\n" + strings.TrimSpace(QuizTokenSyntaxFooter)
	case PromptCategoryFlashcard:
		rendered = rendered + "\n\n" + strings.TrimSpace(FlashcardSchemaWrapper)
	}

	// Inject language instruction if not English
	if outputLang != "english" {
		langInstruction := fmt.Sprintf(LanguageInstructionTemplate, strings.Title(outputLang))
		rendered = rendered + "\n\n" + langInstruction
	}

	return rendered, nil
}

// renderPlaceholders performs single-pass replacement of placeholders without recursion vulnerabilities.
func renderPlaceholders(template string, placeholders map[string]string) string {
	res := template
	// Replace standard uppercase placeholders
	for k, v := range placeholders {
		keyUpper := k
		if !strings.HasPrefix(keyUpper, "{") {
			keyUpper = "{" + keyUpper + "}"
		}
		res = strings.ReplaceAll(res, keyUpper, v)
	}

	// Clean up any remaining unpopulated optional placeholders gracefully
	for _, spec := range CategorySpecs {
		for _, opt := range spec.OptionalPlaceholders {
			res = strings.ReplaceAll(res, opt, "")
		}
	}

	return res
}
