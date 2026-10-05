package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/Muxcore-Media/contracts-ai/infer"
	"github.com/Muxcore-Media/core/sdk/go/module/netguard"
)

func TestGuardOutboundURL(t *testing.T) {
	if err := guardOutboundURL(""); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		"https://api.openai.com/v1/chat/completions",
		"http://127.0.0.1:11434/v1/chat/completions",
		"http://ollama:11434/v1/chat/completions",
	} {
		if err := guardOutboundURL(raw); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
	}
	for _, raw := range []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/",
		"file:///etc/passwd",
	} {
		if err := guardOutboundURL(raw); !errors.Is(err, netguard.ErrBlocked) {
			t.Fatalf("%s: err=%v", raw, err)
		}
	}
}

func TestCompleteRejectsMetadataBase(t *testing.T) {
	p := OpenAIProvider{BaseURL: "http://169.254.169.254", APIKey: "k"}
	_, err := p.Complete(context.Background(), infer.CompleteRequest{Prompt: "hi"})
	if !errors.Is(err, netguard.ErrBlocked) {
		t.Fatalf("err=%v", err)
	}
}
