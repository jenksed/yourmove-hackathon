package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenksed/yourmove-go/internal/storage"
)

func TestDiscoveryConsentSourcesReviewAndRestart(t *testing.T) {
	a, dir := testWorkspace(t)
	mux := newMux(a)
	calls := 0
	a.discoveryGenerate = func(_ context.Context, prompt, policy string) (string, error) {
		calls++
		if strings.Contains(policy, "investigation leads") {
			return `{"leads":[{"id":"lead-1","title":"Database investigation","why":"May reveal diagnosis method"},{"id":"lead-2","title":"Customer support work","why":"May reveal collaboration"},{"id":"lead-3","title":"Linux troubleshooting","why":"May reveal technical method"}]}`, nil
		}
		if calls == 2 {
			return `{"question":"What did you personally do during the database investigation?","proposal":null}`, nil
		}
		if !strings.Contains(prompt, "Another team implemented") {
			t.Fatal("response absent from adaptive prompt")
		}
		return `{"question":"Do you know what the resulting performance impact was?","proposal":{"title":"MySQL diagnosis","experience":"Investigated production query performance","claim":"Reviewed MySQL slow query logs and used EXPLAIN","source_id":"response-1","quote":"I reviewed MySQL slow query logs and used EXPLAIN. Another team implemented the changes.","unresolved":["Performance impact unknown; implementation belongs to another team."]}}`, nil
	}
	values := url.Values{"resume": {"Provided Linux and database troubleshooting support."}}
	if got := postCareerEvidence(mux, "/career-discovery", values); got.Code != 400 || calls != 0 {
		t.Fatal("inference without consent")
	}
	values.Set("consent", "yes")
	if got := postCareerEvidence(mux, "/career-discovery", values); got.Code != 303 {
		t.Fatal(got.Code, got.Body.String())
	}
	p, err := a.discoveryByID(1)
	if err != nil || len(p.State.Leads) != 3 || p.State.Question != "" {
		t.Fatal(p, err)
	}
	if got := postCareerEvidence(mux, "/career-discovery/1/select", url.Values{"revision": {"2"}, "lead_id": {"lead-1"}, "consent": {"yes"}}); got.Code != 303 {
		t.Fatal(got.Code, got.Body.String())
	}
	p, err = a.discoveryByID(1)
	if err != nil || p.State.Question == "" || p.State.SelectedLeadID != "lead-1" {
		t.Fatal(p, err)
	}
	answer := "I reviewed MySQL slow query logs and used EXPLAIN. Another team implemented the changes."
	values = url.Values{"revision": {"4"}, "answer": {answer}, "consent": {"yes"}}
	if got := postCareerEvidence(mux, "/career-discovery/1/next", values); got.Code != 303 {
		t.Fatal(got.Code, got.Body.String())
	}
	if got := postCareerEvidence(mux, "/career-discovery/1/next", values); got.Code != 409 || calls != 3 {
		t.Fatal("stale response generated again")
	}
	if got := postCareerEvidence(mux, "/career-discovery/1/evidence", url.Values{"revision": {"6"}}); got.Code != 303 {
		t.Fatal(got.Code, got.Body.String())
	}
	e, err := careerEvidenceByID(a.db, 1)
	if err != nil || e.Status != "draft" || e.OutcomeState != "unknown" || !strings.Contains(e.SourceWriting, answer) || !strings.Contains(e.SourceNotes, "response-1") {
		t.Fatal(e, err)
	}
	form := careerEvidenceForm(e.Revision, e.content())
	form.Set("actions", "Diagnosed slow queries; another team implemented changes.")
	if got := postCareerEvidence(mux, "/career-evidence/1/review", form); got.Code != 400 {
		t.Fatal("review accepted without confirmation")
	}
	form.Set("confirm_review", "yes")
	if got := postCareerEvidence(mux, "/career-evidence/1/review", form); got.Code != 303 {
		t.Fatal(got.Code, got.Body.String())
	}
	if err := a.db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := storage.OpenDB(filepath.Join(dir, "jobs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restarted := newTestApp(t, db, filepath.Join(dir, "resumes"))
	e, err = careerEvidenceByID(db, 1)
	if err != nil || e.Status != "reviewed" || !strings.Contains(e.Actions, "another team") || !strings.Contains(e.SourceWriting, answer) {
		t.Fatal(e, err)
	}
	p, err = restarted.discoveryByID(1)
	if err != nil || p.EvidenceID != 1 || p.State.Responses[0].Source.Text != answer {
		t.Fatal(p, err)
	}
	page := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost/career-discovery/1", nil)
	newMux(restarted).ServeHTTP(page, req)
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Review saved evidence") {
		t.Fatal(page.Code, page.Body.String())
	}
}

func TestDiscoveryFailureRetainsCandidateInput(t *testing.T) {
	a, _ := testWorkspace(t)
	mux := newMux(a)
	a.discoveryGenerate = func(_ context.Context, _, policy string) (string, error) {
		if strings.Contains(policy, "investigation leads") {
			return `{"leads":[{"id":"lead-1","title":"Database work","why":"May reveal method"},{"id":"lead-2","title":"Customer work","why":"May reveal collaboration"},{"id":"lead-3","title":"Linux work","why":"May reveal diagnosis"}]}`, nil
		}
		return `{"question":"What did you investigate?","proposal":null}`, nil
	}
	postCareerEvidence(mux, "/career-discovery", url.Values{"resume": {"Database support"}, "consent": {"yes"}})
	postCareerEvidence(mux, "/career-discovery/1/select", url.Values{"revision": {"2"}, "lead_id": {"lead-1"}, "consent": {"yes"}})
	a.discoveryGenerate = func(context.Context, string, string) (string, error) { return "", errors.New("provider unavailable") }
	postCareerEvidence(mux, "/career-discovery/1/next", url.Values{"revision": {"4"}, "answer": {"I reviewed logs but did not implement the fix."}, "consent": {"yes"}})
	p, err := a.discoveryByID(1)
	if err != nil || len(p.State.Responses) != 1 || p.State.Question != "" || p.State.Proposal != nil || !strings.Contains(p.Notice, "input is saved") {
		t.Fatal(p, err)
	}
	var n int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM career_evidence`).Scan(&n); err != nil || n != 0 {
		t.Fatal("failure created evidence", err)
	}
}

func TestDiscoveryBusyRequestDoesNotDispatchOrCreateSession(t *testing.T) {
	a, _ := testWorkspace(t)
	a.discoveryGenerate = func(context.Context, string, string) (string, error) {
		t.Fatal("concurrent charged request dispatched")
		return "", nil
	}
	a.discoveryMu.Lock()
	defer a.discoveryMu.Unlock()
	got := postCareerEvidence(newMux(a), "/career-discovery", url.Values{"resume": {"Database support"}, "consent": {"yes"}})
	if got.Code != http.StatusConflict {
		t.Fatal(got.Code, got.Body.String())
	}
	var n int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM career_discovery`).Scan(&n); err != nil || n != 0 {
		t.Fatal("busy request created session", err)
	}
}

