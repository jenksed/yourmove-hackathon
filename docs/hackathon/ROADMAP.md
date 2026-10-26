# YourMove — Meta Hackathon Competition Roadmap

**Competition:** Meta Global AI Developer Hackathon, Indie Developer Track  
**Window:** October 26 – November 4, 2026  
**Target:** First-place-caliber product demonstration  
**Feature freeze:** November 2  
**Engineering model:** Muse Spark 1.3 Contributor by default

## 1. North Star

Build an AI career investigator that progressively discovers, understands, and activates a person's professional experience.

The breakthrough is not generating better application text. It is creating a living, evidence-backed understanding of a candidate's career that becomes more valuable through conversation.

**The defining demonstration:**

Five ordinary résumé bullets become three or more rich, candidate-approved career stories. YourMove discovers transferable capabilities, identifies legitimate career opportunities, generates grounded application materials for two different roles, and respects corrections across everything that depends on the underlying evidence.

Judges should see a new category of career software, not another résumé chatbot.

## 2. Current Baseline — October 26

**Host:** `jenksed/yourmove-hackathon`  
**Branch:** `feat/career-discovery-b1`  
**Verified checkpoint:** `1215ad3`

**Independent engine:** `jenksed/yourmove-career-discovery`  
**Verified checkpoint:** `8dd7122`

Implemented and reported live-verified:

- Muse-powered multi-experience investigation leads.
- Candidate-selected investigations.
- Adaptive technical interviews.
- Source-linked proposed career stories.
- Candidate review and correction.
- Persistence and restart recovery.
- Contributor-backed application answers with Evidence Trace.
- Evidence-powered cover letters.
- Cover-letter evidence-revision tracking.
- Redesigned initial investigator workspace.

Open limitations include cross-role intelligence, multiple-story synthesis, answer-artifact change awareness, complete application packages, native Wails acceptance, and submission reconstruction.

Recheck repository state before changing these statuses.

## 3. Product Pillars

### P1 — Investigative Career Intelligence

Muse must behave like a skilled technical interviewer.

Develop:

- Better investigation-lead selection beyond résumé keyword repetition.
- Adaptive follow-ups based on unresolved responsibility, decisions, methods, collaboration, and outcomes.
- Multi-story investigation across a candidate's history.
- Identification of connections between distinct experiences.
- Useful clarification questions without repeated or unnecessary interviewing.
- Explicit distinctions between facts, candidate self-report, model interpretations, and unknowns.

**Stretch:** Muse notices a recurring professional strength across several unrelated experiences and invites the candidate to investigate that hypothesis.

### P2 — Living Career Understanding

Create an intuitive view of the candidate's evolving professional history.

Show:

- Reviewed career stories.
- Supported capabilities and technologies.
- Transferable skills with their supporting experiences.
- Relationships between experiences.
- Unresolved questions.
- New insights discovered through interviews.

Every important factual statement must remain inspectable.

A model interpretation cannot silently become a candidate-approved fact.

Do not introduce a graph database unless existing SQLite structures demonstrably cannot support the experience.

### P3 — Cross-Role Intelligence

Analyze the same approved history against different professional opportunities.

Initial comparison:

- Technical Support Engineer.
- Technical Account Manager.

Muse should explain:

- Why each experience is relevant.
- Which candidate actions support specific requirements.
- Which skills transfer across professional contexts.
- Which qualifications are missing or unsupported.
- Why an experience matters differently to each role.

**Stretch:** Identify an unexpected but plausible third professional direction and explain the evidence and gaps behind that suggestion.

Recommendations remain hypotheses, not promises of job fit.

### P4 — Evidence-Powered Application Studio

Unify existing résumé selection, application answers, cover letters, and evidence review into a coherent job-specific experience.

An application package should contain:

- The appropriate existing résumé.
- Relevant approved Career Evidence.
- Grounded application answers.
- A distinct, natural cover letter.
- Source-inspection controls.
- Review and readiness status.

Use current document and writing infrastructure.

Do not build a résumé generator.

Do not automatically submit employer applications.

### P5 — Challengeable Career Intelligence

The candidate must be able to ask, "Where did that come from?"

YourMove reveals the original recorded statement and its relationship to the generated material.

When evidence is corrected:

