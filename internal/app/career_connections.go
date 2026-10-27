package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	discovery "github.com/jenksed/yourmove-career-discovery"
	coverprovider "github.com/jenksed/yourmove-go/internal/coverletter/provider"
)

// HACK-006: cross-experience synthesis persistence, challenge state, and the
// Living Career Intelligence Studio. Interpretations are hypotheses; they
// are never written into career_evidence and never auto-rewrite artifacts.

func initializeConnections(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS career_connections (
	 id INTEGER PRIMARY KEY CHECK(id=1), evidence_hash TEXT NOT NULL DEFAULT '', synthesis TEXT NOT NULL DEFAULT '',
	 notice TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '')`); err != nil {
		return err
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS career_connection_challenges (
	 synthesis_hash TEXT NOT NULL, connection_id TEXT NOT NULL, state TEXT NOT NULL DEFAULT '',
	 note TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '', PRIMARY KEY(synthesis_hash, connection_id))`)
	return err
}

// discoveryDestination names the actual inference destination before the
// candidate authorizes a request. HACK-BASETEN: Baseten is disclosed when
// selected; unset preserves the Meta destination.
func discoveryDestination() string {
	backend, err := coverprovider.SelectedMuseBackend()
	if err == nil && backend == coverprovider.MuseBackendBaseten {
		return "Baseten (temporary hackathon backend, meta-models/muse-spark-open-1026)"
	}
	return "Meta (muse-spark-1.3, standard mode)"
}

type connectionView struct {
	discovery.Connection
	Challenge string
	Note      string
	Stale     bool // a supporting record changed revision since synthesis
}

type studioPage struct {
	Evidences   []CareerEvidence
	Drafts      []CareerEvidence
	Connections []connectionView
	Synthesis   *discovery.ConnectionSynthesis
	Hash        string
	StoredHash  string
	Stale       bool
	Notice      string
	Destination string
	Roles       []roleAnalysisPage
	ChallengeN  int
}

func reviewedSynthesisEvidences(a *App) ([]CareerEvidence, []discovery.ConnectionEvidence, error) {
	reviewed, err := reviewedRoleEvidencesAll(a)
	if err != nil {
		return nil, nil, err
	}
	inputs := make([]discovery.ConnectionEvidence, 0, len(reviewed))
	for _, e := range reviewed {
		in := discovery.ConnectionEvidence{ID: e.ID, Revision: e.Revision, Title: e.Title, Situation: e.Situation, Actions: e.Actions, OutcomeState: e.OutcomeState}
		if sources, err := a.careerEvidenceTraceSources(e); err == nil {
			for _, s := range sources {
				in.SourceTexts = append(in.SourceTexts, s.ResponseText)
			}
		}
		inputs = append(inputs, in)
	}
	return reviewed, inputs, nil
}

// reviewedRoleEvidencesAll is reviewedRoleEvidences without the role cap:
// synthesis accepts up to MaxConnectionEvidences records.
func reviewedRoleEvidencesAll(a *App) ([]CareerEvidence, error) {
	entries, err := a.careerEvidenceEntries()
	if err != nil {
		return nil, err
	}
	var out []CareerEvidence
	for _, e := range entries {
		if e.Status != "reviewed" || strings.TrimSpace(e.Title) == "" || strings.TrimSpace(e.Situation) == "" || strings.TrimSpace(e.Actions) == "" {
			continue
		}
		out = append(out, e)
		if len(out) >= discovery.MaxConnectionEvidences {
			break
		}
	}
	return out, nil
}

func (p studioPage) EvidenceTitle(id int64) string {
	for _, e := range p.Evidences {
		if e.ID == id {
			return e.Title
		}
	}
	return "Record (no longer reviewed or present)"
}

