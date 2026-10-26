package app

import (
	"path/filepath"
	"testing"

	"github.com/jenksed/yourmove-go/internal/storage"
)

func provenanceSnapshot(job Job) answerGenerationSnapshot {
	return answerGenerationSnapshot{JobID: job.ID, Destination: job.URL, QuestionKey: "q-1",
		Sources: []answerAISource{{Category: "Applicant-reviewed Career Evidence", ID: 1, Revision: 2, CreatedAt: "t0", Content: "{}"}}}
}

func TestAnswerProvenanceFlagsCorrectionWithoutRewriting(t *testing.T) {
	a, dir := testWorkspace(t)
	if _, err := a.db.Exec(`INSERT INTO career_evidence(title,situation,actions,outcome_state,status,revision,created_at,updated_at) VALUES('Diagnosis','Outage','Checked logs','unknown','reviewed',2,'t0','t0')`); err != nil {
		t.Fatal(err)
	}
	if _, err := insertJob(a.db, Job{URL: "https://example.test/demo/support-engineer", Host: "example.test", Title: "Support", Company: "Example", Description: "Troubleshoot.", SourceStatus: "unsupported"}); err != nil {
		t.Fatal(err)
	}
	job, err := a.jobByID(1)
	if err != nil {
		t.Fatal(err)
	}
	a.recordAnswerProvenance(provenanceSnapshot(job), answerAIHash("Generated answer text"), "t1")
	if _, err := a.db.Exec(`INSERT INTO application_answers(job_id,destination,provider,question_key,label,answer,revision) VALUES(?,?,'compatible','q-1','Describe an incident','Generated answer text',1)`, job.ID, job.URL); err != nil {
		t.Fatal(err)
	}
	flags := a.savedAnswerProvenance(job)
	if len(flags) != 1 || flags[0].State != "grounded" {
		t.Fatal(flags)
	}
	// Candidate corrects the record: revision bumps, saved text untouched.
	if _, err := a.db.Exec(`UPDATE career_evidence SET actions='Checked logs and more',revision=3 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	flags = a.savedAnswerProvenance(job)
	if len(flags) != 1 || flags[0].State != "needs-review" {
		t.Fatal(flags)
	}
	var saved string
	if err := a.db.QueryRow(`SELECT answer FROM application_answers WHERE job_id=?`, job.ID).Scan(&saved); err != nil || saved != "Generated answer text" {
		t.Fatal("saved answer was rewritten", saved, err)
	}
	// Candidate edits the answer after adoption.
	if _, err := a.db.Exec(`UPDATE application_answers SET answer='Edited text',revision=2 WHERE job_id=?`, job.ID); err != nil {
		t.Fatal(err)
	}
	flags = a.savedAnswerProvenance(job)
	if len(flags) != 1 || flags[0].State != "edited" {
		t.Fatal(flags)
	}
	// Restart persistence: rows outlive the App.
	if err := a.db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := storage.OpenDB(filepath.Join(dir, "jobs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ai_answer_provenance`).Scan(&n); err != nil || n != 1 {
		t.Fatal("provenance did not survive restart", n, err)
	}
}
