package internal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/Muxcore-Media/contracts-ai/infer"
)

// Provider is the inference seam. Feature modules call ai-runtime; tests inject this.
type Provider interface {
	Name() string
	Complete(ctx context.Context, req infer.CompleteRequest) (infer.CompleteResponse, error)
	Embed(ctx context.Context, req infer.EmbedRequest) (infer.EmbedResponse, error)
	Transcribe(ctx context.Context, req infer.TranscribeRequest) (infer.TranscribeResponse, error)
	Classify(ctx context.Context, req infer.ClassifyRequest) (infer.ClassifyResponse, error)
	Translate(ctx context.Context, req infer.TranslateRequest) (infer.TranslateResponse, error)
}

// HeuristicProvider is the default offline engine. It is deterministic and
// never leaves the process — used in tests and households without API keys.
type HeuristicProvider struct{}

func (HeuristicProvider) Name() string { return infer.ProviderHeuristic }

func (HeuristicProvider) Complete(_ context.Context, req infer.CompleteRequest) (infer.CompleteResponse, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return infer.CompleteResponse{}, fmt.Errorf("prompt required")
	}
	text := heuristicComplete(req.System, req.Prompt)
	return infer.CompleteResponse{JobID: req.JobID, Provider: infer.ProviderHeuristic, Text: text}, nil
}

func (HeuristicProvider) Embed(_ context.Context, req infer.EmbedRequest) (infer.EmbedResponse, error) {
	if strings.TrimSpace(req.Text) == "" {
		return infer.EmbedResponse{}, fmt.Errorf("text required")
	}
	vec := heuristicEmbed(req.Text, 32)
	return infer.EmbedResponse{
		JobID: req.JobID, Provider: infer.ProviderHeuristic,
		Embedding: vec, Dimensions: len(vec),
	}, nil
}

func (HeuristicProvider) Transcribe(_ context.Context, req infer.TranscribeRequest) (infer.TranscribeResponse, error) {
	cues := heuristicTranscribe(req.TranscriptHint, req.DurationSec)
	lang := req.Language
	if lang == "" {
		lang = detectLanguage(req.TranscriptHint)
	}
	return infer.TranscribeResponse{
		JobID: req.JobID, Provider: infer.ProviderHeuristic,
		Language: lang, Cues: cues,
	}, nil
}

func (HeuristicProvider) Classify(_ context.Context, req infer.ClassifyRequest) (infer.ClassifyResponse, error) {
	if strings.TrimSpace(req.Text) == "" {
		return infer.ClassifyResponse{}, fmt.Errorf("text required")
	}
	labels := req.Labels
	if len(labels) == 0 {
		labels = defaultClassifyLabels
	}
	hits := heuristicClassify(req.Text, labels)
	return infer.ClassifyResponse{JobID: req.JobID, Provider: infer.ProviderHeuristic, Hits: hits}, nil
}

func (HeuristicProvider) Translate(_ context.Context, req infer.TranslateRequest) (infer.TranslateResponse, error) {
	if strings.TrimSpace(req.Text) == "" {
		return infer.TranslateResponse{}, fmt.Errorf("text required")
	}
	if strings.TrimSpace(req.Target) == "" {
		return infer.TranslateResponse{}, fmt.Errorf("target required")
	}
	src := req.Source
	if src == "" {
		src = detectLanguage(req.Text)
	}
	out := heuristicTranslate(req.Text, src, req.Target)
	return infer.TranslateResponse{
		JobID: req.JobID, Provider: infer.ProviderHeuristic,
		Text: out, Source: src, Target: req.Target,
	}, nil
}

var defaultClassifyLabels = []string{
	"sexual", "graphic_violence", "drug_use", "profanity", "jump_scare", "self_harm",
	"wrong_audio_language", "missing_subtitles", "wrong_match", "quality_issue",
}

