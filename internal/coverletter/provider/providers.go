package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	discovery "github.com/jenksed/yourmove-career-discovery"
)

const maxLetterBytes = 12000

// Provider endpoints are fixed by the application. The optional compatible
// endpoint is operator-configured at startup, never supplied by an HTTP form.
type coverBackend struct {
	Provider string
	Model    string
	URL      string
	APIKey   string

	metaDiscovery bool
	baseten       bool // HACK-BASETEN: temporary backend flag; delete on removal.
	jsonOutput     bool
	letterBudget   bool
	observe       func(RequestMetrics)
}

// HACK-BASETEN: temporary hackathon backend. Baseten hosts Muse Spark Open
// behind an OpenAI-compatible endpoint; it is a different model from Meta's
// direct Muse Spark 1.3, with its own behavior and billing. Removal restores
// the Meta-only integration; see docs/hackathon/BASETEN_TEMPORARY_BACKEND.md.
const (
	// MuseBackendMeta selects Meta's direct Muse Spark endpoint.
	MuseBackendMeta = "meta"
	// MuseBackendBaseten selects Baseten's hosted Muse Spark Open model.
	MuseBackendBaseten = "baseten"

	basetenChatURL = "https://inference.baseten.co/v1/chat/completions"
	// BasetenMuseModel is the exact restricted hackathon model identifier.
	// It is not Baseten's general public model identifier.
	BasetenMuseModel = "meta-models/muse-spark-open-1026"
)

// RequestMetrics deliberately excludes prompts, credentials and model content.
// Backend names the hackathon Muse backend that served the request ("meta" or
// "baseten") and Model carries its exact model identifier, so diagnostics and
// evaluation results preserve which model actually ran.
type RequestMetrics struct {
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	FinishReason     string `json:"finish_reason"`
	Backend          string `json:"backend"`
	Model            string `json:"model"`
}

func (r Runner) GenerateDiscovery(ctx context.Context, prompt, instructions string, observe func(RequestMetrics)) (string, error) {
	backend := r.backend
	backend.observe = observe
	return runProviderProfile(ctx, backend, prompt, instructions, nil, true)
}

var (
	errCoverAPIKey       = errors.New("provider API key is not configured")
	errCoverModel        = errors.New("local Ollama model not configured")
	errCoverModelMissing = errors.New("Ollama model not installed")
	errCoverOffline      = errors.New("local Ollama service not reachable")
	errCoverAuth         = errors.New("provider rejected API credentials")
	errCoverRate         = errors.New("provider rate limit reached")
	errCoverCredits      = errors.New("provider account has no available credits")
	errCoverTimeout      = errors.New("provider timed out")
	errCoverUnavailable  = errors.New("provider is unavailable")
	errCoverOutput       = errors.New("provider returned no usable text")
	errCoverEndpoint     = errors.New("compatible provider endpoint is missing or invalid")
	errMuseBackend       = errors.New("hackathon Muse backend is not recognized") // HACK-BASETEN: delete on removal.
)

// Runner contains validated provider configuration without exposing endpoint
// or credential details to application callers.
type Runner struct {
	provider string
	backend  coverBackend
}

// HACK-BASETEN: temporary selection helper; delete on removal.
// SelectedMuseBackend reports which hackathon Muse backend serves Career
// Discovery and related Muse workflows: "meta" by default, "baseten" when
// HACKATHON_MUSE_BACKEND=baseten. Any other value is an actionable error;
// callers must not silently use a backend the operator did not select.
func SelectedMuseBackend() (string, error) {
	raw := strings.TrimSpace(os.Getenv("HACKATHON_MUSE_BACKEND"))
	if raw == "" || strings.EqualFold(raw, MuseBackendMeta) {
		return MuseBackendMeta, nil
	}
	if strings.EqualFold(raw, MuseBackendBaseten) {
		return MuseBackendBaseten, nil
	}
	return "", fmt.Errorf("HACKATHON_MUSE_BACKEND=%q is not recognized (use %q or %q): %w", raw, MuseBackendMeta, MuseBackendBaseten, errMuseBackend)
}

