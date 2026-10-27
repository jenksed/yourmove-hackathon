# Career Intelligence Sprint Report — October 27, 2026

Branch `feat/hackathon-baseten-backend`. Host commit `bc19de1`; engine
`career-discovery` commit `f3a930b` (lockfile pinned to match). No push, no
merge, no employer submissions. Private oracles were not used; all candidate
material below comes from public v2/v3 fixtures and the prior evaluation
report. A live-validation pass ran the same day against Baseten
(`meta-models/muse-spark-open-1026`, keys from the ignored project `.env`);
its results are recorded inline below and supersede the no-key limitation
wherever stated. All candidate answers were written from public v2/v3
fixtures only; no evaluator was available, so hidden-fact discovery stays
unresolved. Native Wails remains NOT VERIFIED (browser-host verified
end-to-end; desktop-tagged code vets clean).

## 1. Most significant investigative improvement

One-sentence addition to the engine interview `Policy`
(`career-discovery/discovery.go`): when recorded answers report conflicting
dates, figures, or documents, the next question reconciles that conflict
before continuing the prior thread, adopting neither figure.

Stage 1 diagnosis: the C09 failure was a reasoning/instruction gap, not a
context gap. `discovery.Next` sends the full resume plus every recorded
response in one request, so both sides of the 2021-vs-2022 / 18% conflict
reached the model; nothing told it to prefer the conflict over its current
thread. Q3/Q4 ignored two disclosures; only explicit redirection produced
the exemplary Q5. Tests: `TestInterviewPolicyReconcilesReportedContradictions`
plus `TestContradictionContextReachesInference`, which replays the C09 shape
(resume date vs draft date plus metric) through a stub and asserts both sides
plus the reconcile direction reach inference. Both fail on the pre-fix policy
(verified via stash) and pass after. Live before/after comparison against the
current implementation still needs keyed inference and is recorded unresolved.

## 2. Most significant cross-domain capability

Saved fictional jobs now compare through the same `AnalyzeRole` engine path
as the hardcoded TSE/TAM roles (`internal/app/career_roles.go`).
`roleProfileFromJob` builds the profile from the job's own title and
requirement bullets; prose without bullets is refused (422) rather than split
into invented requirements. The roles page lists every parseable saved job
with its own analysis, unmet list, and opportunity link.
`TestSavedJobComparisonUsesActualRequirements` runs two different-domain
fictional jobs (Long-Term Care LPN, Logistics Coordinator) against one
reviewed record with a stubbed model: the LPN reads non-relevant with all
requirements unmet, logistics reads transferable-relevant with the forklift
certification honestly unmet. `TestParseJobRequirementsBoundaries` guards
dedupe, numbering styles, the 8-requirement cap, and prose refusal.

## 3. Strongest Career Mirror moment

New `#mirror` section on Studio (`templates/career_studio.html`), template-only,
reusing the existing synthesis payload: what reviewed records establish
(quotes), what Muse uncovered across them, what it could mean for
applications (`candidate_use` with a link to roles), and what still is not
established (limitations), each with source links and an anchor to the
challenge controls. The strongest honest moment is the pair it demonstrates:
`TestCareerMirrorShowsDiscoveryAndRestrainedAbstention` shows the discovery
rendering, then re-runs synthesis with zero defensible connections and shows
the mirror stating nothing is presented as a discovery.

## 4. Clearest refused unsupported interpretation

A prose-only opportunity ("exciting growth, competitive culture") cannot be
compared: no requirement boundary exists, so the handler refuses instead of
inventing one, and the roles page omits it. At the engine layer, `AnalyzeRole`
already rejects any `unmet` entry not echoed from the supplied requirements,
so a model cannot smuggle in new qualifications.

## 5. What did not improve

- Live contradiction pursuit is now observed (see §11): new policy
  reconciles unprompted 2/2; hidden-fact discovery with an evaluator still
  unresolved.
- Synthesis novelty still depends on what interviews surface; no change made
  to follow-up depth beyond the contradiction rule.
- No new role profiles, résumé generation, job board, or evidence states.
- Native Wails acceptance still not established (see §11). Meta-direct
  inference still 401 in prior sessions; not re-probed (Baseten used live).

## 6. Workflow evidence and tests

- `go build ./...` clean; host `go test -count=1 ./...` all packages PASS
  (app suite ~25s); engine `go test ./...` PASS.
- Prior agents' D1/D2 regression tests re-run green here:
  `TestStudioKeepsSavedSynthesisVisibleAfterDraftDemotion`,
  `TestDiscoveryLimitNoticeWithoutProposal/WithProposal`.