- The revision is preserved.
- Dependent materials are identified.
- Affected drafts are marked for review.
- No user-authored content is silently rewritten.
- Previously exported or submitted materials are not described as updated.

This is a critical differentiator.

Changes to source validation, persistence, approval, consent, and artifact dependencies require an explicit implementation plan, tests, independent review, and live-path acceptance.

### P6 — Exceptional Desktop and Web Experience

Move beyond administrative screens and nested forms.

Prioritize:

- An engaging, one-question-at-a-time investigation.
- Career stories presented as readable narratives.
- A visual understanding of emerging career capabilities.
- Role comparisons that explain relevance clearly.
- Application writing beside supporting evidence.
- Simple movement between discovery, review, and application preparation.
- Attractive and usable wide, compact, light, and dark layouts.

Use existing Go templates and progressive enhancement.

No frontend framework rewrite.

Preserve keyboard accessibility, applicant consent, data ownership, and recovery.

## 4. Secondary Feature — Job Discovery

Add optional in-app job discovery after the core competition experience is reliable.

Potential capabilities:

- Search supported public ATS job feeds.
- Discover relevant positions using title, skills, location, and work arrangements.
- Support additional posting sources when reliable and permitted.
- Explain potential relevance using reviewed career evidence.
- Identify gaps instead of manufacturing fit.
- Selectively import postings through existing URL intake.

Keep this secondary to Career Discovery.

Do not claim YourMove knows the best jobs until relevance quality is demonstrated through evaluation.

**Hackathon status:** Stretch. Discovery architecture may be explored in parallel, but implementation must not displace the core demonstration.

## 5. Model Engineering and Evaluation

Use real Meta inference extensively with synthetic development material.

Initial target: approximately 100–150 bounded evaluation requests, adjusted according to observed costs, remaining credits, and quality.

Evaluate:

- Investigative question quality.
- Novel useful details uncovered.
- Unsupported or invented claims.
- Responsibility and outcome accuracy.
- Source-resolution success.
- Cross-role relevance.
- Transferable-skill reasoning.
- Writing quality.
- Correction handling.
- Latency, tokens, failure rate, and estimated cost.

Benchmark multiple synthetic candidates, not only the MySQL incident.

Compare approaches using measured results.

Contributor mode is the engineering default for synthetic and non-sensitive work. Protect real candidate information and credentials. Independently review mission-critical functionality.

The goal is better intelligence, not simply consuming inference credits.

## 6. Execution Schedule

### October 26–27 — Foundation and Competition Architecture

- Verify current host and engine state.
- Record accepted milestones and remaining gaps.
- Confirm parallel work boundaries.
- Establish the multi-story career intelligence design.
- Maintain independent event-only submission provenance.
- Begin broader synthetic inference evaluation.

**Exit:** Working investigative baseline and concrete next implementation contracts.

### October 28 — Deeper Investigation

- Improve lead quality and adaptive questioning.
- Develop at least three substantive career stories from five ordinary bullets.
- Test uncertainty, ownership, and multiple experience types.
- Make emerging insights visible in the candidate interface.

**Exit:** Real Muse inference produces richer, inspectable, candidate-reviewable career history.

### October 29 — Living Career Understanding

- Present a coherent collection of candidate-approved stories.
- Surface supported capabilities and transferable skill hypotheses.
- Connect each meaningful insight to supporting evidence.
- Refine the Discovery and Career Story experience.

**Exit:** The candidate can understand what YourMove has learned and challenge important interpretations.

### October 30 — Cross-Role Intelligence

- Analyze two distinct opportunities.
- Produce evidence-grounded relevance explanations.
- Identify gaps and transferable capabilities.
- Validate that the analyses differ meaningfully without changing the facts.

**Exit:** One approved career history supports two clearly different legitimate career directions.

### October 31 — Application Activation

- Connect role intelligence to existing answer generation and cover letters.
- Present job-specific application packages.
- Preserve existing résumé selection and human approval.
- Improve writing and evidence inspection.

**Exit:** Two usable draft packages, each grounded in the same approved candidate history.

### November 1 — Challenge and Correction

- Implement durable evidence dependencies for saved answers where justified.
- Extend existing cover-letter change awareness.
- Demonstrate correction impact.
- Verify that candidate edits and documents remain intact.

**Exit:** Correcting a source visibly flags affected materials without rewriting them.

### November 2 — Full Competition Demonstration

