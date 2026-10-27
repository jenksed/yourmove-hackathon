package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	discovery "github.com/jenksed/yourmove-career-discovery"
	coverprovider "github.com/jenksed/yourmove-go/internal/coverletter/provider"
)

// Synthetic demo opportunities. The candidate creates them locally; nothing
// is submitted to an employer.
func careerRoleProfiles() []discovery.RoleProfile {
	return []discovery.RoleProfile{
		{Key: "support-engineer", Title: "Technical Support Engineer", Focus: "Linux troubleshooting, MySQL diagnostics, production incidents and escalation handoffs.",
			Requirements: []string{
				"Diagnose production Linux and MySQL issues",
				"Document incidents and hand off clearly to engineering",
				"Communicate technical status to internal customers",
				"Own customer accounts and renewal targets",
				"Manage multi-account customer relationships",
			}},
		{Key: "account-manager", Title: "Technical Account Manager", Focus: "Customer-facing technical communication, ownership boundaries, collaboration, coordination and escalation.",
			Requirements: []string{
				"Explain technical issues clearly to customers",
				"Coordinate across support and engineering teams",
				"Track follow-through on customer escalations",
				"Distinguish personal contribution from team work",
				"Own customer accounts and renewal targets",
				"Apply production database changes directly",
			}},
	}
}

func careerRoleByKey(key string) (discovery.RoleProfile, bool) {
	for _, role := range careerRoleProfiles() {
		if role.Key == key {
			return role, true
		}
	}
	return discovery.RoleProfile{}, false
}

// roleProfileFromJob derives a comparison profile from a saved opportunity
// instead of a hardcoded demo role. Requirements come only from bullet-like
// description lines; prose without parseable requirement bullets is refused
// rather than split into invented requirements. The comparison itself stays
// a hypothesis: AnalyzeRole validates that unmet items echo these strings.
func roleProfileFromJob(job Job) (discovery.RoleProfile, error) {
	title := strings.TrimSpace(job.Title)
	if title == "" || len(title) > 160 {
		return discovery.RoleProfile{}, errors.New("saved opportunity needs a title before comparison")
	}
	requirements := parseJobRequirements(job.Description)
	if len(requirements) < 2 {
		return discovery.RoleProfile{}, errors.New("saved opportunity needs at least two requirement bullets before comparison")
	}
	focus := strings.TrimSpace(job.Company)
	headline := firstDescriptionLine(job.Description)
	if headline != "" {
		if focus != "" {
			focus += " — "
		}
		focus += headline
	}
	if len(focus) > 600 {
		focus = focus[:600]
	}
	if strings.TrimSpace(focus) == "" {
		focus = title
	}
	return discovery.RoleProfile{
		Key:          fmt.Sprintf("saved-job-%d", job.ID),
		Title:        title,
		Focus:        focus,
		Requirements: requirements,
	}, nil
}

// parseJobRequirements keeps bullet-like description lines (dash, star,
// bullet, or numbered markers) between 4 and 200 bytes, deduplicated and
// capped at the engine's per-role maximum. Plain prose lines are ignored:
// without a bullet marker there is no auditable requirement boundary.
func parseJobRequirements(description string) []string {
	var out []string
	seen := map[string]bool{}
	for _, raw := range strings.Split(description, "\n") {
		line := strings.TrimSpace(raw)
		marked := false
		for _, marker := range []string{"-", "*", "•", "·"} {
			if strings.HasPrefix(line, marker+" ") || strings.HasPrefix(line, marker+"\t") {
				line = strings.TrimSpace(line[len(marker):])
				marked = true
				break
			}
		}
		if !marked && len(line) >= 3 && line[0] >= '0' && line[0] <= '9' {
			if dot := strings.IndexAny(line, ".)"); dot > 0 && dot <= 3 {
				if rest := strings.TrimSpace(line[dot+1:]); rest != "" {
					line = rest
					marked = true
				}
			}
		}
		if !marked || len(line) < 4 || len(line) > 200 || seen[strings.ToLower(line)] {
			continue
		}
		seen[strings.ToLower(line)] = true
		out = append(out, line)
		if len(out) >= discovery.MaxRoleRequirements {
			break
		}
	}
	return out
}

func firstDescriptionLine(description string) string {
	for _, line := range strings.Split(description, "\n") {
		line = strings.TrimSpace(strings.Trim(line, "-*•· \t"))
		if line != "" && len(line) <= 200 {
			return line
		}
	}
	return ""
}

