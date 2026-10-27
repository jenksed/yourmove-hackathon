package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenksed/yourmove-go/internal/storage"
)

func seedSecondReviewed(t *testing.T, a *App) {
	t.Helper()
	now := "2026-10-26T10:00:00Z"
	if _, err := a.db.Exec(`INSERT INTO career_evidence(title,situation,actions,outcome_state,source_notes,source_writing,status,revision,created_at,updated_at) VALUES('Billing escalation','Customer could not access invoices after a plan change','Explained the billing cause to the customer and coordinated a technical handoff to billing engineering; did not own the account.','unknown','synthetic','synthetic','reviewed',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
}

func connGenerate(_ context.Context, _, _ string) (string, error) {
	return `{"connections":[{"id":"conn-1","capability":"Cross-functional technical problem-solving","interpretation":"Both records show diagnosis followed by a documented handoff.","relationship":"Diagnosis plus handoff recurs across database and billing contexts.","supports":[{"evidence_id":1,"evidence_revision":2,"quote":"Checked slow query log and used EXPLAIN","action":"Diagnosed the slow query"},{"evidence_id":2,"evidence_revision":1,"quote":"coordinated a technical handoff to billing engineering","action":"Coordinated the handoff"}],"limitations":["Production change ownership stayed with other teams","Renewal ownership is not established"],"candidate_use":"I can credibly say I diagnose problems and move them across team boundaries with documented handoffs","follow_up":"Did recipients confirm your findings were acted on?"}]}`, nil
}

func TestStudioSynthesisChallengeStaleAndRestart(t *testing.T) {
	a, dir := testWorkspace(t)
	seedReviewedEvidence(t, a)
	seedSecondReviewed(t, a)
	a.discoveryGenerate = connGenerate
	mux := newMux(a)

	if got := postCareerEvidence(mux, "/career-connections/synthesize", url.Values{"consent": {"yes"}}); got.Code != 303 {
		t.Fatal("synthesize", got.Code, got.Body.String())
	}
	get := func(path string) string {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://localhost"+path, nil))
		if rec.Code != 200 {
			t.Fatal(path, rec.Code, rec.Body.String()[:400])
		}
		return rec.Body.String()
	}
	body := get("/career-studio")
	for _, want := range []string{"Cross-functional technical problem-solving", "Checked slow query log and used EXPLAIN", "What this does not establish", "AI hypothesis", "Challenge this interpretation"} {
		if !strings.Contains(body, want) {
			t.Fatal("studio missing", want)
		}
	}
	// Challenge persists.
	if got := postCareerEvidence(mux, "/career-connections/conn-1/challenge", url.Values{"state": {"unsupported"}, "note": {"That was mostly the DBA"}}); got.Code != 303 {
		t.Fatal("challenge", got.Code, got.Body.String())
	}
	if body = get("/career-studio"); !strings.Contains(body, "marked this interpretation") || !strings.Contains(body, "unsupported") {
		t.Fatal("challenge not shown", body[:600])
	}
	// Correction (new revision) marks dependent interpretation stale; evidence itself untouched by challenge.
	if _, err := a.db.Exec(`UPDATE career_evidence SET actions='Checked slow query log; suggested an index.',revision=3 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if body = get("/career-studio"); !strings.Contains(body, "Supporting evidence changed") {
		t.Fatal("stale state not shown after correction")
	}
	// Synthesis requires consent naming the actual destination.
	if got := postCareerEvidence(mux, "/career-connections/synthesize", url.Values{}); got.Code != 400 || !strings.Contains(got.Body.String(), "Authorize sending this to Meta") {
		t.Fatal("consent gate", got.Code, got.Body.String())
	}
	// Restart: synthesis + challenge survive in SQLite.
	if err := a.db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := storage.OpenDB(filepath.Join(dir, "jobs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var syn, state string
	if err := db.QueryRow(`SELECT synthesis FROM career_connections WHERE id=1`).Scan(&syn); err != nil || !strings.Contains(syn, "Cross-functional") {
		t.Fatal("synthesis did not survive restart", err)
	}
	if err := db.QueryRow(`SELECT state FROM career_connection_challenges WHERE connection_id='conn-1'`).Scan(&state); err != nil || state != "unsupported" {
		t.Fatal("challenge did not survive restart", state, err)
	}
}

func TestStudioKeepsSavedSynthesisVisibleAfterDraftDemotion(t *testing.T) {
	a, _ := testWorkspace(t)
	seedReviewedEvidence(t, a)
	seedSecondReviewed(t, a)
	a.discoveryGenerate = connGenerate
	mux := newMux(a)

	if got := postCareerEvidence(mux, "/career-connections/synthesize", url.Values{"consent": {"yes"}}); got.Code != 303 {
		t.Fatal("synthesize", got.Code, got.Body.String())
	}
	if got := postCareerEvidence(mux, "/career-connections/conn-1/challenge", url.Values{"state": {"questioned"}, "note": {"motive goes too far"}}); got.Code != 303 {
		t.Fatal("challenge", got.Code, got.Body.String())
	}
	// A candidate correction returns the record to draft, leaving one
	// reviewed record. The saved synthesis and challenge must stay visible
	// with a stale warning instead of disappearing behind the two-record gate.
	if _, err := a.db.Exec(`UPDATE career_evidence SET status='draft',revision=3 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://localhost/career-studio", nil))
	if rec.Code != 200 {
		t.Fatal("studio", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Cross-functional technical problem-solving", "motive goes too far", "Supporting evidence changed", "A previous synthesis is shown below"} {
		if !strings.Contains(body, want) {
			t.Fatal("studio hid demoted synthesis piece", want)
		}
	}
}

// TestCareerMirrorShowsDiscoveryAndRestrainedAbstention covers both Bet C
// acceptance outcomes from the same Studio data: a useful discovery the
// candidate can inspect (sources, use, limits, challenge link) and an honest
// no-connection result that presents nothing as a discovery.
func TestCareerMirrorShowsDiscoveryAndRestrainedAbstention(t *testing.T) {
	a, _ := testWorkspace(t)
	seedReviewedEvidence(t, a)
	seedSecondReviewed(t, a)
	a.discoveryGenerate = connGenerate
	mux := newMux(a)
	if got := postCareerEvidence(mux, "/career-connections/synthesize", url.Values{"consent": {"yes"}}); got.Code != 303 {
		t.Fatal("synthesize", got.Code, got.Body.String())
	}
	studio := func() string {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://localhost/career-studio", nil))
		if rec.Code != 200 {
			t.Fatal("studio", rec.Code)
		}
		return rec.Body.String()
	}
	body := studio()
	for _, want := range []string{
		"Career Mirror",
		"What your reviewed records establish",
		"What we uncovered across them",
		"What this could mean for your applications",
		"I can credibly say I diagnose problems",
		"What still is not established",
		"Production change ownership stayed with other teams",
		`href="#connection-conn-1"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatal("mirror missing discovery piece", want)
		}
	}
	// Restrained case: re-run synthesis with no defensible connection.
	a.discoveryGenerate = func(context.Context, string, string) (string, error) {
		return `{"connections":[]}`, nil
	}
	if got := postCareerEvidence(mux, "/career-connections/synthesize", url.Values{"consent": {"yes"}}); got.Code != 303 {
		t.Fatal("re-synthesize", got.Code, got.Body.String())
	}
	body = studio()
	if !strings.Contains(body, "found no defensible cross-experience connection, so nothing is presented as a discovery") {
		t.Fatal("mirror missing restrained outcome")
	}
	if strings.Contains(body, "What we uncovered across them") {
		t.Fatal("mirror presents a discovery after abstention")
	}
}

func TestStudioRequiresTwoReviewed(t *testing.T) {
	a, _ := testWorkspace(t)
	seedReviewedEvidence(t, a)
	a.discoveryGenerate = connGenerate
	mux := newMux(a)
	if got := postCareerEvidence(mux, "/career-connections/synthesize", url.Values{"consent": {"yes"}}); got.Code != 422 {
		t.Fatal("expected 422 with one reviewed record", got.Code)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://localhost/career-studio", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "at least two reviewed") {
		t.Fatal("studio gate", rec.Code)
	}
}
