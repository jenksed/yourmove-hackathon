package app

import (
	"strings"
	"testing"

	coverprovider "github.com/jenksed/yourmove-go/internal/coverletter/provider"
)

func TestDiscoveryUsageNoteEmptyWithoutTokens(t *testing.T) {
	if got := discoveryUsageNote(coverprovider.RequestMetrics{}, false); got != "" {
		t.Fatalf("empty metrics note = %q", got)
	}
}

func TestDiscoveryUsageNoteMetaEstimateUnchanged(t *testing.T) {
	got := discoveryUsageNote(coverprovider.RequestMetrics{
		PromptTokens: 400, CompletionTokens: 1400, CachedTokens: 100,
		FinishReason: "stop", Backend: coverprovider.MuseBackendMeta, Model: "muse-spark-1.3",
	}, true)
	want := " Tokens: 400 input / 1400 output; estimated inference $0.00634; finish: stop."
	if got != want {
		t.Fatalf("meta note = %q, want %q", got, want)
	}
}

func TestDiscoveryUsageNoteBasetenKeepsIdentityWithoutMetaPricing(t *testing.T) {
	got := discoveryUsageNote(coverprovider.RequestMetrics{
		PromptTokens: 400, CompletionTokens: 1400,
		FinishReason: "stop", Backend: coverprovider.MuseBackendBaseten, Model: coverprovider.BasetenMuseModel,
	}, true)
	if !strings.Contains(got, "via Baseten meta-models/muse-spark-open-1026") {
		t.Fatalf("baseten note must preserve model identity: %q", got)
	}
	if strings.Contains(got, "$") || strings.Contains(got, "estimated inference") {
		t.Fatalf("baseten note must not imply Meta pricing: %q", got)
	}
}
