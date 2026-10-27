# Baseten Temporary Hackathon Backend

Baseten is a temporary hackathon inference backend, not a permanent YourMove
dependency. It reuses the existing OpenAI-compatible transport
(`runProviderProfile`); there is no Baseten HTTP client and no Baseten SDK.

Every temporary line is marked `HACK-BASETEN`. Find them with:

```sh
grep -rn "HACK-BASETEN" --include="*.go" internal/
```

## Enable / disable

Enable (hackathon host only; the key lives in the ignored `.env` and is never
printed, committed, or sent to the browser):

```sh
HACKATHON_MUSE_BACKEND=baseten ./yourmove
```

Disable (restores the Meta route with no settings or data changes):

```sh
unset HACKATHON_MUSE_BACKEND
```

Unset or `HACKATHON_MUSE_BACKEND=meta` uses Meta's direct endpoint
(`https://api.meta.ai/v1/chat/completions`, model `muse-spark-1.3`) with
`MODEL_API_KEY`. Baseten uses `https://inference.baseten.co/v1/chat/completions`
with the restricted model `meta-models/muse-spark-open-1026` and
`BASETEN_API_KEY`. A missing Baseten key is an error; selection never falls
back to the other account.

## Workflows served

All Career Discovery Muse traffic flows through `ConfigureMeta`, so the
override covers investigation leads, adaptive follow-ups, evidence proposals,
and cross-role analyses. Grounded answers and cover letters use the
user-selected provider and are unchanged.

## Removal (restores the Meta-only integration)

No database fields, migrations, settings, engine types, or persisted
configuration were added. Persisted notices are plain text; existing rows stay
valid after removal.

1. `internal/coverletter/provider/providers.go`
   - Delete the `baseten` field on `coverBackend`.
   - Delete the `HACK-BASETEN` const block (`MuseBackendMeta`,
     `MuseBackendBaseten`, `basetenChatURL`, `BasetenMuseModel`).
   - Delete `errMuseBackend`, `SelectedMuseBackend`, the baseten branch of
     `ConfigureMeta` (keeping the `MODEL_API_KEY` Meta path), the
     `MuseBackend`/`MuseModel` accessors, `museIdentity`, and the
     `m.Backend, m.Model` metrics line.
   - `RequestMetrics.Backend`/`Model` may stay (harmless, empty for Meta) or
     be deleted with the metrics line.
2. `internal/app/career_discovery.go` — delete the `HACK-BASETEN` branch in
   `discoveryUsageNote` (the Meta estimate remains).
3. `internal/app/ai_handlers.go` — delete the two `HACK-BASETEN` display
   branches (the Meta ready/setup text remains).
4. Delete `internal/coverletter/provider/muse_backend_test.go` and
   `internal/app/discovery_usage_note_test.go` (or keep the Meta assertions
   by folding them into the existing suites).
5. Delete this file.
6. Remove `BASETEN_API_KEY` and `HACKATHON_MUSE_BACKEND` from the launch
   environment.

Then `go build ./...` and `go test ./internal/coverletter/provider/
./internal/app/` must pass with no workflow, schema, or fixture changes.

Untouched by this integration (no removal work): `career-discovery/`
engine, evidence validation, application writing, UI templates, database
schema, `~/Projects/yourmove-go`.
