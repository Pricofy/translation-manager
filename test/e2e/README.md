# E2E Tests - Translation Manager

End-to-end tests for the translation manager Lambda.

## Prerequisites

1. **Deploy the stack:**
   ```bash
   make deploy ENV=dev
   ```

2. **Deploy translator Lambdas:**
   ```bash
   cd ../translation-services
   make deploy ENV=dev
   ```

3. **AWS credentials:**
   ```bash
   export AWS_PROFILE=pricofy-dev
   ```

## Running Tests

```bash
# Run all E2E tests
npm test

# Run with verbose output
npm test -- --verbose
```

## Test Coverage

### Basic Functionality
- Single text translation (es→fr)
- Batch translation (multiple texts)
- All language pair combinations

### Chunking
- Large batch (100+ texts) splits correctly
- Results maintain order

### Error Handling
- Unsupported language pair returns error
- Invalid request returns error

### Performance
- Cold start latency
- Warm invocation latency
- Chunked batch latency

## Expected Latencies

| Scenario | Expected |
|----------|----------|
| Cold start | < 500ms |
| Single text (warm) | < 2s |
| 50 texts (warm, 1 chunk) | < 3s |
| 300 texts (warm, ~3 chunks) | < 8s |
