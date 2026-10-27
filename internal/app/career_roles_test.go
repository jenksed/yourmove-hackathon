package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
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

func seedFictionalJob(t *testing.T, a *App, title, company, description string) int64 {
	t.Helper()
	res, err := insertJob(a.db, Job{URL: "https://example.test/jobs/" + strings.ReplaceAll(strings.ToLower(title), " ", "-"), Host: "example.test", Title: title, Company: company, Description: description, SourceStatus: "unsupported", SourceNote: "Fictional fixture posting for local testing; never submitted."})
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// TestSavedJobComparisonUsesActualRequirements is the Bet B vertical slice:
// two fictional saved jobs from different domains compare through the same
// AnalyzeRole engine path as the hardcoded demo roles, with differentiated
// relevance and honest unmet requirements echoed from each job's own bullets.
func TestSavedJobComparisonUsesActualRequirements(t *testing.T) {
	a, _ := testWorkspace(t)
	seedReviewedEvidence(t, a)
	lpn := seedFictionalJob(t, a, "Long-Term Care LPN", "Copperleaf Nursing Cooperative",
		"Provide authorized practical-nursing care.\n- Current relevant practical nursing credential\n- Documented handoffs\n- Escalate resident status changes")
	coords := seedFictionalJob(t, a, "Logistics Coordinator", "Example Freight Co.",
		"Coordinate inbound freight and track exceptions.\n- Coordinate inbound freight schedules\n- Track shipment exceptions in the shared log\n- Current forklift certification")
	a.discoveryGenerate = func(_ context.Context, payload, _ string) (string, error) {
		if strings.Contains(payload, "practical nursing") {
			return `{"items":[{"evidence_id":1,"evidence_revision":2,"relevant":false,"capabilities":[],"gaps":["No nursing credential or resident-care record"],"explanation":"Database diagnosis does not establish practical-nursing care or credential."}],"unmet":["Current relevant practical nursing credential","Documented handoffs","Escalate resident status changes"]}`, nil
		}
		if strings.Contains(payload, "forklift") {
			return `{"items":[{"evidence_id":1,"evidence_revision":2,"relevant":true,"capabilities":["Exception tracking across systems"],"gaps":["No forklift certification or freight scheduling record"],"explanation":"Transferable: diagnosis plus status-thread updates support exception tracking; freight and certification are not established."}],"unmet":["Coordinate inbound freight schedules","Current forklift certification"]}`, nil
		}
		t.Fatal("stub saw neither saved-job profile")
		return "", nil
	}
	mux := newMux(a)
	if got := postCareerEvidence(mux, "/career-roles/saved-job-"+strconv.FormatInt(lpn, 10)+"/analyze", url.Values{"consent": {"yes"}}); got.Code != 303 {
		t.Fatal("lpn analyze", got.Code, got.Body.String())
	}
	if got := postCareerEvidence(mux, "/career-roles/saved-job-"+strconv.FormatInt(coords, 10)+"/analyze", url.Values{"consent": {"yes"}}); got.Code != 303 {
		t.Fatal("coordinator analyze", got.Code, got.Body.String())
	}
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "http://localhost/career-roles", nil))
	body := page.Body.String()
	for _, want := range []string{
		"Long-Term Care LPN", "Logistics Coordinator",
		"Current relevant practical nursing credential — no approved experience supports this yet.",
		"Current forklift certification — no approved experience supports this yet.",
		"Exception tracking across systems",
	} {
		if !strings.Contains(body, want) {
			t.Fatal("roles page missing", want)
		}
	}
	if strings.Count(body, "supports this role") != 1 {
		t.Fatal("exactly one job should show relevant experience")
	}
}

// TestSavedJobComparisonRefusesProseOnlyDescription guards the no-invention
// boundary: a description without requirement bullets cannot be compared,
// and the roles page omits it instead of showing a fabricated profile.
func TestSavedJobComparisonRefusesProseOnlyDescription(t *testing.T) {
	a, _ := testWorkspace(t)
	seedReviewedEvidence(t, a)
	id := seedFictionalJob(t, a, "Vague Opportunity", "Example Co.",
		"Join our fast-paced team for exciting growth. Competitive culture, big ambitions, apply today.")
	mux := newMux(a)
	if got := postCareerEvidence(mux, "/career-roles/saved-job-"+strconv.FormatInt(id, 10)+"/analyze", url.Values{"consent": {"yes"}}); got.Code != 422 {
		t.Fatal("prose-only job should be refused", got.Code, got.Body.String())
	}
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "http://localhost/career-roles", nil))
	if strings.Contains(page.Body.String(), "Vague Opportunity") {
		t.Fatal("unparseable job shown as comparable role")
	}
}

func TestParseJobRequirementsBoundaries(t *testing.T) {
	reqs := parseJobRequirements("Lead the team.\n- Coordinate inbound freight schedules\n1. Track shipment exceptions\n* Coordinate inbound freight schedules\nA short line\n- ok\n- Confirm dock receipts")
	if len(reqs) != 3 {
		t.Fatalf("want 3 deduped bullets, got %q", reqs)
	}
	if len(parseJobRequirements("Just prose. No bullets here.")) != 0 {
		t.Fatal("prose must not produce requirements")
	}
	many := ""
	for i := 0; i < 12; i++ {
		many += "- Requirement number " + strconv.Itoa(i+1) + "\n"
	}
	if len(parseJobRequirements(many)) != 8 {
		t.Fatal("requirements must cap at the engine maximum")
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