- Integrate the complete candidate journey.
- Test real inference, persistence, and navigation.
- Complete the main UI composition.
- Record successful and failed model behaviors.
- Freeze the principal competition features.

**Exit:** Complete end-to-end browser demonstration suitable for judges.

### November 3 — Qualification and Demo Production

- Run native Wails acceptance.
- Verify compact-window and keyboard operation.
- Exercise errors, interruptions, retries, and restart recovery.
- Finalize inference quality evaluation.
- Record the three-minute demonstration.
- Reconstruct and test the event-only submission artifact.

**Exit:** Demonstration-ready build, video, technical evidence, and access plan.

### November 4 — Submission

- Confirm organizer requirements and exact deadline.
- Verify submission eligibility and preexisting-code disclosure.
- Verify judges can access the actual event-created source.
- Match the submitted implementation to the recorded demo.
- Finalize the video and description.
- Submit and retain confirmation.

Do not introduce speculative new features on submission day.

## 7. Parallel Workstreams

**Engine Intelligence:** Interviews, synthesis, relevance and evaluation.

**Product Experience:** Discovery, career stories, role comparisons and Application Studio.

**Artifact Integration:** Grounded writing, document relationships and correction-impact design.

**Integration/Quality:** Live inference, native acceptance, privacy, submission provenance and demo reliability.

Use isolated worktrees where useful. Avoid concurrent edits to shared templates or handlers.

Do not let parallelism create unnecessary integration overhead.

## 8. Competition Scorecard

Target demonstration outcomes:

- Five ordinary résumé bullets investigated.
- At least three materially richer candidate-approved stories.
- Multiple useful supported skills or transferable capabilities.
- At least one non-obvious, defensible career insight.
- Two substantially different role-relevance analyses.
- Two distinct evidence-grounded application packages.
- Exact source inspection for important factual claims.
- One visible correction affecting dependent materials.
- Persistence across application restart.
- Real Muse inference throughout the demonstrated intelligence.
- Zero unauthorized employer submissions.

Measure actual quality rather than merely counting outputs.

If model behavior fails, record it and reduce the unsupported claim rather than fabricate success.

## 9. Stretch Targets

Attempt only after the core demonstration is stable:

1. Proactive discovery of a plausible new career direction.
2. Intelligent recommendations for the next career question worth asking.
3. Evidence-informed in-app job search and selective import.
4. A more expressive interactive career capability map.
5. Portable career evidence export.
6. Optional voice-driven investigation.

These are opportunities, not completion requirements.

Cut additional integrations, generalized infrastructure, and peripheral management features before cutting investigative intelligence or candidate control.

## 10. Non-Negotiable Engineering Boundaries

- Never modify `~/Projects/yourmove-go`.
- Keep hackathon work separate from the November 6 release.
- Maintain an independently meaningful event-created engine.
- Keep the submission integration patch current and reproducible.
- Do not misrepresent inherited application code as newly created.
- No autonomous employer submissions.
- No fabricated career history or qualifications.
- No silent changes to candidate-approved writing.
- No secrets or real candidate information in synthetic Contributor runs.
- No general-purpose graph database or orchestration framework without demonstrated necessity.
- No broad frontend rewrite.
- Do not call untested behavior verified.

## 11. The Three-Minute Winning Story

**0:00–0:25 — The problem:** Five ordinary résumé bullets leave most of a person's career unexplained.

**0:25–1:10 — The investigator:** Muse finds promising stories and asks surprisingly specific follow-up questions.

**1:10–1:35 — The understanding:** Career stories and supported capabilities emerge, connected to the candidate's actual statements.

**1:35–2:15 — The activation:** The same evidence supports two different professional directions and produces distinct application materials.

**2:15–2:45 — The challenge:** Correct one important career detail. Dependent materials require review instead of silently perpetuating the error.

**2:45–3:00 — The conclusion:** YourMove helps people discover what their experience means and communicate it accurately.

## 12. Operating Rule

Ambition is measured by new user-visible intelligence, not feature count.

Every development sprint should answer:

1. What genuinely new capability does the candidate gain?
2. Why is Muse essential to it?
3. How can the candidate inspect or challenge the result?
4. How will we demonstrate it through the real application?
5. What evidence proves it works?

**Build the experience that makes judges reconsider what an AI career assistant can be.**
