package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMuseBackendDefaultIsMeta(t *testing.T) {
	t.Setenv("HACKATHON_MUSE_BACKEND", "")
	t.Setenv("MODEL_API_KEY", "synthetic-unit-key")
	t.Setenv("BASETEN_API_KEY", "")
	runner, err := ConfigureMeta()
	if err != nil {
		t.Fatal(err)
	}
	if runner.MuseBackend() != MuseBackendMeta {
		t.Fatalf("default backend = %q, want meta", runner.MuseBackend())
	}
	if runner.backend.URL != "https://api.meta.ai/v1/chat/completions" || runner.backend.Model != "muse-spark-1.3" {
		t.Fatal("default Meta route changed")
	}
	if runner.MuseModel() != "muse-spark-1.3" {
		t.Fatalf("default model identity = %q", runner.MuseModel())
	}
}

func TestMuseBackendExplicitMeta(t *testing.T) {
	t.Setenv("HACKATHON_MUSE_BACKEND", "meta")
	t.Setenv("MODEL_API_KEY", "synthetic-unit-key")
	t.Setenv("BASETEN_API_KEY", "")
	runner, err := ConfigureMeta()
	if err != nil {
		t.Fatal(err)
	}
	if runner.MuseBackend() != MuseBackendMeta || runner.backend.URL != "https://api.meta.ai/v1/chat/completions" {
		t.Fatalf("explicit meta misconfigured: %+v", runner.backend)
	}
}

func TestMuseBackendBasetenExactEndpointModel(t *testing.T) {
	t.Setenv("HACKATHON_MUSE_BACKEND", "baseten")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("BASETEN_API_KEY", "synthetic-baseten-key")
	runner, err := ConfigureMeta()
	if err != nil {
		t.Fatal(err)
	}
	if runner.MuseBackend() != MuseBackendBaseten {
		t.Fatalf("backend = %q, want baseten", runner.MuseBackend())
	}
	if runner.backend.URL != "https://inference.baseten.co/v1/chat/completions" {
		t.Fatalf("endpoint = %q", runner.backend.URL)
	}
	if runner.backend.Model != "meta-models/muse-spark-open-1026" || runner.MuseModel() != BasetenMuseModel {
		t.Fatalf("model = %q, want the restricted hackathon identifier", runner.backend.Model)
	}
}

func TestMuseBackendBasetenSelectionCaseInsensitive(t *testing.T) {
	t.Setenv("HACKATHON_MUSE_BACKEND", "  BaseTen ")
	t.Setenv("BASETEN_API_KEY", "synthetic-baseten-key")
	runner, err := ConfigureMeta()
	if err != nil {
		t.Fatal(err)
	}
	if runner.MuseBackend() != MuseBackendBaseten {
		t.Fatalf("backend = %q, want baseten", runner.MuseBackend())
	}
}

func TestMuseBackendBasetenMissingKeyNeverFallsBack(t *testing.T) {
	t.Setenv("HACKATHON_MUSE_BACKEND", "baseten")
	t.Setenv("MODEL_API_KEY", "synthetic-meta-key-present")
	t.Setenv("BASETEN_API_KEY", "")
	if _, err := ConfigureMeta(); !errors.Is(err, errCoverAPIKey) {
		t.Fatalf("missing Baseten key must be an actionable key error, got %v", err)
	}
}

func TestMuseBackendUnrecognizedValueErrors(t *testing.T) {
	t.Setenv("HACKATHON_MUSE_BACKEND", "openai")
	t.Setenv("MODEL_API_KEY", "synthetic-unit-key")
	if _, err := ConfigureMeta(); !errors.Is(err, errMuseBackend) {
		t.Fatalf("unrecognized backend must error, got %v", err)
	}
}

func TestBasetenDiscoveryRequestAuthModelAndMetrics(t *testing.T) {
	t.Setenv("HACKATHON_MUSE_BACKEND", "baseten")
	t.Setenv("BASETEN_API_KEY", "synthetic-baseten-key")
	runner, err := ConfigureMeta()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer synthetic-baseten-key" {
			t.Error("Baseten credential not mapped to Bearer auth")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Fatal("invalid request")
		}
		if body["model"] != "meta-models/muse-spark-open-1026" {
			t.Errorf("model = %v", body["model"])
		}
		if body["reasoning_effort"] != "low" || body["max_tokens"] != float64(3072) || body["response_format"].(map[string]any)["type"] != "json_object" {
			t.Error("Baseten discovery request lost the bounded structured profile")
		}
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"{\"question\":\"What did you decide?\",\"proposal\":null}"}}],"usage":{"prompt_tokens":120,"completion_tokens":60,"prompt_tokens_details":{"cached_tokens":10}}}`)
	}))
	defer server.Close()
	backend := runner.backend
	backend.URL = server.URL + "/v1/chat/completions"
	var metrics RequestMetrics
	backend.observe = func(m RequestMetrics) { metrics = m }
	text, err := runProviderProfile(context.Background(), backend, "synthetic context", "trusted discovery policy", nil, true)
	if err != nil || text == "" {
		t.Fatal(text, err)
	}
	if metrics.Backend != MuseBackendBaseten || metrics.Model != BasetenMuseModel {
		t.Fatalf("metrics identity = %+v, want baseten model identity", metrics)
	}
	if metrics.PromptTokens != 120 || metrics.CompletionTokens != 60 || metrics.FinishReason != "stop" {
		t.Fatalf("metrics = %+v", metrics)
	}
}

func TestBasetenFailureNeverFallsBackToMeta(t *testing.T) {
	t.Setenv("HACKATHON_MUSE_BACKEND", "baseten")
	t.Setenv("BASETEN_API_KEY", "synthetic-baseten-key")
	runner, err := ConfigureMeta()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "synthetic failure", http.StatusBadGateway)
	}))
	defer server.Close()
	backend := runner.backend
	backend.URL = server.URL + "/v1/chat/completions"
	if _, err := runProviderProfile(context.Background(), backend, "synthetic context", "trusted discovery policy", nil, true); !errors.Is(err, errCoverUnavailable) {
		t.Fatalf("Baseten failure must surface, got %v", err)
	}
}
