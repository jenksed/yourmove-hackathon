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
