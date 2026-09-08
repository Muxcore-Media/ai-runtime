package internal

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Muxcore-Media/contracts-ai/infer"
)

func (m *Module) registerHTTP(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/providers", m.handleProviders)
	mux.HandleFunc("POST /v1/complete", m.handleComplete)
	mux.HandleFunc("POST /v1/embed", m.handleEmbed)
	mux.HandleFunc("POST /v1/transcribe", m.handleTranscribe)
	mux.HandleFunc("POST /v1/classify", m.handleClassify)
	mux.HandleFunc("POST /v1/translate", m.handleTranslate)
}

func (m *Module) handleProviders(w http.ResponseWriter, _ *http.Request) {
	m.cfgMu.RLock()
	hasOpenAI := m.openaiKey != ""
	pref := m.preferred
	m.cfgMu.RUnlock()
	names := []string{infer.ProviderHeuristic}
	if hasOpenAI {
		names = append(names, infer.ProviderOpenAI)
	}
	writeJSON(w, map[string]any{"providers": names, "preferred": pref})
}

func (m *Module) handleComplete(w http.ResponseWriter, r *http.Request) {
	var req infer.CompleteRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	resp, err := m.provider(req.Provider).Complete(r.Context(), req)
	writeResult(w, resp, err)
}

func (m *Module) handleEmbed(w http.ResponseWriter, r *http.Request) {
	var req infer.EmbedRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	resp, err := m.provider(req.Provider).Embed(r.Context(), req)
	writeResult(w, resp, err)
}

func (m *Module) handleTranscribe(w http.ResponseWriter, r *http.Request) {
	var req infer.TranscribeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	resp, err := m.provider(req.Provider).Transcribe(r.Context(), req)
	writeResult(w, resp, err)
}

func (m *Module) handleClassify(w http.ResponseWriter, r *http.Request) {
	var req infer.ClassifyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	resp, err := m.provider(req.Provider).Classify(r.Context(), req)
	writeResult(w, resp, err)
}

func (m *Module) handleTranslate(w http.ResponseWriter, r *http.Request) {
	var req infer.TranslateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	resp, err := m.provider(req.Provider).Translate(r.Context(), req)
	writeResult(w, resp, err)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, `{"error":"read body"}`, http.StatusBadRequest)
		return false
	}
	if err := json.Unmarshal(body, dest); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return false
	}
	return true
}

func writeResult(w http.ResponseWriter, resp any, err error) {
	if err != nil {
		http.Error(w, `{"error":`+jsonQuote(err.Error())+`}`, http.StatusBadRequest)
		return
	}
	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonQuote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `"error"`
	}
	return string(b)
}
