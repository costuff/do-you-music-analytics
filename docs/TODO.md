# DoYou Music Analytics TODO

This file tracks the active roadmap for analytics development.

## Current Focus

- [ ] Raise package coverage toward 80%+ (priority: `internal/api`, `internal/analyzer`, `internal/db`)
- [ ] Stabilize Docker/local parity for NAS mount and environment defaults
- [ ] Keep scraper integration smooth (`/api/v1/stats`, `/api/v1/stats/detailed`, `/api/v1/tracks`)

## Platform and Reliability

- [ ] Add stronger worker observability (structured progress + error-rate metrics)
- [ ] Add retry/backoff policy tuning for transient analyzer errors
- [ ] Add operational runbook section for cleanup/scan tuning

## API and Product

- [ ] Add optional filters to tracks listing (feature thresholds and sort presets)
- [ ] Add export endpoint for analyzed tracks (CSV/JSON)
- [ ] Add endpoint for bulk re-analysis targeting failed/outdated tracks

## Quality and Tooling

- [ ] Enforce linting in CI with current golangci-lint config
- [ ] Keep `docs/PREPUSH_CHECKLIST.md` aligned with real local workflow
- [ ] Refresh docs examples when endpoint behavior changes

## Completed Recently

- [x] Updated NAS path handling and scanner stability in production compose
- [x] Added explicit pre-push checklist for local validation before pipeline use
