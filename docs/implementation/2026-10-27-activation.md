# Cross-role application activation (HACK-005) — 2026-10-27

Synthetic candidate, real `muse-spark-1.3` inference (Standard via
compatible endpoint, Contributor profile for discovery where noted).
Isolated workspace `/tmp/activation-ws` (removed after validation);
no real candidate data, zero employer submissions (`submissions = 0`).

## Approved Career Evidence (unchanged throughout)

- ID 1 rev 2, reviewed: billing double-charge status updates and handoff.
  Outcome unknown; refund completion unseen; rollback owned by backend team.
- ID 2 rev 2, reviewed: MySQL slow-query/EXPLAIN diagnosis, index
  recommendation handed to engineering. Never ran DDL on production;
  outcome unknown.

Both records stayed at revision 2 from approval through all generation.
No evidence edit, no re-review, no fact change. Correction-impact
behavior preserved: current provenance, no stale flags raised.

## Role intelligence (same evidence, substantively different)

- TSE: MySQL record "directly shows production MySQL diagnostics ...
  core to support-engineer focus"; billing record incident
  communication. Gaps: Linux diagnostics, DBA ownership, account
  ownership, renewals. (464 in / 2779 out, finish stop; one earlier
  attempt failed output validation and saved nothing.)
- TAM: billing record "highly relevant to customer communication and
  coordination"; MySQL record "no customer-facing explanation".
  Gaps: cross-team coordination, escalation follow-through, account
  ownership, prod DB changes. (466 in / 2359 out, finish stop.)

## TSE opportunity (Percona MySQL Database Support Engineer, job 3)

- Answer to "Please shortly describe your experience with MySQL (HA,
  backups, etc)": diagnostic-only framing, "no HA or backup work
  recorded", 8 trace claims all pinned to evidence 2 / rev 2 /
  response-1..3 with verbatim quotes; unresolved: HA experience,
  backup experience, whether the index was used. Adopt-check 200,
  saved locally as early answer rev 1. Provenance sidecar: 2 rows.
- Cover letter draft 1: both evidence refs pinned (id/rev/created),
  provider compatible / muse-spark-1.3, leads with diagnostics.

## TAM opportunity (Percona TAM Team Manager, job 4)

- Cover letter draft: both evidence refs pinned at rev 2, leads with
  customer communication and ownership boundaries — substantively
  different framing from the TSE letter using identical facts.

## TAM opportunity (Ashby Strategic Customer Success Manager, job 10)

- Answer to "In 2-4 sentences, tell us about the most complex product
  you've supported as a Customer Success Manager...": leads with the
  billing story (customer communication precision), supports with the
  MySQL story (technical-to-handoff translation), preserves "never ran
  DDL" and unknown-outcome caveats. 8 trace claims: 4 pinned to
  evidence 1 rev 2, 4 to evidence 2 rev 2, all with verbatim quotes and
  response IDs. Adopt-check on the superseded first ticket correctly
  refused (409); the completed generation saved locally as early
  answer rev 1. Provenance sidecar: 2 rows. Substantively different
  from the TSE answer (ownership-first vs diagnostics-first) on
  identical facts.
- Cover letter (draft 3, both evidence refs rev 2).
- Path here: 13 postings surveyed across 9 orgs (read-only intake +
  rendered preview, no fills). Eligible-but-ungroundable questions
  correctly returned `insufficient` (Percona referral, Coder interest
  twice). A transient provider-side output-validation cluster caused 4
  consecutive 502s across prompts (including the known-good TSE
  control); after cooldown the same TAM request completed with
  finish-equivalent success. No product code changed for the demo.
- Structural finding (no code changed): Greenhouse public_api custom
  questions carry stable numeric IDs (`question_<id>`) that the
  positional-ID guard rejects, so PagerDuty's groundable
  incident-management question stays manual-only. Refining the guard
  or the inspector ID scheme needs migration care — left for a
  planned change, not demo surgery.

## Résumé and submission boundaries

- `synthetic-resume.docx` selected for every activation job; distinct
  `synthetic-resume-master.docx` copy designated master after the
  in-use rule rejected the shared file. No résumé bytes rewritten.
- Saved answers explicitly excluded from transfer until live-form
  reconciliation; letters are local drafts. Transfer controls untouched.

## Validation

- Full host `go test ./...` PASS, `-tags desktop` PASS, engine
  `go test ./...` PASS (2026-10-27, this checkout).
- Controlled Chromium (headless Canary): roles wide + narrow, TSE
  and TAM job Prepare panels, evidence overview captured and
  inspected; saved answers shown excluded from transfer;
  "No submissions recorded".
- Restart (twice): server stopped, relaunched on the same workspace;
  evidence (2 reviewed rev 2), 2 early answers, 3 letters, 2 analyses,
  provenance rows (jobs 3, 10) all retained and served.
- Native Wails acceptance NOT VERIFIED (not claimed).
