package internal

import (
	"context"
	"strings"
	"testing"

	"github.com/Muxcore-Media/contracts-ai/infer"
)

func TestHeuristicCompleteClassifiesWrongLanguage(t *testing.T) {
	p := HeuristicProvider{}
	got, err := p.Complete(context.Background(), infer.CompleteRequest{
		Prompt: "The audio is in Spanish, I want the English version",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Text, `"intent":"wrong_audio_language"`) {
		t.Fatalf("text = %s", got.Text)
	}
	if !strings.Contains(got.Text, `"language":"en"`) {
		t.Fatalf("expected english language, got %s", got.Text)
	}
	if got.Provider != infer.ProviderHeuristic {
		t.Fatalf("provider = %s", got.Provider)
	}
}

func TestHeuristicCompleteRequiresPrompt(t *testing.T) {
	_, err := HeuristicProvider{}.Complete(context.Background(), infer.CompleteRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHeuristicEmbedIsUnitLengthAndStable(t *testing.T) {
	p := HeuristicProvider{}
	a, err := p.Embed(context.Background(), infer.EmbedRequest{Text: "The Dark Knight"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.Embed(context.Background(), infer.EmbedRequest{Text: "The Dark Knight"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Embedding) != 32 {
		t.Fatalf("dim = %d", len(a.Embedding))
	}
	var sum float64
	for i := range a.Embedding {
		if a.Embedding[i] != b.Embedding[i] {
			t.Fatal("embedding not stable")
		}
		sum += a.Embedding[i] * a.Embedding[i]
	}
	if sum < 0.99 || sum > 1.01 {
		t.Fatalf("norm^2 = %f", sum)
	}
}

func TestHeuristicTranscribeSplitsHint(t *testing.T) {
	got, err := HeuristicProvider{}.Transcribe(context.Background(), infer.TranscribeRequest{
		TranscriptHint: "Hello there. How are you?",
		DurationSec:    4,
		Language:       "en",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Cues) != 2 {
		t.Fatalf("cues = %#v", got.Cues)
	}
	if got.Cues[0].Text != "Hello there" || got.Cues[1].Text != "How are you" {
		t.Fatalf("cues = %#v", got.Cues)
	}
	if got.Cues[1].EndMs != 4000 {
		t.Fatalf("end = %d", got.Cues[1].EndMs)
	}
}

func TestHeuristicClassifyHitsProfanity(t *testing.T) {
	got, err := HeuristicProvider{}.Classify(context.Background(), infer.ClassifyRequest{
		Text:   "strong profanity in this scene",
		Labels: []string{"profanity", "sexual"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Hits) != 2 || got.Hits[0].Confidence < 0.9 || got.Hits[1].Confidence != 0 {
		t.Fatalf("hits = %#v", got.Hits)
	}
}

func TestHeuristicTranslateSpanish(t *testing.T) {
	got, err := HeuristicProvider{}.Translate(context.Background(), infer.TranslateRequest{
		Text: "hello movie", Target: "es",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "hola película" {
		t.Fatalf("text = %q", got.Text)
	}
}

func TestRouterDefaultsToHeuristic(t *testing.T) {
	r := Router{Default: HeuristicProvider{}}
	p := r.pick("")
	if p.Name() != infer.ProviderHeuristic {
		t.Fatalf("name = %s", p.Name())
	}
}