// ConfigureMeta reuses the compatible transport without mutating process-wide
// provider settings. Discovery always uses standard mode, never Contributor.
// HACK-BASETEN: the baseten branch below is temporary; delete it on removal
// so this function configures Meta directly again. When
// HACKATHON_MUSE_BACKEND=baseten, Career Discovery runs against Baseten's
// hosted Muse Spark Open model instead of Meta's direct endpoint. A missing
// Baseten key is an error: selection never falls back to the other account.
func ConfigureMeta() (Runner, error) {
	backend, err := SelectedMuseBackend()
	if err != nil {
		return Runner{}, err
	}
	if backend == MuseBackendBaseten {
		key := os.Getenv("BASETEN_API_KEY")
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "\r\n") {
			return Runner{}, fmt.Errorf("BASETEN_API_KEY is not configured for the Baseten hackathon backend: %w", errCoverAPIKey)
		}
		return Runner{provider: "compatible", backend: coverBackend{
			Provider: "compatible", Model: BasetenMuseModel,
			URL: basetenChatURL, APIKey: key, metaDiscovery: true, baseten: true,
		}}, nil
	}
	key := os.Getenv("MODEL_API_KEY")
	if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "\r\n") {
		return Runner{}, errCoverAPIKey
	}
	return Runner{provider: "compatible", backend: coverBackend{
		Provider: "compatible", Model: discovery.Model,
		URL: discovery.Endpoint, APIKey: key, metaDiscovery: true,
	}}, nil
}

// Configure validates a provider/model pair and captures its startup
// configuration. Subscription CLI availability and sign-in are checked when
// Generate runs, as before.
func Configure(provider, model string) (Runner, error) {
	if !validCoverProvider(provider) {
		return Runner{}, errCoverEndpoint
	}
	if coverSubscription(provider) {
		if !validCoverModel(model) {
			return Runner{}, errCoverModel
		}
		return Runner{provider: provider, backend: coverBackend{Provider: provider, Model: model}}, nil
	}
	backend, err := configuredCoverBackend(provider, model)
	if err != nil {
		return Runner{}, err
	}
	return Runner{provider: provider, backend: backend}, nil
}

// Generate sends the supplied evidence prompt to the configured provider.
// Instructions stay application-owned and are passed only for HTTP providers
// that carry them in a separate system/instructions field.
func (r Runner) Generate(ctx context.Context, prompt, instructions string) (string, error) {
	if coverSubscription(r.provider) {
		return runCoverSubscription(ctx, r.provider, r.backend.Model, prompt)
	}
	return runCoverProvider(ctx, r.backend, prompt, instructions, nil)
}

// GenerateNarrative performs one bounded narrative request, independent of letter formatting.
func (r Runner) GenerateNarrative(ctx context.Context, prompt, instructions string) (string, error) {
	if coverSubscription(r.provider) {
		return runSubscriptionProfile(ctx, r.provider, r.backend.Model, prompt+"\n\n"+instructions, true)
	}
	if r.provider == "ollama" {
		prompt += "\n\n" + instructions
	}
	return runProviderProfile(ctx, r.backend, prompt, instructions, nil, true)
}

// GenerateNarrativeJSON requests JSON object output from an OpenAI-compatible
// endpoint for narrow workflows whose callers validate structured references.
func (r Runner) GenerateNarrativeJSON(ctx context.Context, prompt, instructions string) (string, error) {
	if r.backend.Provider == "compatible" {
		r.backend.jsonOutput = true
	}
	return r.GenerateNarrative(ctx, prompt, instructions)
}

// GenerateLetter performs one bounded cover-letter request. Letters carry
// large source prompts, so the compatible provider uses the discovery output
// budget with low reasoning effort but no JSON contract: the letter validator
// rejects structured envelopes.
func (r Runner) GenerateLetter(ctx context.Context, prompt, instructions string) (string, error) {
	if r.backend.Provider == "compatible" {
		r.backend.letterBudget = true
	}
	return r.Generate(ctx, prompt, instructions)
}

