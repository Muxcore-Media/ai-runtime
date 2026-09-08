package internal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestModuleInfoMatchesManifest(t *testing.T) {
	m := NewModule()
	info := m.Info()
	if info.ID != "ai-runtime" {
		t.Fatalf("id = %s", info.ID)
	}
	if info.Capabilities[0] != "ai.runtime" {
		t.Fatalf("caps = %v", info.Capabilities)
	}
}

func TestModuleLifecycleAndCompleteHTTP(t *testing.T) {
	m := New(Config{GRPCAddr: "127.0.0.1:0", HTTPAddr: "127.0.0.1:0"})
	ctx := context.Background()
	if err := m.Init(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Stop(ctx) })

	deadline := time.Now().Add(2 * time.Second)
	var health *http.Response
	var err error
	for time.Now().Before(deadline) {
		health, err = http.Get("http://" + m.HTTPListenAddr() + "/healthz")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	_ = health.Body.Close()
	if health.StatusCode != 200 {
		t.Fatalf("health = %d", health.StatusCode)
	}

	resp, err := http.Post("http://"+m.HTTPListenAddr()+"/v1/complete", "application/json",
		strings.NewReader(`{"prompt":"the audio is in French, I want the English version"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d body %s", resp.StatusCode, raw)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	text, _ := body["text"].(string)
	if !strings.Contains(text, "wrong_audio_language") {
		t.Fatalf("body = %s", raw)
	}
}
