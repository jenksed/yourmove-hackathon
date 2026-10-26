# Career Discovery — October 26 checkpoint

**Host feature:** `2465d18` · **AI Connections cleanup:** `e753a0e` · **Engine:** `1858750f32d466852c9c7ffdf64e37387cd50813`.

The implementation, compliance assessment, live evidence, remaining schedule, submission checklist and next-action handoff are in the fresh engine repository's [engineering report](https://github.com/jenksed/yourmove-career-discovery/blob/1858750f32d466852c9c7ffdf64e37387cd50813/docs/EXECUTION.md). Its authoritative local copy is `career-discovery/docs/EXECUTION.md`.

The original host started clean at isolation commit `c68051349c1d642b83c514a3377555afa9941542`. `yourmove-go` was not modified or made a runtime dependency. The inherited host repository is **not** the submission artifact. The new private repository has one fresh root commit and no inherited application history/source. Its [discovery-only integration patch](https://github.com/jenksed/yourmove-career-discovery/blob/1858750f32d466852c9c7ffdf64e37387cd50813/integration/yourmove.patch) contains new files and zero-context additions, with no deleted preexisting lines. AI Connections cleanup is host-only and excluded from that patch.

## Resume development

The host's `career-discovery/` directory is an ignored independent Git checkout, not a second copy of the engine. `go.mod` uses it directly; `career-discovery.lock` pins the tested revision. After a fresh host clone:

```sh
git clone https://github.com/jenksed/yourmove-career-discovery.git career-discovery
git -C career-discovery checkout 1858750f32d466852c9c7ffdf64e37387cd50813
go test -count=1 ./...
go test -count=1 -tags desktop ./...
go build -o /private/tmp/yourmove-career-discovery-b1 .
python3 scripts/run-discovery-demo.py /private/tmp/yourmove-career-discovery-b1 /private/tmp/yourmove-discovery-b1-workspace
```

Place `MODEL_API_KEY` in the ignored `.env` or launch environment. The launcher reads only that variable without shell evaluation. It uses an explicit synthetic workspace and port 3019. Open `/ai` for Meta readiness/setup and `/career-discovery` for interviews. Do not reinitialize the workspace when verifying restart persistence.

Current native build (build/signature verification only):

```text
/private/tmp/yourmove-discovery-b1-final-native/YourMove Hackathon Edition.app
```

Build a new native artifact with `scripts/build-hackathon-macos.sh /absolute/new/output-directory`. The new workflow still needs native Wails interaction acceptance; the previously running native app was not replaced or stopped.

## Observed acceptance

- Real standard-mode Muse questions and adaptive follow-ups through the full browser app, using synthetic data.
- New MySQL full-table-scan and diagnostic-handoff detail beyond the sparse résumé; fix ownership and measured outcome left unresolved.
- Exact candidate-source inspection; proposed evidence opened in the existing editor; corrected and explicitly reviewed; reviewed content and original transcript survived a process restart.
- Standalone synthetic CLI ran live `muse-spark-1.3-contributor`, in 3.67 seconds, with 363 input / 353 output tokens (about $0.000107 estimated).
- Meta-first AI Connections and its settings disclosure observed in Chrome. Other services grouped into concise native disclosures.
- Complete ordinary and desktop-tagged Go suites PASS; independent module suite PASS; native bundle build/signing PASS. Narrow and native visual acceptance NOT VERIFIED.
- Muse Code performed the requested bounded review. Its useful retry-handling issue was fixed; source rebinding was added at persistence; its ordinary-provider-failure orphan claim was not reproduced. Regression tests pass.

One initial live follow-up was rejected by provider output validation. Input was retained, no evidence was approved, and an explicit retry succeeded after setting Muse reasoning/output controls. Its original raw failure cause and usage were not captured. Measured standard browser requests ranged from 2.9 to 9.4 seconds. Four measured browser calls cost about $0.02117; this is not a complete account-spend total.

## Handoff

**Working now:** browser B1 discovery/review/persistence, source-linked proposals, independent Contributor synthetic runner and concise Meta-first AI Connections.

**Implemented but unverified:** native interaction and narrow-layout behavior; broader model reliability. Job-answer source infrastructure is preexisting but has not been qualified with this new evidence and Meta.

**Blocked:** no API/credential blocker. Ask organizers whether a disclosed preexisting demo host and event-only integration patches are permitted. The public event page did not expose the supplied FAQ or final rubric. Arrange judge access to the private repository before submission.

**Decisions needed:** no product decision before the next implementation slice. Final visibility, organizer interpretation and owner-controlled submission remain outstanding.

**Next implementation action:** activate one reviewed MySQL experience for a real job-specific question using the existing application-answer path and Muse; make one answer claim open its actual stored source. Demonstrate insufficient evidence for an unsupported qualification. Then finish durable rejection/clarification, native acceptance and the small synthetic evaluation before recording.

Contributor is preferred for synthetic/public development and future code reviews. Candidate-facing discovery stays in standard mode. No autonomous employer submission, source merge, release or hackathon submission was performed.

## Career Evidence → application-answer sprint addendum

This addendum supersedes the “Next implementation action” status above for the October 26 answer-trace sprint. Baseline at sprint start was host `29c368dc085a8c17968a485c9253c527801a01f6` on `feat/career-discovery-b1`, and event engine `1858750f32d466852c9c7ffdf64e37387cd50813` on `main`.

- The engine added `AnswerTrace`, `TraceSource`, and fail-closed validation of exact evidence/revision/response identities and literal quote resolution. Engine code commit: `47717b0e12272a1c91ba56fb2c7af9b84d481a99`.
- The host now binds reviewed evidence to immutable interview responses, requests structured output through the configured compatible provider, and renders an ephemeral unsaved Evidence Trace disclosure. The one-way integration snapshot and updated execution docs are in the current event-engine HEAD. Host lock remains pinned to engine code commit `47717b0e12272a1c91ba56fb2c7af9b84d481a99`; host sprint commit is pending.
- Muse CLI was `v1.4.13`; `muse exec` exposed workspace/worktree, explicit model, reasoning effort, step limit and approval controls. A bounded isolated worktree was created for the event-only engine. Muse execution did not reach model steps: model catalog TLS transport failed, then local session/keychain setup failed (`OSStatus -50`). No worker changes or commit were accepted; no credentials were printed. The delegation attempt added overhead and no implementation value.
- The actual browser used synthetic candidate material and a public Percona MySQL application question with the configured Muse Spark 1.3-compatible endpoint in Standard mode. Three bounded provider calls completed, but each response failed the host's strict structured answer/trace gate. The last request set `response_format: {"type":"json_object"}`. The raw provider response was deliberately not retained; the current user-facing error does not identify whether JSON parsing, answer-policy/format, or source-reference validation failed. The exact branch is therefore NOT VERIFIED. No draft was saved/adopted and no employer form was filled or submitted. Model token/cost metrics were not observed in the host flow.
- Full host `go test -count=1 ./...` passed after allowing existing loopback test servers; focused trace/answer tests and the compatible JSON-request contract passed. The standalone engine suite passed. A browser capture did verify Career Evidence ID 1 revision 2 and the selected Percona question, but no successful answer/trace appeared.

**Current state:** trace validation, host adapter, and candidate-facing disclosure are implemented; the core real-model answer demonstration remains **NEEDS LIVE VERIFICATION** due to the provider-output contract failure. Do not call this milestone complete until a real Muse response produces an answer with a resolvable source and a visible unresolved unsupported qualification.

## Answer-trace repair (October 26, code-level; live acceptance still open)

**Verified failure contributors (in code, not assumed).** The trace-path request combined two contradictory app-owned output directives: `answerWritingPolicy` ("Return only the answer …") and the resource `prompts/application_answer.md` ("Return only the proposed answer.") against `answerTracePolicy` ("Return one JSON object only …"). A model obeying the prose directive fails `parseGeneratedAnswerTrace`; a model obeying the JSON directive still faced the strict verbatim-quote/exact-substring gate. Separately, the trace JSON request used `max_tokens: 1024` with no `reasoning_effort`, while the discovery call that succeeds uses `max_tokens: 2048` plus `reasoning_effort: low` — with `response_format: json_object`. A truncated JSON envelope parses as `trace_parse`, and compatible endpoints may omit `finish_reason`, so truncation is not always caught at the provider layer. The exact live branch remains unknown (raw output deliberately not retained), but every rejection class now reproduces synthetically through the real handler.

**Smallest correction applied (host only; engine untouched).**
- `internal/app/application_answer_ai.go`: new `answerTraceWritingPolicy` keeps every grounding rule but replaces the prose-only return directive with one JSON-envelope directive plus precedence wording, used only when trace sources are bound. The plain-text path is unchanged.
- `internal/coverletter/provider/providers.go`: `GenerateNarrativeJSON` on the compatible provider now mirrors the discovery profile (`max_tokens: 2048`, `reasoning_effort: low`, `response_format: json_object`). Other providers and the plain narrative path are unchanged.
- Stage-specific, privacy-safe diagnostics: output failures now return `failure_stage` (`trace_parse`, `answer_output`, `answer_format`, `trace_shape`, `trace_identity`, `trace_quote`, `trace_claim`, `trace_unresolved`) with a `failure_detail` of byte/item counts only. `classifyTraceReference` mirrors `discovery.ValidateAnswerTrace` for labeling only; acceptance still goes through the engine validator, whose strict source validation and the independent event-only submission repository are preserved. Rejected model output and candidate information are never retained (ticket holds the stage plus counts; nothing is logged). `static/ui.js` appends the stage code to the status text.
- Unsupported HA/backup qualifications: unchanged behavior, now exercised end to end — the trace policy requires unsupported qualifications, ownership details, and unknown outcomes in `unresolved`, and the candidate reviews them before any explicit adoption. Nothing is saved, adopted, or submitted without candidate action.

**Verified this session.** New failing-first tests failed before the fix with the exact live symptom (generic 422, no stage) and pass after: `TestNarrativeTraceInstructionsUseSingleOutputFormat`, `TestNarrativeTraceFailureStages` (11 rejection fixtures, including a no-retention assertion on rejected output), `TestGenerateNarrativeJSONMirrorsDiscoveryProfile`. Full host `go test -count=1 ./...` PASS, standalone engine suite PASS, `go vet` clean on changed packages. A well-formed synthetic trace envelope still completes through the real generate handler (`TestNarrativeCareerDiscoveryEvidenceTrace`).

**Remaining manual acceptance steps (not done — no model credentials in this environment: no `MODEL_API_KEY`, no compatible endpoint, local Ollama unavailable).**
1. Configure the Muse Spark 1.3-compatible endpoint, open the real application browser flow with synthetic career material, review one Career Evidence record, and generate one job-specific answer.
2. Confirm the answer appears with an inspectable Evidence Trace: at least one claim opens the exact recorded response from the reviewed record/revision.
3. Confirm an unsupported qualification (HA, backups, implementation ownership, unknown measured outcome) appears under unresolved and that insufficient-evidence input yields `insufficient` rather than prose.
4. If generation still fails, record only the reported `failure_stage` and counts — do not paste model output — and continue from that stage.
5. Candidate explicitly reviews/adopts; zero employer submissions; then proceed to native Wails acceptance and persistent reject/clarify as separate follow-ups.

**Next three actions:** (1) add safe branch-level diagnostics and a synthetic response fixture for every output rejection class, then fix the observed provider contract; accept only after one live browser response displays a cited original response; (2) exercise the same flow in native Wails and record exact candidate review/adoption and cancellation behavior; (3) implement persistent candidate rejection/clarification and prove that edits invalidate/rebuild trace assumptions. Separately verify the competition judge path and capture evidence only after the live trace succeeds.