func heuristicComplete(system, prompt string) string {
	blob := strings.ToLower(system + "\n" + prompt)
	switch {
	case strings.Contains(blob, "wrong language") || strings.Contains(blob, "audio is in") || strings.Contains(blob, "dubbed in"):
		lang := extractWantedLanguage(blob)
		return fmt.Sprintf(`{"intent":"wrong_audio_language","language":%q,"action":"replace_media"}`, lang)
	case strings.Contains(blob, "missing subtitle") || strings.Contains(blob, "no subtitles"):
		return `{"intent":"missing_subtitles","action":"generate_subtitles"}`
	case strings.Contains(blob, "wrong match") || strings.Contains(blob, "not the right movie") || strings.Contains(blob, "misidentified"):
		return `{"intent":"wrong_match","action":"rematch_metadata"}`
	case strings.Contains(blob, "recommend") || strings.Contains(blob, "suggest"):
		return `{"intent":"recommend","action":"suggest_additions"}`
	case strings.Contains(blob, "classify"):
		return `{"intent":"quality_issue","action":"notify_admin"}`
	default:
		return strings.TrimSpace(prompt)
	}
}

func extractWantedLanguage(blob string) string {
	for _, lang := range []string{"english", "spanish", "french", "german", "japanese", "korean", "portuguese", "italian", "chinese"} {
		if strings.Contains(blob, "in "+lang) || strings.Contains(blob, "want "+lang) || strings.Contains(blob, lang+" version") {
			return lang[:2]
		}
	}
	if strings.Contains(blob, "english") {
		return "en"
	}
	return "en"
}

func heuristicEmbed(text string, dim int) []float64 {
	vec := make([]float64, dim)
	toks := tokenize(text)
	for _, tok := range toks {
		sum := sha256.Sum256([]byte(tok))
		idx := int(binary.BigEndian.Uint32(sum[:4]) % uint32(dim))
		vec[idx] += 1
	}
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm == 0 {
		return vec
	}
	norm = math.Sqrt(norm)
	for i := range vec {
		vec[i] /= norm
	}
	return vec
}

var sentenceSplit = regexp.MustCompile(`(?m)[.!?\n]+`)

func heuristicTranscribe(hint string, durationSec float64) []infer.TranscriptCue {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		if durationSec <= 0 {
			return nil
		}
		return []infer.TranscriptCue{{
			StartMs: 0,
			EndMs:   int64(durationSec * 1000),
			Text:    "[unintelligible]",
		}}
	}
	parts := sentenceSplit.Split(hint, -1)
	var lines []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			lines = append(lines, p)
		}
	}
	if len(lines) == 0 {
		lines = []string{hint}
	}
	if durationSec <= 0 {
		durationSec = float64(len(lines)) * 2.5
	}
	slot := durationSec * 1000 / float64(len(lines))
	cues := make([]infer.TranscriptCue, 0, len(lines))
	for i, line := range lines {
		start := int64(float64(i) * slot)
		end := int64(float64(i+1) * slot)
		if end <= start {
			end = start + 500
		}
		cues = append(cues, infer.TranscriptCue{StartMs: start, EndMs: end, Text: line})
	}
	return cues
}

func heuristicClassify(text string, labels []string) []infer.ClassifyHit {
	lower := strings.ToLower(text)
	hits := make([]infer.ClassifyHit, 0, len(labels))
	for _, label := range labels {
		key := strings.ToLower(strings.ReplaceAll(label, "_", " "))
		conf := 0.0
		if strings.Contains(lower, strings.ToLower(label)) || strings.Contains(lower, key) {
			conf = 0.92
		} else {
			for _, w := range strings.Fields(key) {
				if len(w) > 2 && strings.Contains(lower, w) {
					conf = 0.7
					break
				}
			}
		}
		hits = append(hits, infer.ClassifyHit{Label: label, Confidence: conf})
	}
	return hits
}

var tinyDict = map[string]map[string]string{
	"es": {"hello": "hola", "the": "el", "movie": "película", "language": "idioma"},
	"fr": {"hello": "bonjour", "the": "le", "movie": "film", "language": "langue"},
	"de": {"hello": "hallo", "the": "der", "movie": "film", "language": "sprache"},
}