func TestDiscoveryRetryRejectsUnexpectedAnswer(t *testing.T) {
	a, _ := testWorkspace(t)
	mux := newMux(a)
	calls := 0
	a.discoveryGenerate = func(context.Context, string, string) (string, error) { calls++; return "", errors.New("unavailable") }
	postCareerEvidence(mux, "/career-discovery", url.Values{"resume": {"Database support"}, "consent": {"yes"}})
	got := postCareerEvidence(mux, "/career-discovery/1/next", url.Values{"revision": {"2"}, "answer": {"Do not drop this answer"}, "consent": {"yes"}})
	if got.Code != 400 || calls != 1 {
		t.Fatal("retry silently discarded text or sent another request", got.Code, calls)
	}
}

func TestDiscoveryDraftRechecksStoredSource(t *testing.T) {
	a, _ := testWorkspace(t)
	mux := newMux(a)
	raw := `{"resume":{"id":"resume","kind":"document_excerpt","text":"Database support"},"responses":[{"question":"What did you do?","source":{"id":"response-1","kind":"self_report","text":"I reviewed logs."}}],"proposal":{"title":"Investigation","experience":"A slow database","claim":"Reviewed logs","source_id":"response-99","quote":"I reviewed logs.","unresolved":[]}}`
	if _, err := a.db.Exec(`INSERT INTO career_discovery(state,created_at) VALUES(?,'synthetic')`, raw); err != nil {
		t.Fatal(err)
	}
	got := postCareerEvidence(mux, "/career-discovery/1/evidence", url.Values{"revision": {"1"}})
	if got.Code != 422 {
		t.Fatal("unresolved source persisted", got.Code, got.Body.String())
	}
	var n int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM career_evidence`).Scan(&n); err != nil || n != 0 {
		t.Fatal("invalid source created evidence", err)
	}
}
