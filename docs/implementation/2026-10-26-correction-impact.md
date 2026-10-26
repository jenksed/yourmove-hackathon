# Correction impact plan (HACK-005)

## Current state

- Saved answers (`application_answers`, `early_application_answers`,
  `reusable_answers`) store answer text plus an optimistic-revision counter.
  They store no evidence references. The generation snapshot (with validated
  career-evidence id/revision/created) lives only in the ephemeral ticket
  registry (10–30 minute expiry).
- Cover letters pin `evidence_refs` (id, revision, created) and
  `coverLetterEvidenceStatus` reports stale vs current. Drafts are never
  rewritten.
- Submissions record resume identity and notes only. Employer-portal content
  is outside YourMove. Flags therefore cannot rewrite submitted history.

## Minimal durable relationship

New sidecar table `ai_answer_provenance`, written server-side when a
generation completes, keyed to the saved-answer natural key
`(job_id, destination, question_key)`:

- one row per selected evidence: id, revision, created-at
- `answer_hash`: hash of the generated answer text
- `created_at`: generation time

No existing table changes. No backfill: absence of rows means
"provenance not recorded", never an error.

## Correction behavior

- Evidence edit bumps its revision. Readers compare pinned revision/created
  against the current record.
- Saved answer text whose hash matches a provenance row: revision mismatch
  renders "needs review — supporting evidence changed". No text is altered.
- Saved text whose hash matches no row: "edited after generation —
  references describe the generation basis, not the current text".
- Submitted content is never described as updated; flags apply to workspace
  drafts only.

## Migration and rollback

- `CREATE TABLE IF NOT EXISTS` at startup. Rollback is dropping the table;
  readers treat a missing table as unknown provenance. Previously saved
  answers keep working with no provenance shown.

## Tests

- Complete generation pins exact evidence id/revision/created.
- Evidence correction flags the saved answer without changing its text.
- Candidate edits after adoption show edited, not grounded.
- Rows survive application restart.
