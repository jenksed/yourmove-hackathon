package app

import (
	"database/sql"
	"strings"
)

// AI-answer provenance is a read-only flag source. It never rewrites a saved
// answer, never describes submitted history, and treats missing rows as
// unknown rather than an error.
func initializeAnswerProvenance(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ai_answer_provenance (
	 job_id INTEGER NOT NULL, destination TEXT NOT NULL DEFAULT '', question_key TEXT NOT NULL DEFAULT '',
	 evidence_id INTEGER NOT NULL, evidence_rev INTEGER NOT NULL DEFAULT 0, evidence_created TEXT NOT NULL DEFAULT '',
	 answer_hash TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL)`)
	return err
}

// recordAnswerProvenance replaces the proposed basis for one question with
// the completed generation's validated evidence selection. It runs after a
// generation completes and never blocks serving the answer.
func (a *App) recordAnswerProvenance(snapshot answerGenerationSnapshot, answerHash, now string) {
	if snapshot.JobID < 1 || snapshot.QuestionKey == "" || answerHash == "" {
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM ai_answer_provenance WHERE job_id=? AND destination=? AND question_key=?`, snapshot.JobID, snapshot.Destination, snapshot.QuestionKey); err != nil {
		return
	}
	for _, source := range snapshot.Sources {
		if !strings.HasPrefix(source.Category, "Applicant-reviewed Career Evidence") || source.ID < 1 || source.Revision < 1 || source.CreatedAt == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO ai_answer_provenance(job_id,destination,question_key,evidence_id,evidence_rev,evidence_created,answer_hash,created_at) VALUES(?,?,?,?,?,?,?,?)`,
			snapshot.JobID, snapshot.Destination, snapshot.QuestionKey, source.ID, source.Revision, source.CreatedAt, answerHash, now); err != nil {
			return
		}
	}
	_ = tx.Commit()
}

type answerProvenanceFlag struct {
	QuestionKey, Label string
	State              string // grounded, needs-review, edited, no-provenance
	Detail             string
}

// savedAnswerProvenance compares saved workspace answers against the pinned
// generation basis. Missing rows report no-provenance, never an error.
func (a *App) savedAnswerProvenance(job Job) []answerProvenanceFlag {
	type saved struct {
		key, label, answer string
	}
	var answers []saved
	rows, err := a.db.Query(`SELECT question_key,label,answer FROM application_answers WHERE job_id=? AND destination=?`, job.ID, job.URL)
	if err == nil {
		for rows.Next() {
			var s saved
			if rows.Scan(&s.key, &s.label, &s.answer) == nil {
				answers = append(answers, s)
			}
		}
		rows.Close()
	}
	early, err := a.db.Query(`SELECT COALESCE(live_question_key,''),question_label,answer FROM early_application_answers WHERE job_id=? AND destination=?`, job.ID, job.URL)
	if err == nil {
		for early.Next() {
			var s saved
			if early.Scan(&s.key, &s.label, &s.answer) == nil {
				answers = append(answers, s)
			}
		}
		early.Close()
	}
	var flags []answerProvenanceFlag
	for _, s := range answers {
		flag := answerProvenanceFlag{QuestionKey: s.key, Label: s.label, State: "no-provenance", Detail: "Saved without recorded evidence basis."}
		hash := answerAIHash(s.answer)
		type pin struct {
			evidenceID, evidenceRev int64
			evidenceCreated         string
			answerHash              string
		}
		var pins []pin
		prows, err := a.db.Query(`SELECT evidence_id,evidence_rev,evidence_created,answer_hash FROM ai_answer_provenance WHERE job_id=? AND destination=? AND question_key=?`, job.ID, job.URL, s.key)
		if err == nil {
			for prows.Next() {
				var p pin
				if prows.Scan(&p.evidenceID, &p.evidenceRev, &p.evidenceCreated, &p.answerHash) == nil {
					pins = append(pins, p)
				}
			}
			prows.Close()
		}
		if len(pins) == 0 {
			flags = append(flags, flag)
			continue
		}
		matched := false
		for _, p := range pins {
			if p.answerHash == hash {
				matched = true
			}
		}
		if !matched {
			flag.State = "edited"
			flag.Detail = "Edited after generation. References below describe the generation basis, not the current text."
			flags = append(flags, flag)
			continue
		}
		stale := []string{}
		for _, p := range pins {
			if p.answerHash != hash {
				continue
			}
			current, err := careerEvidenceByID(a.db, p.evidenceID)
			if err != nil || current.Revision != p.evidenceRev || current.CreatedAt != p.evidenceCreated {
				stale = append(stale, current.Title)
			}
		}
		if len(stale) > 0 {
			flag.State = "needs-review"
			flag.Detail = "Supporting evidence changed since generation: " + strings.Join(stale, "; ") + ". Text unchanged."
		} else {
			flag.State = "grounded"
			flag.Detail = "Matches the generated text; supporting evidence unchanged."
		}
		flags = append(flags, flag)
	}
	return flags
}