func (a *App) loadStudio() (studioPage, error) {
	page := studioPage{Destination: discoveryDestination()}
	entries, err := a.careerEvidenceEntries()
	if err != nil {
		return page, err
	}
	for _, e := range entries {
		if e.Status == "reviewed" {
			page.Evidences = append(page.Evidences, e)
		} else {
			page.Drafts = append(page.Drafts, e)
		}
	}
	reviewed, _, err := reviewedSynthesisEvidences(a)
	if err != nil {
		return page, err
	}
	page.Hash = roleEvidenceHash(reviewed)
	var raw, notice string
	err = a.db.QueryRow(`SELECT evidence_hash,synthesis,notice FROM career_connections WHERE id=1`).Scan(&page.StoredHash, &raw, &notice)
	if err == nil && raw != "" {
		var syn discovery.ConnectionSynthesis
		if json.Unmarshal([]byte(raw), &syn) == nil {
			page.Synthesis = &syn
			page.Notice = notice
			page.Stale = page.StoredHash != page.Hash
			current := map[int64]int64{}
			for _, e := range reviewed {
				current[e.ID] = e.Revision
			}
			challenges := map[string][2]string{}
			rows, qerr := a.db.Query(`SELECT connection_id,state,note FROM career_connection_challenges WHERE synthesis_hash=?`, page.StoredHash)
			if qerr == nil {
				defer rows.Close()
				for rows.Next() {
					var id, state, note string
					if rows.Scan(&id, &state, &note) == nil {
						challenges[id] = [2]string{state, note}
					}
				}
			}
			for _, c := range syn.Connections {
				v := connectionView{Connection: c}
				if ch, ok := challenges[c.ID]; ok {
					v.Challenge, v.Note = ch[0], ch[1]
					if v.Challenge != "" {
						page.ChallengeN++
					}
				}
				for _, s := range c.Supports {
					if rev, ok := current[s.EvidenceID]; !ok || rev != s.EvidenceRevision {
						v.Stale = true
					}
				}
				page.Connections = append(page.Connections, v)
			}
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return page, err
	}
	// Role summaries reuse the saved HACK-005 analyses; no new inference here.
	hash := roleEvidenceHash(mustCap(reviewed))
	for _, role := range careerRoleProfiles() {
		entry := roleAnalysisPage{Role: role, Hash: hash}
		var araw, anotice string
		if err := a.db.QueryRow(`SELECT analysis,notice FROM career_role_analyses WHERE role_key=? AND evidence_hash=?`, role.Key, hash).Scan(&araw, &anotice); err == nil && araw != "" {
			var analysis discovery.RoleAnalysis
			if json.Unmarshal([]byte(araw), &analysis) == nil && analysis.RoleKey == role.Key {
				entry.Analysis = &analysis
				entry.Notice = anotice
			}
		}
		if job, err := a.roleOpportunity(role.Key); err == nil && job != nil {
			entry.Job = job
			entry.Flags = a.savedAnswerProvenance(*job)
		}
		page.Roles = append(page.Roles, entry)
	}
	return page, nil
}

func mustCap(e []CareerEvidence) []CareerEvidence {
	if len(e) > discovery.MaxRoleEvidences {
		return e[:discovery.MaxRoleEvidences]
	}
	return e
}

func (a *App) careerStudioPage(w http.ResponseWriter, r *http.Request) {
	page, err := a.loadStudio()
	if err != nil {
		http.Error(w, "Unable to load Career Studio", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	_ = a.resources.careerStudioTemplate.Execute(w, page)
}

func (a *App) runSynthesis(ctx context.Context, evidences []CareerEvidence, inputs []discovery.ConnectionEvidence) error {
	generate, metrics, err := a.discoveryRunner()
	var syn discovery.ConnectionSynthesis
	if err == nil {
		syn, err = discovery.SynthesizeConnections(ctx, generate, inputs)
	}
	hash := roleEvidenceHash(evidences)
	notice := fmt.Sprintf("Muse compared %d reviewed experience(s) together. These connections are hypotheses for your judgment; nothing was approved or adopted.", len(evidences))
	if err != nil {
		notice = "Muse could not complete this synthesis (" + coverprovider.ErrorNotice(err) + "). No interpretation was saved."
		if errors.Is(err, discovery.ErrInvalid) {
			notice = "Muse output did not pass source or format checks. Nothing was saved; retry the synthesis."
		}
	}
	notice += discoveryUsageNote(*metrics, true)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err != nil {
		_, dbErr := a.db.Exec(`INSERT INTO career_connections(id,evidence_hash,synthesis,notice,created_at,updated_at) VALUES(1,?,'',?,?,?) ON CONFLICT(id) DO UPDATE SET notice=excluded.notice,updated_at=excluded.updated_at`, hash, notice, now, now)
		return dbErr
	}
	raw, _ := json.Marshal(syn)
	_, dbErr := a.db.Exec(`INSERT INTO career_connections(id,evidence_hash,synthesis,notice,created_at,updated_at) VALUES(1,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET evidence_hash=excluded.evidence_hash,synthesis=excluded.synthesis,notice=excluded.notice,updated_at=excluded.updated_at`, hash, string(raw), notice, now, now)
	return dbErr
}

func (a *App) synthesizeConnections(w http.ResponseWriter, r *http.Request) {
	if !discoveryConsent(w, r) {
		return
	}
	if !a.discoveryMu.TryLock() {
		http.Error(w, "A Muse request is already in progress. Wait before retrying.", http.StatusConflict)
		return
	}
	defer a.discoveryMu.Unlock()
	evidences, inputs, err := reviewedSynthesisEvidences(a)
	if err != nil || len(evidences) < 2 {
		http.Error(w, "Review at least two Career Evidence records before looking for connections.", 422)
		return
	}
	if err := a.runSynthesis(r.Context(), evidences, inputs); err != nil {
		http.Error(w, "Unable to save synthesis", 500)
		return
	}
	http.Redirect(w, r, "/career-studio#connections", http.StatusSeeOther)
}

// challengeConnection records the candidate's judgment. It never edits
// evidence, never deletes the interpretation, and survives restart.
func (a *App) challengeConnection(w http.ResponseWriter, r *http.Request) {
	if !parseLocalForm(w, r, 8<<10) {
		return
	}
	connID := r.PathValue("id")
	if connID != "conn-1" && connID != "conn-2" && connID != "conn-3" {
		http.NotFound(w, r)
		return
	}
	state := r.PostForm.Get("state")
	if state != "" && state != "questioned" && state != "unsupported" {
		http.Error(w, "Choose a challenge state", 400)
		return
	}
	note := strings.TrimSpace(r.PostForm.Get("note"))
	if len(note) > 500 {
		http.Error(w, "Keep the note under 500 bytes", 400)
		return
	}
	var hash string
	if err := a.db.QueryRow(`SELECT evidence_hash FROM career_connections WHERE id=1`).Scan(&hash); err != nil || hash == "" {
		http.Error(w, "No saved synthesis to challenge", 409)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := a.db.Exec(`INSERT INTO career_connection_challenges(synthesis_hash,connection_id,state,note,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(synthesis_hash,connection_id) DO UPDATE SET state=excluded.state,note=excluded.note,updated_at=excluded.updated_at`, hash, connID, state, note, now); err != nil {
		http.Error(w, "Unable to save challenge", 500)
		return
	}
	http.Redirect(w, r, "/career-studio#connections", http.StatusSeeOther)
}