- New tests: 2 engine (Bet A), 4 host (Bet B: 1 slice + 1 refusal + 1
  parser; Bet C: 1 mirror pair). Pre-fix failure demonstrated for Bet A
  (stash) and Bet B (new symbols absent pre-fix).
- No inference tokens spent; no latency to report.

## 7. Code changes and commits

- Engine `f3a930b`: `discovery.go` (+1 policy sentence), `discovery_test.go`
  (+44 lines). Pushed nowhere.
- Host `bc19de1`: `career_roles.go` (+~150: profile derivation, page
  entries, handler branch), `career_roles_test.go` (+~130),
  `career_connections_test.go` (+~55), `templates/career_studio.html`
  (+~25 mirror section, +1 anchor id), `career-discovery.lock` (pin
  `f3a930b`). Other agents' worktree changes (answer/letter policy lines,
  BACKLOG, `.gitignore`, D2 discovery template+tests, untracked
  docs/testdata) left untouched and uncommitted.
- Submission patch `integration/yourmove.patch` deliberately NOT
  regenerated: it already carries prior work and a partial regen from only
  staged files would drop unstaged entries. Regen belongs with the release
  finalization pass once the tree settles.

## 8. Remaining blockers

- Inference credentials for live validation (Meta 401 history; Baseten key
  absent here). The C09 old-vs-new policy comparison and any genuine
  hidden-fact discovery need the evaluator path, never oracle inspection.
