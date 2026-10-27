package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	discovery "github.com/jenksed/yourmove-career-discovery"
	coverprovider "github.com/jenksed/yourmove-go/internal/coverletter/provider"
)

// Only interview state is new storage; approved evidence stays in career_evidence.
func initializeDiscovery(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS career_discovery (
	 id INTEGER PRIMARY KEY, revision INTEGER NOT NULL DEFAULT 1,
	 state TEXT NOT NULL, evidence_id INTEGER NOT NULL DEFAULT 0,
	 notice TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL)`)
	return err
}

type discoveryPage struct {
	ID, Revision, EvidenceID int64
	State                    discovery.State
	Notice                   string
	CanAnswer                bool
	Sessions                 []discoverySessionLink
}
type discoverySessionLink struct {
	ID        int64
	CreatedAt string
}

func (p discoveryPage) NeedsLeads() bool {
	return p.ID != 0 && p.EvidenceID == 0 && len(p.State.Leads) == 0 && len(p.State.Responses) == 0 && p.State.Question == ""
}
func (p discoveryPage) LeadPicker() bool {
	return len(p.State.Leads) > 0 && p.State.SelectedLeadID == "" && len(p.State.Responses) == 0
}

func (a *App) discoveryByID(id int64) (discoveryPage, error) {
	var p discoveryPage
	var raw string
	p.ID = id
	err := a.db.QueryRow(`SELECT revision,state,evidence_id,notice FROM career_discovery WHERE id=?`, id).Scan(&p.Revision, &raw, &p.EvidenceID, &p.Notice)
	if err != nil {
		return p, err
	}
	if json.Unmarshal([]byte(raw), &p.State) != nil {
		return p, discovery.ErrInvalid
	}
	p.CanAnswer = p.EvidenceID == 0 && p.State.Question != "" && len(p.State.Responses) < discovery.MaxResponses
	return p, nil
}

// careerEvidenceTraceSources rebinds interview responses to the exact reviewed
// record revision. A record without a discovery row remains usable for ordinary
// answer drafting, but cannot advertise an interview Evidence Trace.
func (a *App) careerEvidenceTraceSources(e CareerEvidence) ([]discovery.TraceSource, error) {
	var raw string
	err := a.db.QueryRow(`SELECT state FROM career_discovery WHERE evidence_id=?`, e.ID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state discovery.State
	if json.Unmarshal([]byte(raw), &state) != nil || state.Proposal == nil || state.ValidateProposal() != nil || e.SourceWriting != state.Transcript() || len(state.Responses) == 0 {
		return nil, discovery.ErrInvalid
	}
	sources := make([]discovery.TraceSource, 0, len(state.Responses))
	for i, response := range state.Responses {
		if response.Source.ID != fmt.Sprintf("response-%d", i+1) || response.Source.Kind != "self_report" || !validNarrativeText(response.Question) || !validNarrativeText(response.Source.Text) || len(response.Source.Text) > 2000 {
			return nil, discovery.ErrInvalid
		}
		sources = append(sources, discovery.TraceSource{EvidenceID: e.ID, EvidenceRevision: e.Revision, ResponseID: response.Source.ID, ResponseText: response.Source.Text})
	}
	return sources, nil
}

func (a *App) careerDiscoveryPage(w http.ResponseWriter, r *http.Request) {
	var p discoveryPage
	if raw := r.PathValue("id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id < 1 {
			http.NotFound(w, r)
			return
		}
		p, err = a.discoveryByID(id)
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "Unable to load interview", 500)
			return
		}
	} else {
		rows, err := a.db.Query(`SELECT id,created_at FROM career_discovery ORDER BY id DESC LIMIT 20`)
		if err != nil {
			http.Error(w, "Unable to load interviews", 500)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var link discoverySessionLink
			if err := rows.Scan(&link.ID, &link.CreatedAt); err != nil {
				http.Error(w, "Unable to load interviews", 500)
				return
			}
			p.Sessions = append(p.Sessions, link)
		}
		if rows.Err() != nil {
			http.Error(w, "Unable to load interviews", 500)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	_ = a.resources.careerDiscoveryTemplate.Execute(w, p)
}

func discoveryConsent(w http.ResponseWriter, r *http.Request) bool {
	if !parseLocalForm(w, r, 64<<10) {
		return false
	}
	if len(r.PostForm["consent"]) != 1 || r.PostForm.Get("consent") != "yes" {
		http.Error(w, "Authorize sending this interview to Meta before continuing", 400)
		return false
	}
	return true
}

func (a *App) startCareerDiscovery(w http.ResponseWriter, r *http.Request) {
	if !discoveryConsent(w, r) {
		return
	}
	if !a.discoveryMu.TryLock() {
		http.Error(w, "A Muse interview request is already in progress. Wait before retrying.", http.StatusConflict)
		return
	}
	defer a.discoveryMu.Unlock()
	if len(r.PostForm["resume"]) != 1 {
		http.Error(w, "Provide one resume excerpt", 400)
		return
	}
	s, err := discovery.New(r.PostForm.Get("resume"))
	if err != nil {
		http.Error(w, "Use a resume excerpt of 1–8000 bytes", 422)
		return
	}
	raw, _ := json.Marshal(s)
	result, err := a.db.Exec(`INSERT INTO career_discovery(state,created_at) VALUES(?,?)`, string(raw), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		http.Error(w, "Unable to save interview", 500)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Unable to locate saved interview", 500)
		return
	}
	p := discoveryPage{ID: id, Revision: 1, State: s}
	if err := a.runLeads(r.Context(), p); err != nil {
		http.Error(w, fmt.Sprintf("Interview input saved, but the model result could not be saved. Return to /career-discovery/%d to recover it.", id), 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/career-discovery/%d", id), http.StatusSeeOther)
}

func (a *App) discoveryRunner() (discovery.Generate, *coverprovider.RequestMetrics, error) {
	if a.discoveryGenerate != nil {
		return a.discoveryGenerate, &coverprovider.RequestMetrics{}, nil
	}
	runner, err := coverprovider.ConfigureMeta()
	if err != nil {
		return nil, &coverprovider.RequestMetrics{}, err
	}
	metrics := &coverprovider.RequestMetrics{}
	gen := func(ctx context.Context, prompt, instructions string) (string, error) {
		return runner.GenerateDiscovery(ctx, prompt, instructions, func(m coverprovider.RequestMetrics) { *metrics = m })
	}
	return gen, metrics, nil
}

// discoveryUsageNote reports observed inference usage with the actual backend
// and model identity, so diagnostics and evaluation results preserve which
// model ran. Meta dollar estimates never apply to Baseten usage, which bills
// the hackathon account separately.
func discoveryUsageNote(m coverprovider.RequestMetrics, withFinish bool) string {
	if m.PromptTokens <= 0 && m.CompletionTokens <= 0 {
		return ""
	}
	// HACK-BASETEN: temporary backend branch; delete on removal.
	if m.Backend == coverprovider.MuseBackendBaseten {
		note := fmt.Sprintf(" Tokens: %d input / %d output via Baseten %s", m.PromptTokens, m.CompletionTokens, m.Model)
		if withFinish {
			note += fmt.Sprintf("; finish: %s", m.FinishReason)
		}
		return note + "."
	}
	// Standard-mode estimate, not an account charge; rates checked 2026-10-26.
	cached := min(m.CachedTokens, m.PromptTokens)
	cost := (float64(m.PromptTokens-cached)*1.25 + float64(cached)*0.15 + float64(m.CompletionTokens)*4.25) / 1e6
	note := fmt.Sprintf(" Tokens: %d input / %d output; estimated inference $%.5f", m.PromptTokens, m.CompletionTokens, cost)
	if withFinish {
		note += fmt.Sprintf("; finish: %s", m.FinishReason)
	}
	return note + "."
}

func (a *App) runLeads(ctx context.Context, p discoveryPage) error {
	generate, metrics, err := a.discoveryRunner()
	next := p.State
	if err == nil {
		var leads []discovery.Lead
		leads, err = discovery.FindLeads(ctx, generate, p.State.Resume.Text)
		if err == nil {
			next.Leads = leads
		}
	}
	notice := "Muse identified investigation leads. Choose one to explore, or start a new excerpt."
	if err != nil {
		notice = "Muse could not identify leads (" + coverprovider.ErrorNotice(err) + "). Your excerpt is saved; retry below. Nothing was approved."
		if errors.Is(err, discovery.ErrInvalid) {
			notice = "Muse output did not pass lead checks. Your excerpt is saved; retry below. Nothing was approved."
		}
	}
	notice += discoveryUsageNote(*metrics, false)
	raw, _ := json.Marshal(next)
	result, err := a.db.Exec(`UPDATE career_discovery SET state=?,notice=?,revision=revision+1 WHERE id=? AND revision=? AND evidence_id=0`, string(raw), notice, p.ID, p.Revision)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrCareerEvidenceStale
	}
	return nil
}

func (a *App) retryCareerLeads(w http.ResponseWriter, r *http.Request) {
	if !discoveryConsent(w, r) {
		return
	}
	if !a.discoveryMu.TryLock() {
		http.Error(w, "A Muse interview request is already in progress. Wait before retrying.", http.StatusConflict)
		return
	}
	defer a.discoveryMu.Unlock()
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	p, err := a.discoveryByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	revision, err := parseCareerEvidenceRevision(r)
	if err != nil || revision != p.Revision || p.EvidenceID != 0 || len(p.State.Leads) > 0 {
		http.Error(w, "Interview changed. Reload before continuing.", 409)
		return
	}
	if err := a.runLeads(r.Context(), p); err != nil {
		http.Error(w, "Unable to save leads.", 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/career-discovery/%d", id), http.StatusSeeOther)
}

func (a *App) selectCareerLead(w http.ResponseWriter, r *http.Request) {
	if !discoveryConsent(w, r) {
		return
	}
	if !a.discoveryMu.TryLock() {
		http.Error(w, "A Muse interview request is already in progress. Wait before retrying.", http.StatusConflict)
		return
	}
	defer a.discoveryMu.Unlock()
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	p, err := a.discoveryByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	revision, err := parseCareerEvidenceRevision(r)
	if err != nil || revision != p.Revision || p.EvidenceID != 0 {
		http.Error(w, "Interview changed. Reload before continuing.", 409)
		return
	}
	p.State, err = discovery.SelectLead(p.State, r.PostForm.Get("lead_id"))
	if err != nil {
		http.Error(w, "Choose a listed lead before responses begin.", 422)
		return
	}
	raw, _ := json.Marshal(p.State)
	result, err := a.db.Exec(`UPDATE career_discovery SET state=?,notice='Request in progress. If interrupted, retry from this page.',revision=revision+1 WHERE id=? AND revision=? AND evidence_id=0`, string(raw), id, p.Revision)
	if err != nil {
		http.Error(w, "Unable to save selection", 500)
		return
	}
	if n, _ := result.RowsAffected(); n != 1 {
		http.Error(w, "Interview changed. Reload before continuing.", 409)
		return
	}
	p.Revision++
	if err := a.runDiscovery(r.Context(), p); err != nil {
		http.Error(w, "Your selection is saved, but the model result could not be saved. Reload the interview.", 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/career-discovery/%d", id), http.StatusSeeOther)
}

// exploreCareerLead starts a fresh focused session for another lead, keeping
// the original session and any linked evidence untouched.
func (a *App) exploreCareerLead(w http.ResponseWriter, r *http.Request) {
	if !discoveryConsent(w, r) {
		return
	}
	if !a.discoveryMu.TryLock() {
		http.Error(w, "A Muse interview request is already in progress. Wait before retrying.", http.StatusConflict)
		return
	}
	defer a.discoveryMu.Unlock()
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	p, err := a.discoveryByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if _, err := parseCareerEvidenceRevision(r); err != nil {
		http.Error(w, "Reload before continuing.", 409)
		return
	}
	fresh, err := discovery.New(p.State.Resume.Text)
	if err != nil {
		http.Error(w, "Saved excerpt is no longer valid.", 422)
		return
	}
	fresh.Leads = p.State.Leads
	fresh, err = discovery.SelectLead(fresh, r.PostForm.Get("lead_id"))
	if err != nil {
		http.Error(w, "Choose a listed lead.", 422)
		return
	}
	raw, _ := json.Marshal(fresh)
	result, err := a.db.Exec(`INSERT INTO career_discovery(state,created_at) VALUES(?,?)`, string(raw), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		http.Error(w, "Unable to save interview", 500)
		return
	}
	newID, _ := result.LastInsertId()
	np := discoveryPage{ID: newID, Revision: 1, State: fresh}
	if err := a.runDiscovery(r.Context(), np); err != nil {
		http.Error(w, fmt.Sprintf("Interview saved, but the model result could not be saved. Return to /career-discovery/%d to recover it.", newID), 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/career-discovery/%d", newID), http.StatusSeeOther)
}

func (a *App) nextCareerDiscovery(w http.ResponseWriter, r *http.Request) {
	if !discoveryConsent(w, r) {
		return
	}
	if !a.discoveryMu.TryLock() {
		http.Error(w, "A Muse interview request is already in progress. Wait before retrying.", http.StatusConflict)
		return
	}
	defer a.discoveryMu.Unlock()
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	p, err := a.discoveryByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	revision, err := parseCareerEvidenceRevision(r)
	if err != nil || revision != p.Revision || p.EvidenceID != 0 {
		http.Error(w, "Interview changed. Reload before continuing.", 409)
		return
	}
	if p.State.Question != "" {
		if len(r.PostForm["answer"]) != 1 {
			http.Error(w, "Provide one response", 400)
			return
		}
		p.State, err = discovery.RecordAnswer(p.State, r.PostForm.Get("answer"))
		if err != nil {
			http.Error(w, "Use a response of 1–2000 bytes; interviews allow up to six responses", 422)
			return
		}
	} else if len(r.PostForm["answer"]) != 0 {
		http.Error(w, "This is a retry of saved input. Reload before adding another response; no request was sent.", http.StatusBadRequest)
		return
	}
	// Claim a pending revision before inference. A duplicate POST cannot send a
	// second charged request from the same page. Record candidate text first.
	raw, _ := json.Marshal(p.State)
	result, err := a.db.Exec(`UPDATE career_discovery SET state=?,notice='Request in progress. If interrupted, retry from this page.',revision=revision+1 WHERE id=? AND revision=? AND evidence_id=0`, string(raw), id, p.Revision)
	if err != nil {
		http.Error(w, "Unable to save response", 500)
		return
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		http.Error(w, "Interview changed. Reload before continuing.", 409)
		return
	}
	p.Revision++
	if err := a.runDiscovery(r.Context(), p); err != nil {
		http.Error(w, "Your response is saved, but the model result could not be saved. Reload the interview.", 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/career-discovery/%d", id), http.StatusSeeOther)
}

func (a *App) runDiscovery(ctx context.Context, p discoveryPage) error {
	generate := a.discoveryGenerate
	var metrics coverprovider.RequestMetrics
	var err error
	if generate == nil {
		var runner coverprovider.Runner
		runner, err = coverprovider.ConfigureMeta()
		if err == nil {
			generate = func(ctx context.Context, prompt, instructions string) (string, error) {
				return runner.GenerateDiscovery(ctx, prompt, instructions, func(m coverprovider.RequestMetrics) { metrics = m })
			}
		}
	}
	started := time.Now()
	next := p.State
	if err == nil {
		next, err = discovery.Next(ctx, generate, p.State)
	}
	notice := fmt.Sprintf("Muse response received in %.1f seconds. Proposals need your review.", time.Since(started).Seconds())
	if err != nil {
		notice = "Muse request did not produce usable interview output (" + coverprovider.ErrorNotice(err) + "). Your recorded input is saved; no evidence was approved."
		if errors.Is(err, discovery.ErrInvalid) {
			notice = "Muse output did not pass source or format checks. Your input is saved; retry or start a new interview. No evidence was approved."
		}
	}
	notice += discoveryUsageNote(metrics, true)
	raw, _ := json.Marshal(next)
	// A late result cannot overwrite a newer candidate response or review draft.
	result, err := a.db.Exec(`UPDATE career_discovery SET state=?,notice=?,revision=revision+1 WHERE id=? AND revision=? AND evidence_id=0`, string(raw), notice, p.ID, p.Revision)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrCareerEvidenceStale
	}
	return nil
}

func (a *App) discoveryEvidenceDraft(w http.ResponseWriter, r *http.Request) {
	if !parseLocalForm(w, r, 4<<10) {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	p, err := a.discoveryByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if p.EvidenceID != 0 {
		http.Redirect(w, r, fmt.Sprintf("/career-evidence/%d", p.EvidenceID), 303)
		return
	}
	revision, err := parseCareerEvidenceRevision(r)
	if err != nil || revision != p.Revision || p.State.Proposal == nil {
		http.Error(w, "Reload the current proposal before creating a draft", 409)
		return
	}
	if p.State.ValidateProposal() != nil {
		http.Error(w, "The proposal source no longer resolves to a recorded response. No draft was saved.", http.StatusUnprocessableEntity)
		return
	}
	proposal := p.State.Proposal
	notes := fmt.Sprintf("AI proposal from Career Discovery interview %d. Self-report; not independently verified.\nSupporting source [%s]:\n%s", id, proposal.SourceID, proposal.Quote)
	if len(proposal.Unresolved) > 0 {
		notes += "\nUnresolved (not established facts):\n" + strings.Join(proposal.Unresolved, "\n")
	}
	content := careerEvidenceContent{Title: proposal.Title, Situation: proposal.Experience, Actions: proposal.Claim, OutcomeState: "unknown", SourceNotes: notes}
	if !content.valid() {
		http.Error(w, "Proposal is too long for Career Evidence; continue discovery", 422)
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		http.Error(w, "Unable to save draft", 500)
		return
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := tx.Exec(`INSERT INTO career_evidence(title,situation,actions,outcome_state,source_notes,source_writing,status,revision,created_at,updated_at) VALUES(?,?,?,'unknown',?,?,'draft',1,?,?)`, content.Title, content.Situation, content.Actions, content.SourceNotes, p.State.Transcript(), now, now)
	if err != nil {
		http.Error(w, "Unable to save draft", 500)
		return
	}
	evidenceID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Unable to locate draft", 500)
		return
	}
	result, err = tx.Exec(`UPDATE career_discovery SET evidence_id=?,revision=revision+1 WHERE id=? AND revision=? AND evidence_id=0`, evidenceID, id, p.Revision)
	if err != nil {
		http.Error(w, "Unable to link draft", 500)
		return
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		http.Error(w, "Interview changed. Reload before saving.", 409)
		return
	}
	if tx.Commit() != nil {
		http.Error(w, "Unable to commit draft", 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/career-evidence/%d", evidenceID), http.StatusSeeOther)
}
