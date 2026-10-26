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

func seedReviewedEvidence(t *testing.T, a *App) {
	t.Helper()
	now := "2026-10-26T10:00:00Z"
	if _, err := a.db.Exec(`INSERT INTO career_evidence(title,situation,actions,outcome_state,source_notes,source_writing,status,revision,created_at,updated_at) VALUES('Slow-query diagnosis','Payroll outage with a full table scan','Checked slow query log and used EXPLAIN; suggested an index; DBA team applied the production change.','unknown','synthetic','synthetic', 'reviewed',2,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
}

func roleGenerate(t *testing.T) func(context.Context, string, string) (string, error) {
	t.Helper()
	return func(_ context.Context, payload, policy string) (string, error) {
		if !strings.Contains(policy, "ONE role") {
			t.Fatalf("wrong policy: %.60s", policy)
		}
		if strings.Contains(payload, "support-engineer") {
			return `{"items":[{"evidence_id":1,"evidence_revision":2,"relevant":true,"capabilities":["MySQL slow-query diagnosis"],"gaps":["Production change ownership stayed with the DBA team"],"explanation":"Directly supports production diagnosis and engineering handoff; does not establish account ownership."}],"unmet":["Own customer accounts and renewal targets","Manage multi-account customer relationships"]}`, nil
		}
		return `{"items":[{"evidence_id":1,"evidence_revision":2,"relevant":true,"capabilities":["Technical translation for customers"],"gaps":["No direct customer-facing record"],"explanation":"Transferable: diagnosis plus handoff supports coordination, but customer communication is inferred, not established."}],"unmet":["Own customer accounts and renewal targets","Apply production database changes directly"]}`, nil
	}
}

func TestCareerRolesCompareAndPersistAcrossRestart(t *testing.T) {
	a, dir := testWorkspace(t)
	seedReviewedEvidence(t, a)
	a.discoveryGenerate = roleGenerate(t)
	mux := newMux(a)
	for _, role := range []string{"support-engineer", "account-manager"} {
		got := postCareerEvidence(mux, "/career-roles/"+role+"/analyze", url.Values{"consent": {"yes"}})
		if got.Code != 303 {
			t.Fatal(role, got.Code, got.Body.String())
		}
	}
	page := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost/career-roles", nil)
	mux.ServeHTTP(page, req)
	body := page.Body.String()
	if page.Code != 200 || !strings.Contains(body, "supports this role") || !strings.Contains(body, "Own customer accounts and renewal targets") {
		t.Fatal(page.Code, body[:min(600, len(body))])
	}
	if strings.Count(body, "supports this role") != 2 {
		t.Fatal("both roles should show relevant experience")
	}
	if err := a.db.Close(); err != nil {
		t.Fatal(err)
	}
	var raw, hash string
	db, err := storage.OpenDB(filepath.Join(dir, "jobs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.QueryRow(`SELECT analysis,evidence_hash FROM career_role_analyses WHERE role_key='support-engineer'`).Scan(&raw, &hash); err != nil || raw == "" || hash == "" {
		t.Fatal("analysis did not survive restart", err)
	}
}

func TestCareerRolesStaleAfterCorrection(t *testing.T) {
	a, _ := testWorkspace(t)
	seedReviewedEvidence(t, a)
	a.discoveryGenerate = roleGenerate(t)
	mux := newMux(a)
	if got := postCareerEvidence(mux, "/career-roles/support-engineer/analyze", url.Values{"consent": {"yes"}}); got.Code != 303 {
		t.Fatal(got.Code)
	}
	// Candidate corrects the record: new revision, new hash.
	if _, err := a.db.Exec(`UPDATE career_evidence SET actions='Checked slow query log; suggested an index.',revision=3 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "http://localhost/career-roles", nil))
	if strings.Contains(page.Body.String(), "supports this role") {
		t.Fatal("stale analysis shown after evidence correction")
	}
	if !strings.Contains(page.Body.String(), "re-run a comparison to refresh") {
		t.Fatal("no refresh guidance after correction")
	}
}