// HACK-BASETEN: temporary identity accessors; delete on removal.
// MuseBackend reports which hackathon Muse backend this Runner serves:
// "baseten" for Baseten's hosted Muse Spark Open, "meta" for Meta's direct
// endpoint, or "" for runners configured for other providers.
func (r Runner) MuseBackend() string {
	if r.backend.metaDiscovery && r.backend.baseten {
		return MuseBackendBaseten
	}
	if r.backend.metaDiscovery {
		return MuseBackendMeta
	}
	return ""
}

// MuseModel reports the exact model identifier this Muse Runner sends.
// It is empty for runners configured for other providers.
func (r Runner) MuseModel() string {
	if r.backend.metaDiscovery {
		return r.backend.Model
	}
	return ""
}

// ExecutionLocality describes the configured transport, never inference guarantees.
func (r Runner) ExecutionLocality() string {
	if coverSubscription(r.provider) {
		return "local command using remote vendor services"
	}
	if u, err := url.Parse(r.backend.URL); err == nil {
		host := u.Hostname()
		ip := net.ParseIP(host)
		if strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback()) {
			return "loopback service; may forward to remote inference"
		}
	}
	return "remote service"
}

func ValidProvider(provider string) bool { return validCoverProvider(provider) }

func ValidModel(model string) bool { return validCoverModel(model) }

func DefaultModel(provider string) string { return coverDefaultModel(provider) }

func Configured(provider string) bool { return coverProviderConfigured(provider) }

func SubscriptionReadiness(provider string) (string, string) { return coverCLIReadiness(provider) }

// ErrorNotice returns the stable application notice code for provider errors.
func ErrorNotice(err error) string { return coverErrorNotice(err) }

func validCoverProvider(provider string) bool {
	switch provider {
	case "ollama", "openai", "anthropic", "openrouter", "compatible", "codex", "claude-cli":
		return true
	}
	return false
}

func validCoverModel(model string) bool {
	return model != "" && len(model) <= 120 && !strings.ContainsAny(model, " \t\r\n\\\x00")
}

// Never derive a network destination or credential from the user-facing form.
func configuredCoverBackend(provider, model string) (coverBackend, error) {
	if !validCoverProvider(provider) {
		return coverBackend{}, errCoverEndpoint
	}
	if !validCoverModel(model) {
		return coverBackend{}, errCoverModel
	}
	config := coverBackend{Provider: provider, Model: model}
	switch provider {
	case "ollama":
		config.URL = "http://127.0.0.1:11434/api/generate"
	case "openai":
		config.URL, config.APIKey = "https://api.openai.com/v1/responses", os.Getenv("OPENAI_API_KEY")
	case "anthropic":
		config.URL, config.APIKey = "https://api.anthropic.com/v1/messages", os.Getenv("ANTHROPIC_API_KEY")
	case "openrouter":
		config.URL, config.APIKey = "https://openrouter.ai/api/v1/chat/completions", os.Getenv("OPENROUTER_API_KEY")
	case "compatible":
		config.URL, config.APIKey = os.Getenv("YOURMOVE_COMPAT_URL"), os.Getenv("YOURMOVE_COMPAT_API_KEY")
		if !validCompatibleEndpoint(config.URL) {
			return coverBackend{}, errCoverEndpoint
		}
	}
	if provider != "ollama" && (provider != "compatible" || strings.HasPrefix(config.URL, "https://")) && strings.TrimSpace(config.APIKey) == "" {
		return coverBackend{}, errCoverAPIKey
	}
	if strings.ContainsAny(config.APIKey, "\r\n") {
		return coverBackend{}, errCoverAPIKey
	}
	return config, nil
}