- Patch regen + full baseline-apply verification deferred (see §7).
- BACKLOG entry for this sprint left for the owner to merge (file carries
  another agent's uncommitted changes); suggested text is appended below.

## 9. Owner review exercises

### Review A — Blind Insight Comparison (C09 Darius Coleman, public fixture)

Initial résumé statements
(`YourMove_Synthetic_Candidate_Pool_v2/candidates/C09-darius-coleman/resumes/markdown/master_resume.md`):
Account Support Representative, Silvercreek Utility Services, 2022–Present;
"Investigate billing corrections and explain account adjustments to
customers"; "Maintain weekly exception summaries". No metric, no start-date
ambiguity.

Archived draft excerpt (same directory, `interviews/archived_draft_excerpt.md`):
same role, 2021–Present, plus "billing adjustments fell around 18% after
process changes", marked not reconciled.

Findings after investigation (prior live eval, Baseten): stories 2–3
proposals kept responsibility ending at handoff, preserved stated limits
verbatim, listed Unknown outcome and explicit unresolved items; synthesis
found "building complete verification packets for downstream handoff" across
employers with exact quotes and a "does not establish" section. The
date/metric conflict itself stayed unresolved — the model never adopted
either figure, and only an explicit redirect produced the correct
verification question (onboarding record, pay stub, supervisor confirmation).

Mirror result: the cross-story handoff pattern is the candidate-visible
discovery (inspectable in Studio with quotes and limits); the 2021-vs-2022
and 18% questions are shown as still-unestablished, not as findings.

Question for the owner: which insight, if any, reveals something genuinely
new about this person's work, and which would actually help an application?

### Review B — Skeptical Candidate Challenge (C14 metric, public fixture)

Plausible but overreaching interpretation: "Owen reduced handoff delays —
his spreadsheet cut weekly volumes from 12 to 7." Actual sources: a rough
self-reported before/after estimate from his own spreadsheet, with a staffing
change confound and unknown supervisor use. The evaluated proposal correctly
kept every qualification ("rough", self-reported, small-sample, confound,
"supervisor only thanked") and listed causality as unresolved; synthesis
denied ownership and causality in "does not establish".

Question for the owner, with the Studio challenge controls open: would you
trust the reduced-delays sentence about your own career? The evidence you
should demand before approving it: how the before/after was measured, what
changed besides the spreadsheet, and what the supervisor did with it — all
currently in the unresolved list, none in the claim. Use Questionable or
Unsupported if it falls short; challenging never rewrites the record.

## 10. What v5 should challenge (without creating or accessing v5)

- Withheld-fact discovery through the evaluator path only: plant conflicts
  the résumé never states and check spontaneous pursuit, not redirected
  pursuit.
- Domains where the honest answer is large-scale mismatch (nursing, trades,
  culinary vs tech roles) to confirm refusal generalizes beyond C16/C21.
- Multilingual provenance (C40 pattern): translated evidence must keep
  quotes exact and flag translation as a qualification boundary.
- Abstention pressure: histories sharing only generic vocabulary must yield
  empty connections, not weak ones.
- Correction cascades: a mid-journey correction must stale every dependent
  synthesis, role analysis, answer, and letter without rewriting any of them.

## 11. Live validation addendum (same day, Baseten)

Harness: host binary with committed `bc19de1`/`f3a930b`, three isolated
`YOURMOVE_DATA_DIR` workspaces, scripted HTTP journeys plus headless-Chromium
(gstack) screenshots. Candidate answers written from public fixtures only.
Raw journey logs: `/tmp/run-old1.json`, `/tmp/run-old2.json`,
`/tmp/run-new2.json`, `/tmp/journey.json`, `/tmp/journey2.json` (session
scratch, not committed). Live workspace `/tmp/ym-live` retained for review.

**Bet A before/after (C09 shape, 2 runs per policy, identical inputs).**
Old policy Q2 acknowledged the conflict then adopted the résumé figure
("we'll stay with your current resume as listed") and continued the prior
thread; no verification question in either run. New policy Q2 named both
figures and asked which start date is accurate and whether the 18% reflects
the candidate's own work, in both runs; Q3/Q4 pressed for a verification
source (HR record, offer letter, pay history) and for confirmation the 18%
is not attributable work. One model output failed source/format validation
mid-journey (1484 in / 1104 out via Baseten) and the retry produced the
proposal — the known transient pattern, input preserved. Proposal unresolved
list (live): start year unknown with neither date adopted, correction outcome
unknown, 18% explicitly not measured/confirmed. Continued uncertainty was the
correct outcome and was retained, not resolved by invention.

**Bet B (two candidates × two jobs, all live).** C09 billing history vs J25
Billing Support Specialist (direct fit, limitations noted) and J49 LPN
(both records not relevant, all four requirements unmet, no laundering of
case-note handoff into clinical handoff). C21 apprentice history (one live
interview, 4 answers, proposal reviewed) vs J61 Electrical Apprentice
(direct match with coursework/authority limits) and J75 Senior EV
High-Voltage Master Technician (all records not relevant, all six
requirements unmet, continuity checks not upgraded into HV qualification).
TSE/TAM demo paths re-verified on the same history with differentiated
readings. Résumé selection: C09 billing PDF selected for J25; J49 left
without a résumé (mismatch identified, nothing unfitting attached).
Grounded cover letter generated for J25 (provider `compatible`, Baseten,
draft 1, pinned to evidence revisions): actions and boundaries grounded with
exact-quote discipline — but it asserts "since 2022" for tenure while the
reviewed evidence marks the start year unknown. That is a real
claim-hierarchy finding: résumé dating overrode an explicit reviewed
unknown. Letter stayed a draft; nothing submitted.

**Bet C / Stage 5 (same live workspace).** Synthesis (2 C09 records) yielded
one connection ("Documenting account-history findings into clear customer
explanations and field-operation handoffs") with exact quotes, first-person
candidate_use, and three limitations; the mirror rendered all four rows plus
challenge controls (screenshots `/tmp/ym-mirror.png`, `/tmp/ym-roles.png`).
Challenge (`questioned` + note) persisted. Correcting record 1 ("last field
visit note" → "most recent field visit note I could access") demoted it to
draft rev 3; Studio showed the stale banner with the saved synthesis, and
the roles page showed refresh guidance with saved analyses withheld for the
changed set. Server restart: synthesis, challenge, all six role analyses,
jobs, résumés, master selection, and the letter draft survived; verified
over HTTP.

**Browser vs Wails.** All of the above ran through the real HTTP UI, with
Studio and roles pages additionally exercised in headless Chromium
(screenshots inspected pixel-level). Native Wails: not verified — no Wails
CLI in this environment, so the bundle was never launched; `go vet -tags
desktop ./...` is clean.

**Not improved / still open.** Hidden-fact discovery (no evaluator
channel); synthesis novelty beyond interview content; the letter tenure-date
finding needs a policy or validator fix; one prose-form 422 round-trip
needed a delete-form field stripped (harness issue, not product).

## Suggested BACKLOG entry (paste on settle)

- **Career-intelligence sprint (2026-10-27, live-validated on Baseten):**
  engine contradiction-reconciliation policy + plumbing tests (`f3a930b`,
  lock pinned); saved-job cross-domain comparison reusing AnalyzeRole with
  prose refusal; Studio Career Mirror with discovery + abstention states.
  Host `bc19de1`, suites green, no push. Live before/after: new policy
  reconciles the C09 conflict unprompted 2/2, old policy adopts the résumé
  2/2. Live matrix: J25 fit / J49 gap (C09), J61 fit / J75 gap (C21);
  grounded J25 letter drafted with one tenure-date finding. Stages 5 +
  correction cascade + restart verified over HTTP and headless Chromium.
  Patch regen and native Wails still open; letter date-hierarchy fix new.
