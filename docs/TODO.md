# DoYou Music Analytics TODO

Active roadmap for the analytics service, synchronized with the scraper integration state as of 2026-03-07.

## Next Up (Ordered)

1. [ ] **Coverage push to 80%+ (priority)**
	- `internal/analyzer`: add failure-mode tests (python errors, invalid JSON, timeout paths)
	- `internal/api`: add more pagination/filter/error response tests
	- `internal/db`: expand claim/retry/missing-status integration scenarios
2. [ ] **Export API (phase 7)**
	- Add CSV + JSON exports for analyzed tracks
	- Support filters (`status`, pagination, optional feature thresholds)
3. [ ] **Bulk re-analysis endpoint (phase 8)**
	- Trigger re-analysis for `failed` and optionally `completed` older than threshold
	- Include dry-run mode and progress reporting
4. [ ] **Worker observability and resilience**
	- Add clearer progress metrics in logs (claimed/processed/failed rates)
	- Tune retry/backoff for transient analyzer failures
5. [ ] **Dual-repo dev ergonomics**
	- Keep local workflow docs aligned with scraper integration
	- Maintain shared expectations for ports and env values

## Integration Contracts to Keep Stable

- [ ] Preserve response compatibility for:
  - `GET /api/v1/stats`
  - `GET /api/v1/stats/detailed`
  - `GET /api/v1/tracks`
- [ ] Re-validate CORS and connectivity against scraper frontend (`http://localhost:5173`) after API changes

## Documentation and Tooling

- [ ] Keep `docs/PREPUSH_CHECKLIST.md` and runtime docs synchronized
- [ ] Enforce linting with current `golangci-lint` configuration in CI
- [ ] Add short operator runbook for cleanup and scan cadence tuning

## Completed Recently

- [x] Updated NAS path handling and scanner stability in production compose
- [x] Added explicit pre-push checklist for local validation before pipeline use
- [x] Established dual-repo local workflow guidance for analytics + scraper development