func validCompatibleEndpoint(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		!strings.HasSuffix(u.Path, "/chat/completions") {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	if u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func coverProviderConfigured(provider string) bool {
	if coverSubscription(provider) {
		return coverCLIInstalled(provider)
	}
	switch provider {
	case "ollama":
		return true // Connectivity/model availability is checked at generation.
	case "openai":
		return strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != ""
	case "anthropic":
		return strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) != ""
	case "openrouter":
		return strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")) != ""
	case "compatible":
		return validCompatibleEndpoint(os.Getenv("YOURMOVE_COMPAT_URL"))
	}
	return false
}

// One bounded request, no redirects, no retries (and therefore no double charge).
// Provider response and request bodies are never logged, including on failures.
func runCoverProvider(ctx context.Context, backend coverBackend, prompt, instructions string, client *http.Client) (string, error) {
	return runProviderProfile(ctx, backend, prompt, instructions, client, false)
}

func runProviderProfile(ctx context.Context, backend coverBackend, prompt, instructions string, client *http.Client, narrative bool) (string, error) {
	if !validCoverProvider(backend.Provider) || !validCoverModel(backend.Model) || backend.URL == "" {
		return "", errCoverModel
	}
	if backend.Provider != "ollama" && backend.Provider != "compatible" && backend.APIKey == "" {
		return "", errCoverAPIKey
	}
	var request any
	switch backend.Provider {
	case "ollama":
		// Disable optional reasoning before budgeting. MiniMax M3.1 Flash on
		// Ollama Cloud still reasons with think:false (verified on both APIs);
		// its reasoning and answer share num_predict. Keep other models bounded
		// as before, and allow this model the budget its measured calls need.
		budget := 850
		if backend.Model == "minimax-m3.1-flash:cloud" || backend.Model == "minimax-m3.1-flash" {
			budget = 16384
		}
		if narrative {
			budget = 1024
		}
		request = map[string]any{"model": backend.Model, "prompt": prompt, "stream": false, "think": false,
			"options": map[string]any{"temperature": 0.35, "num_predict": budget}}
	case "openai":
		// The API's output_text is an SDK convenience; parse output[] in raw HTTP.
		request = map[string]any{"model": backend.Model, "input": prompt, "instructions": instructions, "store": false, "max_output_tokens": 1400}
	case "anthropic":
		request = map[string]any{"model": backend.Model, "max_tokens": 900, "system": instructions,
			"messages": []map[string]string{{"role": "user", "content": prompt}}}
	default:
		request = map[string]any{"model": backend.Model, "stream": false, "max_tokens": 900,
			"messages": []map[string]string{{"role": "system", "content": instructions}, {"role": "user", "content": prompt}}}
	}
	if narrative {
		fields := request.(map[string]any)
		switch backend.Provider {
		case "openai":
			fields["max_output_tokens"] = 1600
		case "ollama":
		default:
			fields["max_tokens"] = 1024
		}
	}
	if backend.metaDiscovery {
		fields := request.(map[string]any)
		// Proposal-bearing turns exhausted 2048 (finish=length) in live
		// Standard and Contributor runs; discovery gets a larger budget.
		fields["max_tokens"] = 3072
		fields["reasoning_effort"] = "low"
		fields["response_format"] = map[string]string{"type": "json_object"}
	} else if narrative && backend.jsonOutput && backend.Provider == "compatible" {
		// Structured answer traces carry the answer plus verbatim claims, so
		// they use the discovery profile that already succeeds: a larger
		// output budget with low reasoning effort and a JSON object contract.
		fields := request.(map[string]any)
		fields["max_tokens"] = 2048
		fields["reasoning_effort"] = "low"
		fields["response_format"] = map[string]string{"type": "json_object"}
	} else if backend.letterBudget && backend.Provider == "compatible" {
		// Cover letters stopped on the output cap with the default budget, so
		// they use the discovery output profile without a JSON contract.
		fields := request.(map[string]any)
		fields["max_tokens"] = 2048
		fields["reasoning_effort"] = "low"
	}
	data, err := json.Marshal(request)
	if err != nil {
		return "", errCoverOutput
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, backend.URL, bytes.NewReader(data))
	if err != nil {
		return "", errCoverEndpoint
	}
	req.Header.Set("Content-Type", "application/json")
	switch backend.Provider {
	case "anthropic":
		req.Header.Set("x-api-key", backend.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case "openai", "openrouter":
		req.Header.Set("Authorization", "Bearer "+backend.APIKey)
	case "compatible":
		if backend.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+backend.APIKey)
		}
	}
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	// Redirects must never forward credential headers or sensitive source text.
	// Copying the transport lets tests supply a local recorder without an external call.
	guarded := *client
	guarded.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := guarded.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", errCoverTimeout
		}
		if backend.Provider == "ollama" {
			return "", errCoverOffline
		}
		return "", errCoverUnavailable
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return "", errCoverAuth
	case http.StatusPaymentRequired:
		return "", errCoverCredits
	case http.StatusTooManyRequests:
		return "", errCoverRate
	case http.StatusNotFound:
		return "", errCoverModelMissing
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return "", errCoverTimeout
	default:
		return "", errCoverUnavailable
	}
	limited, err := io.ReadAll(io.LimitReader(resp.Body, 128<<10+1))
	if err != nil || len(limited) > 128<<10 || (narrative && !utf8.Valid(limited)) {
		return "", errCoverOutput
	}
	var text string
	if backend.observe != nil {
		var metadata struct {
			Usage struct {
				Prompt     int `json:"prompt_tokens"`
				Completion int `json:"completion_tokens"`
				Details    struct {
					Cached int `json:"cached_tokens"`
				} `json:"prompt_tokens_details"`
			} `json:"usage"`
			Choices []struct {
				Reason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if json.Unmarshal(limited, &metadata) == nil {
			m := RequestMetrics{PromptTokens: metadata.Usage.Prompt, CompletionTokens: metadata.Usage.Completion, CachedTokens: metadata.Usage.Details.Cached}
			if len(metadata.Choices) > 0 {
				m.FinishReason = metadata.Choices[0].Reason
			}
			m.Backend, m.Model = backend.museIdentity() // HACK-BASETEN: delete on removal.
			backend.observe(m)
		}
	}
	switch backend.Provider {
	case "ollama":
		var result struct {
			Done       *bool  `json:"done"`
			DoneReason string `json:"done_reason"`
			Response   string `json:"response"`
			Error      string `json:"error"`
		}
		if json.Unmarshal(limited, &result) != nil || result.Error != "" {
			return "", errCoverOutput
		}
		if narrative && ((result.Done != nil && !*result.Done) || result.DoneReason == "length") {
			return "", errCoverOutput
		}
		text = result.Response
	case "openai":
		var result struct {
			Status string `json:"status"`
			Output []struct {
				Type    string `json:"type"`
				Status  string `json:"status"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"output"`
		}
		if json.Unmarshal(limited, &result) != nil || (result.Status != "" && result.Status != "completed") {
			return "", errCoverOutput
		}
		var parts []string
		for _, item := range result.Output {
			if narrative && item.Status != "" && item.Status != "completed" {
				return "", errCoverOutput
			}
			if item.Type != "message" {
				if narrative && item.Type != "reasoning" {
					return "", errCoverOutput
				}
				continue
			}
			for _, part := range item.Content {
				if part.Type == "output_text" {
					parts = append(parts, part.Text)
				}
			}
		}
		text = strings.Join(parts, "\n")
	case "anthropic":
		var result struct {
			StopReason string `json:"stop_reason"`
			Content    []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if json.Unmarshal(limited, &result) != nil {
			return "", errCoverOutput
		}
		if narrative && result.StopReason != "" && result.StopReason != "end_turn" && result.StopReason != "stop_sequence" {
			return "", errCoverOutput
		}
		var parts []string
		for _, part := range result.Content {
			if narrative && part.Type != "text" {
				return "", errCoverOutput
			}
			if part.Type == "text" {
				parts = append(parts, part.Text)
			}
		}
		text = strings.Join(parts, "\n")
	default:
		var result struct {
			Choices []struct {
				FinishReason string `json:"finish_reason"`
				Message      struct {
					Content      string          `json:"content"`
					ToolCalls    json.RawMessage `json:"tool_calls"`
					FunctionCall json.RawMessage `json:"function_call"`
				} `json:"message"`
			} `json:"choices"`
		}
		if json.Unmarshal(limited, &result) != nil || len(result.Choices) == 0 {
			// Metadata only: envelope shape and byte counts, never content.
			log.Printf("compatible provider output: provider=%s envelope=unusable bytes=%d", backend.Provider, len(limited))
			return "", errCoverOutput
		}
		// A completion stopped by the output cap is not a complete letter.
		// Compatible endpoints may omit finish_reason; reject known failures,
		// and require a successful stop when OpenRouter supplies a reason.
		reason := strings.ToLower(strings.TrimSpace(result.Choices[0].FinishReason))
		if reason == "length" || reason == "max_tokens" || reason == "content_filter" || reason == "tool_calls" ||
			((backend.Provider == "openrouter" || narrative) && reason != "" && reason != "stop") {
			log.Printf("compatible provider output: provider=%s finish=%s bytes=%d text_bytes=%d", backend.Provider, reason, len(limited), len(result.Choices[0].Message.Content))
			return "", errCoverOutput
		}
		if narrative && ((len(result.Choices[0].Message.ToolCalls) > 0 && string(result.Choices[0].Message.ToolCalls) != "[]" && string(result.Choices[0].Message.ToolCalls) != "null") || (len(result.Choices[0].Message.FunctionCall) > 0 && string(result.Choices[0].Message.FunctionCall) != "null")) {
			return "", errCoverOutput
		}
		text = result.Choices[0].Message.Content
	}
	min, max := 30, maxLetterBytes
	if narrative {
		min, max = 1, 6000
	}
	if len(text) < min || (narrative && strings.TrimSpace(text) == "") || len(text) > max || strings.ContainsRune(text, 0) {
		return "", errCoverOutput
	}
	return text, nil
}

// HACK-BASETEN: temporary identity helper; delete on removal.
// museIdentity names the hackathon Muse backend and exact model for observed
// requests. It is empty for runners configured for other providers.
func (b coverBackend) museIdentity() (string, string) {
	if !b.metaDiscovery {
		return "", ""
	}
	if b.baseten {
		return MuseBackendBaseten, b.Model
	}
	return MuseBackendMeta, b.Model
}

func coverErrorNotice(err error) string {
	switch {
	case errors.Is(err, errCoverModel), errors.Is(err, errCoverEndpoint):
		return "model"
	case errors.Is(err, errCoverCLI):
		return "cli"
	case errors.Is(err, errCoverLogin):
		return "login"
	case errors.Is(err, errCoverCLIFailed):
		return "cli-failed"
	case errors.Is(err, errCoverAPIKey):
		return "key"
	case errors.Is(err, errCoverAuth):
		return "auth"
	case errors.Is(err, errCoverRate):
		return "rate"
	case errors.Is(err, errCoverCredits):
		return "credits"
	case errors.Is(err, errCoverTimeout):
		return "timeout"
	case errors.Is(err, errCoverOffline):
		return "offline"
	case errors.Is(err, errCoverModelMissing):
		return "model-missing"
	case errors.Is(err, errCoverOutput):
		return "output"
	case errors.Is(err, errCoverUnavailable):
		return "provider-unavailable"
	default:
		return "failed"
	}
}

func coverDefaultModel(provider string) string {
	switch provider {
	case "codex":
		return "default"
	case "claude-cli":
		return "sonnet"
	case "openai":
		return "gpt-4.1-mini"
	case "anthropic":
		return "claude-sonnet-4-5"
	case "openrouter":
		return "openai/gpt-4o-mini"
	case "compatible":
		return strings.TrimSpace(os.Getenv("YOURMOVE_COMPAT_MODEL"))
	default:
		if s := strings.TrimSpace(os.Getenv("YOURMOVE_OLLAMA_MODEL")); s != "" {
			return s
		}
		return "qwen3.5:4b"
	}
}
