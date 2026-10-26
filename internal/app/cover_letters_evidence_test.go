package app

import (
	"bytes"
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func testLetterEvidenceContent() careerEvidenceContent {
	return careerEvidenceContent{
		Title:        "MySQL slow query diagnosis without implementation",
		Situation:    "Investigated a production MySQL database responding slowly as diagnostic work.",
		Actions:      "Reviewed slow query logs and used EXPLAIN to identify inefficient queries. Another team implemented the changes.",
		OutcomeState: "unknown",
	}
}

func setupEvidenceLetterJob(t *testing.T) (*App, CareerEvidence) {
	t.Helper()
	a, _ := testWorkspace(t)
	if err := a.saveJob(Job{URL: "https://example.com/role", Host: "example.com", Title: "Support Engineer", Company: "Example", Description: "Work on Linux incidents and MySQL queries"}); err != nil {
		t.Fatal(err)
	}
	master, err := a.importResume("master.docx", bytes.NewReader(sampleDOCXForPreview(t, "Managed production incidents")))
	if err != nil {
		t.Fatal(err)
	}
	role, err := a.importResume("role.docx", bytes.NewReader(sampleDOCXForPreview(t, "Linux and MySQL troubleshooting")))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.setMasterResume(master.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := a.assignResume(1, role.ID); err != nil {
		t.Fatal(err)
	}
	return a, addCareerEvidenceForSearch(t, a, testLetterEvidenceContent(), true)
}

func TestCoverLetterEvidenceGroundsPromptAndPinsRefs(t *testing.T) {
	a, evidence := setupEvidenceLetterJob(t)
	var prompt string
	a.coverLetterGenerate = func(_ context.Context, p string) (string, error) {
		prompt = p
		return "I investigated your slow production MySQL database work and would bring that diagnostic experience to this support role.", nil
	}
	identity := letterEvidenceRefIdentity(letterEvidenceRef{ID: evidence.ID, Revision: evidence.Revision, CreatedAt: evidence.CreatedAt})
	rec := postApp(newMux(a), "/jobs/1/cover-letter/generate", url.Values{
		"provider":         {"compatible"},
		"model":            {"muse-spark-1.3-contributor"},
		"career_evidence":  {identity},
		"confirm_external": {"yes"},
	}, "", "")
	if rec.Code != 303 || !strings.Contains(rec.Header().Get("Location"), "cover=generated") {
		t.Fatalf("generate with evidence: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	for _, phrase := range []string{
		evidence.Title, evidence.Situation, evidence.Actions, "unknown (not established as a result)",
		"Preserve investigation versus implementation", "END OF SOURCE MATERIAL.",
	} {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("prompt missing %q", phrase)
		}
	}
	if got := strings.Index(prompt, "Evidence: "+evidence.Title); got < 0 || got > strings.Index(prompt, "END OF SOURCE MATERIAL.") {
		t.Error("evidence block must stay source material above the END marker")
	}
	if !strings.HasSuffix(prompt, a.resources.coverLetterInstructions) {
		t.Error("evidence must not disturb the canonical instruction tail")
	}
	draft, err := a.latestCoverLetter(1)
	if err != nil {
		t.Fatal(err)
	}
	refs, stale := a.coverLetterEvidenceStatus(draft)
	if stale || len(refs) != 1 || refs[0].ID != evidence.ID || refs[0].Revision != evidence.Revision {
		t.Fatalf("draft refs not pinned: %+v stale=%v", refs, stale)
	}
	page := httptest.NewRecorder()
	newMux(a).ServeHTTP(page, httptest.NewRequest("GET", "http://localhost/jobs/1", nil))
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Supporting evidence:") || !strings.Contains(page.Body.String(), evidence.Title) {
		t.Fatalf("draft evidence refs not displayed: %d", page.Code)
	}
}

func TestCoverLetterEvidenceRejectsUnreviewedChangedOrExtra(t *testing.T) {
	a, evidence := setupEvidenceLetterJob(t)
	draftEvidence := addCareerEvidenceForSearch(t, a, testLetterEvidenceContent(), false)
	calls := 0
	a.coverLetterGenerate = func(_ context.Context, p string) (string, error) {
		calls++
		return "I would bring production MySQL diagnostic experience to this support role.", nil
	}
	identity := letterEvidenceRefIdentity(letterEvidenceRef{ID: evidence.ID, Revision: evidence.Revision, CreatedAt: evidence.CreatedAt})
	mux := newMux(a)
	before, err := a.listCoverLetters(1)
	if err != nil {
		t.Fatal(err)
	}
	for name, values := range map[string]url.Values{
		"unreviewed": {"provider": {"compatible"}, "model": {"m"}, "career_evidence": {letterEvidenceRefIdentity(letterEvidenceRef{ID: draftEvidence.ID, Revision: draftEvidence.Revision, CreatedAt: draftEvidence.CreatedAt})}, "confirm_external": {"yes"}},
		"tampered":   {"provider": {"compatible"}, "model": {"m"}, "career_evidence": {letterEvidenceRefIdentity(letterEvidenceRef{ID: evidence.ID, Revision: evidence.Revision + 1, CreatedAt: evidence.CreatedAt})}, "confirm_external": {"yes"}},
		"extra":      {"provider": {"compatible"}, "model": {"m"}, "career_evidence": {identity, identity, "9:1:2026-01-01T00:00:00Z"}, "confirm_external": {"yes"}},
	} {
		rec := postApp(mux, "/jobs/1/cover-letter/generate", values, "", "")
		if rec.Code != 303 || rec.Header().Get("Location") != coverNoticeURL(1, "sources") {
			t.Fatalf("%s: %d %q", name, rec.Code, rec.Header().Get("Location"))
		}
	}
	if calls != 0 {
		t.Fatalf("rejected evidence reached generation %d times", calls)
	}
	after, err := a.listCoverLetters(1)
	if err != nil || len(after) != len(before) {
		t.Fatalf("rejected evidence created drafts: %+v %v", after, err)
	}
}

func TestCoverLetterEvidenceStalenessFlagsChangedRecords(t *testing.T) {
	a, evidence := setupEvidenceLetterJob(t)
	a.coverLetterGenerate = func(_ context.Context, p string) (string, error) {
		return "I investigated slow production MySQL queries and handed diagnostic findings to another team for implementation.", nil
	}
	identity := letterEvidenceRefIdentity(letterEvidenceRef{ID: evidence.ID, Revision: evidence.Revision, CreatedAt: evidence.CreatedAt})
	mux := newMux(a)
	if rec := postApp(mux, "/jobs/1/cover-letter/generate", url.Values{
		"provider": {"compatible"}, "model": {"m"}, "career_evidence": {identity}, "confirm_external": {"yes"},
	}, "", ""); rec.Code != 303 {
		t.Fatalf("generate: %d", rec.Code)
	}
	draft, err := a.latestCoverLetter(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, stale := a.coverLetterEvidenceStatus(draft); stale {
		t.Fatal("fresh refs reported stale")
	}
	content := testLetterEvidenceContent()
	content.Actions += " Corrected to clarify handoff ownership."
	if err := a.updateCareerEvidence(evidence.ID, evidence.Revision, content, false); err != nil {
		t.Fatal(err)
	}
	refs, stale := a.coverLetterEvidenceStatus(draft)
	if !stale || len(refs) != 1 {
		t.Fatalf("changed record not flagged: %+v stale=%v", refs, stale)
	}
	if body := draft.Body; !strings.Contains(body, "slow production MySQL") {
		t.Fatalf("flagged draft was rewritten: %q", body)
	}
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, httptest.NewRequest("GET", "http://localhost/jobs/1", nil))
	if page.Code != 200 || !strings.Contains(page.Body.String(), "A supporting record changed since this draft") {
		t.Fatalf("stale notice missing: %d", page.Code)
	}
}

func TestCoverLetterOutputFailureStages(t *testing.T) {
	valid := "Dear Hiring Team, I investigated slow production MySQL queries and handed my diagnostic findings to another team. I would bring that diagnostic discipline to this support role. Sincerely,"
	cases := map[string]string{
		"letter_length":    "Too short.",
		"letter_format":    `{"letter": "this structured object is not letter prose"}`,
		"letter_preamble":   "Here is a summary of my background. " + valid,
		"letter_commentary": "Let me think about the best structure. " + valid,
		"letter_structure": valid + "\n- investigated slow queries\n- handed off findings",
	}
	for want, body := range cases {
		if err := validateGeneratedLetter(body); err == nil {
			t.Fatalf("%s: invalid body accepted", want)
		}
		if stage, detail := classifyLetterOutput(body); stage != want || detail == "" || strings.Contains(detail, body) {
			t.Fatalf("%s: stage=%q detail=%q", want, stage, detail)
		}
	}
	if err := validateGeneratedLetter(valid); err != nil {
		t.Fatalf("valid letter rejected: %v", err)
	}
	if stage, _ := classifyLetterOutput(valid); stage != "letter_output" {
		t.Fatalf("valid letter staged as %q", stage)
	}
}

func TestCoverLetterWithoutEvidenceLeavesRefsEmpty(t *testing.T) {
	a, _ := setupEvidenceLetterJob(t)
	a.coverLetterGenerate = func(_ context.Context, p string) (string, error) {
		return "I have worked on Linux troubleshooting and production support for several years.", nil
	}
	if rec := postApp(newMux(a), "/jobs/1/cover-letter/generate", url.Values{
		"provider": {"ollama"}, "model": {"qwen3.5:4b"}, "confirm_local": {"yes"},
	}, "", ""); rec.Code != 303 {
		t.Fatalf("plain generate: %d", rec.Code)
	}
	draft, err := a.latestCoverLetter(1)
	if err != nil || draft.EvidenceRefs != "" {
		t.Fatalf("plain draft carries refs: %+v %v", draft, err)
	}
	if _, stale := a.coverLetterEvidenceStatus(draft); stale {
		t.Fatal("plain draft reported stale")
	}
}