func initializeRoleAnalyses(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS career_role_analyses (
	 role_key TEXT PRIMARY KEY, evidence_hash TEXT NOT NULL DEFAULT '', analysis TEXT NOT NULL DEFAULT '',
	 notice TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`)
	return err
}

type roleAnalysisPage struct {
	Role     discovery.RoleProfile
	Analysis *discovery.RoleAnalysis
	Hash     string
	Notice   string
	Job      *Job
	Flags    []answerProvenanceFlag
}

type careerRolesPage struct {
	Roles       []roleAnalysisPage
	Evidences   []CareerEvidence
	Notice      string
	Destination string
}

func (p careerRolesPage) EvidenceTitle(id int64) string {
	for _, e := range p.Evidences {
		if e.ID == id {
			return e.Title
		}
	}
	return "Record"
}

func roleEvidenceHash(evidences []CareerEvidence) string {
	parts := make([]string, 0, len(evidences))
	for _, e := range evidences {
		parts = append(parts, fmt.Sprintf("%d:%d:%s", e.ID, e.Revision, e.CreatedAt))
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func reviewedRoleEvidences(a *App) ([]CareerEvidence, error) {
	entries, err := a.careerEvidenceEntries()
	if err != nil {
		return nil, err
	}
	var out []CareerEvidence
	for _, e := range entries {
		if e.Status != "reviewed" {
			continue
		}
		if strings.TrimSpace(e.Title) == "" || strings.TrimSpace(e.Situation) == "" || strings.TrimSpace(e.Actions) == "" {
			continue
		}
		out = append(out, e)
		if len(out) >= discovery.MaxRoleEvidences {
			break
		}
	}
	return out, nil
}

func roleProjectEvidence(e CareerEvidence) discovery.RoleEvidence {
	return discovery.RoleEvidence{ID: e.ID, Revision: e.Revision, Title: e.Title, Situation: e.Situation, Actions: e.Actions, OutcomeState: e.OutcomeState}
}

// roleOpportunity returns the demo job linked to a role, if the candidate
// created one.
func (a *App) roleOpportunity(roleKey string) (*Job, error) {
	jobs, err := a.allJobs()
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		if j.Host == "example.test" && strings.Contains(j.URL, "/demo/"+roleKey) {
			job := j
			return &job, nil
		}
	}
	return nil, nil
}

func (a *App) careerRolesPage(w http.ResponseWriter, r *http.Request) {
	evidences, err := reviewedRoleEvidences(a)
	if err != nil {
		http.Error(w, "Unable to load reviewed evidence", 500)
		return
	}
	hash := roleEvidenceHash(evidences)
	page := careerRolesPage{Evidences: evidences, Destination: discoveryDestination()}
	appendRoleEntry := func(role discovery.RoleProfile, job *Job) {
		entry := roleAnalysisPage{Role: role, Hash: hash}
		var raw, notice string
		err := a.db.QueryRow(`SELECT analysis,notice FROM career_role_analyses WHERE role_key=? AND evidence_hash=?`, role.Key, hash).Scan(&raw, &notice)
		if err == nil && raw != "" {
			var analysis discovery.RoleAnalysis
			if json.Unmarshal([]byte(raw), &analysis) == nil && analysis.RoleKey == role.Key {
				entry.Analysis = &analysis
				entry.Notice = notice
			}
		}
		entry.Job = job
		if job != nil {
			entry.Flags = a.savedAnswerProvenance(*job)
		}
		page.Roles = append(page.Roles, entry)
	}
	for _, role := range careerRoleProfiles() {
		job, err := a.roleOpportunity(role.Key)
		if err == nil {
			appendRoleEntry(role, job)
		} else {
			appendRoleEntry(role, nil)
		}
	}
	// Saved opportunities with parseable requirement bullets compare through
	// the same engine path as the demo roles. Unparseable descriptions are
	// omitted here; their opportunity page stays the path for answers.
	if jobs, err := a.allJobs(); err == nil {
		for _, job := range jobs {
			profile, err := roleProfileFromJob(job)
			if err != nil {
				continue
			}
			jobCopy := job
			appendRoleEntry(profile, &jobCopy)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	_ = a.resources.careerRolesTemplate.Execute(w, page)
}

func (a *App) runRoleAnalysis(ctx context.Context, role discovery.RoleProfile, evidences []CareerEvidence) error {
	generate, metrics, err := a.discoveryRunner()
	var analysis discovery.RoleAnalysis
	inputs := make([]discovery.RoleEvidence, 0, len(evidences))
	for _, e := range evidences {
		inputs = append(inputs, roleProjectEvidence(e))
	}
	if err == nil {
		analysis, err = discovery.AnalyzeRole(ctx, generate, role, inputs)
	}
	hash := roleEvidenceHash(evidences)
	notice := fmt.Sprintf("Muse compared %d reviewed experience(s) against this role. Explanations need your judgment; nothing was adopted.", len(evidences))
	if err != nil {
		notice = "Muse could not complete this comparison (" + coverprovider.ErrorNotice(err) + "). No analysis was saved."
		if errors.Is(err, discovery.ErrInvalid) {
			notice = "Muse output did not pass evidence checks. Nothing was saved; retry the comparison."
		}
	}
	raw := ""
	if err == nil {
		encoded, _ := json.Marshal(analysis)
		raw = string(encoded)
	}
	notice += discoveryUsageNote(*metrics, true)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if raw == "" {
		// Surface the failure without overwriting a prior good analysis.
		_, dbErr := a.db.Exec(`INSERT INTO career_role_analyses(role_key,evidence_hash,analysis,notice,created_at,updated_at) VALUES(?,?,?,?,?,?)
		 ON CONFLICT(role_key) DO UPDATE SET notice=excluded.notice,updated_at=excluded.updated_at`, role.Key, hash, "", notice, now, now)
		return dbErr
	}
	_, dbErr := a.db.Exec(`INSERT INTO career_role_analyses(role_key,evidence_hash,analysis,notice,created_at,updated_at) VALUES(?,?,?,?,?,?)
	 ON CONFLICT(role_key) DO UPDATE SET evidence_hash=excluded.evidence_hash,analysis=excluded.analysis,notice=excluded.notice,updated_at=excluded.updated_at`, role.Key, hash, raw, notice, now, now)
	return dbErr
}

func (a *App) analyzeCareerRole(w http.ResponseWriter, r *http.Request) {
	if !discoveryConsent(w, r) {
		return
	}
	if !a.discoveryMu.TryLock() {
		http.Error(w, "A Muse request is already in progress. Wait before retrying.", http.StatusConflict)
		return
	}
	defer a.discoveryMu.Unlock()
	role, ok := careerRoleByKey(r.PathValue("role"))
	if !ok {
		// Saved-opportunity profiles use keys like saved-job-7.
		var jobID int64
		if _, err := fmt.Sscanf(r.PathValue("role"), "saved-job-%d", &jobID); err != nil || jobID < 1 {
			http.NotFound(w, r)
			return
		}
		saved, err := a.jobByID(jobID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		var profileErr error
		if role, profileErr = roleProfileFromJob(saved); profileErr != nil {
			http.Error(w, "This opportunity cannot be compared yet: "+profileErr.Error()+". Add requirement bullets to its description.", 422)
			return
		}
	}
	evidences, err := reviewedRoleEvidences(a)
	if err != nil || len(evidences) == 0 {
		http.Error(w, "Review at least one Career Evidence record before comparing roles.", 422)
		return
	}
	if err := a.runRoleAnalysis(r.Context(), role, evidences); err != nil {
		http.Error(w, "Unable to save analysis", 500)
		return
	}
	http.Redirect(w, r, "/career-roles", http.StatusSeeOther)
}

// createRoleOpportunity inserts a local synthetic job for a role. The
// candidate already owns this action on the jobs page; this only pre-fills it.
func (a *App) createRoleOpportunity(w http.ResponseWriter, r *http.Request) {
	if !parseLocalForm(w, r, 4<<10) {
		return
	}
	role, ok := careerRoleByKey(r.PathValue("role"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	if existing, err := a.roleOpportunity(role.Key); err == nil && existing != nil {
		http.Redirect(w, r, fmt.Sprintf("/jobs/%d", existing.ID), http.StatusSeeOther)
		return
	}
	descriptions := map[string]struct{ title, company, description string }{
		"support-engineer": {"Technical Support Engineer", "Example Internal Platform Co.", "Own Linux troubleshooting and MySQL diagnostics for production incidents. Diagnose slow queries and system issues, document findings, and escalate clearly to engineering. Communicate status to internal customers during incidents."},
		"account-manager":  {"Technical Account Manager", "Example Customer Platform Co.", "Serve as the technical face for customer accounts. Explain technical issues, coordinate support and engineering follow-through, and track escalations. Own account communication; production database changes stay with engineering."},
	}
	seed := descriptions[role.Key]
	result, err := insertJob(a.db, Job{URL: "https://example.test/demo/" + role.Key, Host: "example.test", Title: seed.title, Company: seed.company, Description: seed.description, SourceStatus: "unsupported", SourceNote: "Synthetic demo opportunity; never submitted."})
	if err != nil {
		http.Error(w, "Unable to create opportunity", 500)
		return
	}
	id, _ := result.LastInsertId()
	http.Redirect(w, r, fmt.Sprintf("/jobs/%d", id), http.StatusSeeOther)
}
