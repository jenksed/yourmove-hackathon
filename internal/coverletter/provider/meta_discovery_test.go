package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetaDiscoveryCompatibleRequestAndMetrics(t *testing.T) {
	t.Setenv("MODEL_API_KEY", "synthetic-unit-key")
	runner, err := ConfigureMeta()
	if err != nil {
		t.Fatal(err)
	}
	if runner.backend.Model != "muse-spark-1.3" || runner.backend.URL != "https://api.meta.ai/v1/chat/completions" {
		t.Fatal("wrong Meta configuration")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-unit-key" {
			t.Error("missing credential mapping")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Fatal("invalid request")
		}
		if body["reasoning_effort"] != "low" || body["max_tokens"] != float64(2048) || body["response_format"].(map[string]any)["type"] != "json_object" {
			t.Error("unbounded or invalid Muse structured request")
		}
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"{\"question\":\"What did you do?\",\"proposal\":null}"}}],"usage":{"prompt_tokens":400,"completion_tokens":1400,"prompt_tokens_details":{"cached_tokens":100}}}`)
	}))
	defer server.Close()
	backend := runner.backend
	backend.URL = server.URL
	var metrics RequestMetrics
	backend.observe = func(m RequestMetrics) { metrics = m }
	text, err := runProviderProfile(context.Background(), backend, "synthetic context", "trusted discovery policy", nil, true)
	if err != nil || text == "" || metrics.PromptTokens != 400 || metrics.CompletionTokens != 1400 || metrics.CachedTokens != 100 || metrics.FinishReason != "stop" {
		t.Fatal(text, metrics, err)
	}
}