func heuristicTranslate(text, source, target string) string {
	source = normalizeLang(source)
	target = normalizeLang(target)
	if source == target {
		return text
	}
	dict := tinyDict[target]
	if dict == nil {
		return text
	}
	words := strings.Fields(text)
	for i, w := range words {
		clean := strings.ToLower(strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) }))
		if repl, ok := dict[clean]; ok {
			words[i] = strings.ReplaceAll(strings.ToLower(w), clean, repl)
		}
	}
	return strings.Join(words, " ")
}

func detectLanguage(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "hola") || strings.Contains(lower, "película") || strings.Contains(lower, "qué"):
		return "es"
	case strings.Contains(lower, "bonjour") || strings.Contains(lower, "le film"):
		return "fr"
	case strings.Contains(lower, "hallo") || strings.Contains(lower, "sprache"):
		return "de"
	default:
		return "en"
	}
}

func normalizeLang(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	switch code {
	case "english", "eng":
		return "en"
	case "spanish", "spa":
		return "es"
	case "french", "fra", "fre":
		return "fr"
	case "german", "deu", "ger":
		return "de"
	case "japanese", "jpn":
		return "ja"
	case "korean", "kor":
		return "ko"
	default:
		if len(code) >= 2 {
			return code[:2]
		}
		return code
	}
}

func tokenize(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

// OpenAIProvider calls an OpenAI-compatible /v1/chat/completions endpoint.
// Households without a key keep using HeuristicProvider.
type OpenAIProvider struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

func (p OpenAIProvider) Name() string { return infer.ProviderOpenAI }

func (p OpenAIProvider) client() *http.Client {
	if p.HTTPClient != nil {
		return p.HTTPClient
	}
	return newGuardedClient(30 * time.Second)
}

func (p OpenAIProvider) Complete(ctx context.Context, req infer.CompleteRequest) (infer.CompleteResponse, error) {
	if p.APIKey == "" {
		return infer.CompleteResponse{}, fmt.Errorf("openai api key not configured")
	}
	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com"
	}
	if err := guardOutboundURL(base + "/v1/chat/completions"); err != nil {
		return infer.CompleteResponse{}, err
	}
	model := p.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": req.System},
			{"role": "user", "content": req.Prompt},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return infer.CompleteResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return infer.CompleteResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client().Do(httpReq)
	if err != nil {
		return infer.CompleteResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return infer.CompleteResponse{}, err
	}
	if resp.StatusCode >= 300 {
		return infer.CompleteResponse{}, fmt.Errorf("openai: %s", bytes.TrimSpace(data))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return infer.CompleteResponse{}, err
	}
	text := ""
	if len(parsed.Choices) > 0 {
		text = parsed.Choices[0].Message.Content
	}
	return infer.CompleteResponse{JobID: req.JobID, Provider: infer.ProviderOpenAI, Text: text}, nil
}

func (p OpenAIProvider) Embed(ctx context.Context, req infer.EmbedRequest) (infer.EmbedResponse, error) {
	// Cloud embeddings are opt-in; fall back to the local hasher so feature
	// modules keep working when only a chat key is configured.
	return HeuristicProvider{}.Embed(ctx, req)
}

func (p OpenAIProvider) Transcribe(ctx context.Context, req infer.TranscribeRequest) (infer.TranscribeResponse, error) {
	return HeuristicProvider{}.Transcribe(ctx, req)
}

func (p OpenAIProvider) Classify(ctx context.Context, req infer.ClassifyRequest) (infer.ClassifyResponse, error) {
	return HeuristicProvider{}.Classify(ctx, req)
}

func (p OpenAIProvider) Translate(ctx context.Context, req infer.TranslateRequest) (infer.TranslateResponse, error) {
	return HeuristicProvider{}.Translate(ctx, req)
}

// Router picks a named provider, defaulting to heuristic.
type Router struct {
	Default   Provider
	OpenAI    Provider
	Preferred string
}

func (r Router) pick(name string) Provider {
	if name == "" {
		name = r.Preferred
	}
	switch name {
	case infer.ProviderOpenAI:
		if r.OpenAI != nil {
			return r.OpenAI
		}
	}
	if r.Default != nil {
		return r.Default
	}
	return HeuristicProvider{}
}
