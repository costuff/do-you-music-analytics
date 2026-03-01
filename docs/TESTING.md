# Testing Guide

## Overview

The DoYou Music Analytics project follows **Test-Driven Development (TDD)** principles with comprehensive test coverage across all packages.

**Current Test Stats:**
- **96 tests** covering JSON serialization, scanner, config, API, analyzer, and logger
- **Coverage:** config (91.9%), api (29.7%), logger (32.8%), analyzer (5.6%)
- **Target:** 80%+ coverage for all packages

---

## Running Tests

### All Tests
```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run with verbose output
go test ./... -v

# Count total tests
go test ./... -v 2>&1 | grep "^=== RUN" | wc -l
```

### Specific Packages
```bash
# Database tests (requires DATABASE_URL)
DATABASE_URL="postgresql://doyou:password@localhost:5432/doyou_music" \
go test ./internal/db -v

# Config tests
go test ./internal/config -v

# API tests
go test ./internal/api -v

# Scanner tests (filesystem only, no DB needed)
go test ./internal/db -v -run "TestScan"
```

### Coverage Reports
```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# Coverage by package
go test ./... -cover | grep coverage
```

---

## Test Categories

### Unit Tests
Test individual functions in isolation:
- `internal/config/config_test.go` - Configuration validation
- `internal/analyzer/analyzer_test.go` - Audio analysis
- `pkg/logger/logger_test.go` - Logging functionality

### Integration Tests
Test multiple components together:
- `internal/db/scanner_test.go` - File scanning + database (requires DATABASE_URL)
- `internal/api/handlers_test.go` - HTTP handlers

### Edge Case Tests
Test corner cases and error handling:
- `internal/db/json_test.go` - JSON serialization, nullable fields, precision
- `internal/db/scanner_test.go` - Special characters, symlinks, permissions
- `internal/config/config_test.go` - Docker vs local paths, validation

---

## Test-Driven Development Workflow

**MANDATORY for all features:**

1. **Write failing test first** - Before writing production code
2. **Run test and verify it fails** - `go test -v ./...`
3. **Write minimal code** to make test pass
4. **Run test and verify it passes**
5. **Refactor** while keeping tests green
6. **Update test stats** in documentation
7. **Commit** test + implementation together

---

## Testing the Audio Analyzer

### Quick Test (Single Track)

```bash
# Test analyzer with a single MP3 file
VALIDATE_NAS_ON_STARTUP=false \
DATABASE_URL="postgresql://doyou:password@localhost:5432/doyou_music" \
PYTHON_PATH="$(pwd)/.venv/bin/python3" \
ANALYSIS_SCRIPT="$(pwd)/scripts/analyze_audio.py" \
go run ./cmd/test-analyzer ~/Music/test-track.mp3
```

**Expected:** Analysis takes 30-90 seconds per track (librosa is CPU-intensive).

### Verify Results

```sql
-- Check in database
SELECT t.file_name, t.analysis_status, af.bpm, af.key, af.energy 
FROM tracks t 
LEFT JOIN audio_features af ON t.id = af.track_id 
WHERE t.analysis_status = 'completed'
ORDER BY t.analyzed_at DESC LIMIT 5;
```

---

## Pre-Push Testing Checklist

**CRITICAL:** Always test Docker builds locally before pushing!

See [DEPLOYMENT.md](DEPLOYMENT.md#🧪-pre-push-local-testing) for detailed workflow.
For the short operational version, see [PREPUSH_CHECKLIST.md](PREPUSH_CHECKLIST.md).

**Quick checklist:**
1. ✅ Run all tests: `go test ./...`
2. ✅ Build API image: `docker build -f deployments/docker/Dockerfile.api -t doyou-analytics-api:local .`
3. ✅ Build Worker image: `docker build -f deployments/docker/Dockerfile.worker -t doyou-analytics-worker:local .`
4. ✅ Test locally: `docker-compose up`
5. ✅ Health check: `curl http://localhost:8080/health`
6. ✅ Check worker logs: `docker-compose logs -f worker`
7. ✅ Bump VERSION file (PATCH/MINOR/MAJOR)
8. ✅ **Only push after all checks pass**

---

## Adding New Tests

### Test File Naming
- `*_test.go` in same package as code
- Example: `analyzer.go` → `analyzer_test.go`

### Test Naming Convention
```go
func TestFunctionName_Scenario_ExpectedBehavior(t *testing.T) {
    // Example: TestAnalyzer_InvalidFile_ReturnsError
}
```

### Table-Driven Tests
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    int
        wantErr bool
    }{
        {"valid input", "test", 4, false},
        {"empty input", "", 0, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

## Testing Best Practices

- **Mock external dependencies** (database, filesystem, subprocess)
- **Use t.TempDir()** for temporary file operations
- **Test both success and error paths**
- **Keep tests independent** - no shared state
- **Use meaningful test names** - describe what's being tested
- **Add comments** for complex test logic
- **Aim for 80%+ coverage** on core business logic
- **Run tests before every commit**

---

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testing Best Practices](https://github.com/golang/go/wiki/TestComments)
