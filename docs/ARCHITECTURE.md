# Architecture Documentation

**Service:** Pricofy Translation Manager  
**Version:** 1.0.0  
**Last Updated:** December 2024

---

## Table of Contents

1. [Overview](#overview)
2. [Architectural Patterns](#architectural-patterns)
3. [Component Structure](#component-structure)
4. [Data Flow](#data-flow)
5. [Design Decisions](#design-decisions)
6. [Quality Metrics](#quality-metrics)

---

## Overview

The Translation Manager is a **lightweight orchestrator** that routes translation requests to specialized translator Lambdas. It handles:

- **Request Validation** - Validate source/target language pairs
- **Token-Based Chunking** - Split large batches into memory-safe chunks
- **Lambda Routing** - Invoke correct `translator-{src}-{tgt}` Lambda
- **Result Aggregation** - Combine chunk results maintaining order

### Key Principles

1. **Single Responsibility** - Orchestration only, no translation logic
2. **Stateless** - No persistent state, pure function
3. **Memory Efficient** - 128MB sufficient for orchestration
4. **Parallel Ready** - Can invoke multiple translator Lambdas (future)

---

## Architectural Patterns

### Clean Architecture (Simplified)

```
┌─────────────────────────────────────────────────────────────┐
│                      Entry Point                              │
│                   (Lambda Handler)                            │
│  ┌────────────────────────────────────────────────────┐      │
│  │ cmd/lambda/main.go - Lambda start                  │      │
│  └────────────────────────────────────────────────────┘      │
└────────────────────────┬──────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   Handler Layer                               │
│               (Request/Response)                              │
│  ┌────────────────────────────────────────────────────┐      │
│  │ internal/handler/handler.go                        │      │
│  │  - Request validation                              │      │
│  │  - Orchestration logic                             │      │
│  │  - Response formatting                             │      │
│  └────────────────────────────────────────────────────┘      │
└────────────────────────┬──────────────────────────────────────┘
                         │
            ┌────────────┴────────────┐
            │                         │
            ▼                         ▼
┌───────────────────────┐   ┌───────────────────────┐
│     Chunker           │   │      Router           │
│  ┌─────────────────┐  │   │  ┌─────────────────┐  │
│  │ chunker.go      │  │   │  │ router.go       │  │
│  │ - Token est.    │  │   │  │ - Lambda invoke │  │
│  │ - Batch split   │  │   │  │ - Pair routing  │  │
│  └─────────────────┘  │   │  └─────────────────┘  │
└───────────────────────┘   └───────────────────────┘
```

---

## Component Structure

### Entry Point

**Location:** `cmd/lambda/main.go`

**Responsibilities:**
- Initialize Lambda runtime
- Register handler function

### Handler

**Location:** `internal/handler/handler.go`

**Responsibilities:**
- Request validation (sourceLang, targetLang, texts)
- Coordinate chunker and router
- Error handling and response formatting

### Chunker

**Location:** `internal/chunker/chunker.go`

**Responsibilities:**
- Estimate token count for texts
- Split text batches into chunks
- Ensure chunks don't exceed MAX_TOKENS

### Router

**Location:** `internal/router/router.go`

**Responsibilities:**
- Validate language pair support
- Build Lambda function name
- Invoke translator Lambda
- Parse response

---

## Data Flow

### Translation Request Flow

```
1. Lambda Event
   {texts: [...], sourceLang: "es", targetLang: "fr"}
   ↓
2. Handler validates request
   ↓
3. Handler checks pair support via Router
   ↓
4. Chunker splits texts by token count
   [chunk1: [...], chunk2: [...], ...]
   ↓
5. Router.TranslateChunks(ctx, "es", "fr", allChunks)
   → Single invocation to pricofy-translator-es-fr
   → Translator processes each chunk sequentially
   ← Receive all translations
   ↓
6. Handler flattens chunk results
   ↓
7. Lambda Response
   {translations: [...], chunksProcessed: N}
```

**Key Design:** Single Lambda invocation per request, regardless of chunks.
The translator Lambda processes chunks sequentially internally, avoiding:
- Multiple cold starts
- Lambda concurrency explosion
- Higher costs

### Token Estimation

```
text: "iPhone 12 Pro en buen estado"
      ↓
EstimateTokens(text)
      ↓
len("iPhone 12 Pro en buen estado") / 4 = 7 tokens
```

### Chunking Logic

```
Input: ["text1", "text2", ..., "textN"]
MAX_TOKENS: 3000

for each text:
  tokens = EstimateTokens(text)
  
  if currentChunkTokens + tokens > MAX_TOKENS:
    flush currentChunk
    start newChunk
  
  add text to currentChunk
  currentChunkTokens += tokens

Output: [[chunk1], [chunk2], ...]
```

---

## Design Decisions

### 1. Why Separate Manager from Translators?

**Decision:** Dedicated orchestrator Lambda vs. single fat Lambda

**Rationale:**
- **Memory Isolation:** Translator needs 384MB for ML model, manager only needs 128MB
- **Language Pair Scaling:** Deploy only pairs you need
- **Independent Updates:** Update translator model without touching orchestrator
- **Future Pivoting:** Easy to add EN pivot logic in manager

**Trade-offs:**
- Additional Lambda invocation latency (~50-100ms per chunk)
- More infrastructure to manage
- **Verdict:** Worth it for flexibility and memory optimization

### 2. Why Token-Based Chunking?

**Decision:** Chunk by estimated tokens, not by count

**Rationale:**
- **Memory Safety:** Translator Lambda has fixed memory (384MB)
- **Variable Text Length:** Product titles vary widely in length
- **Predictable:** ~4 chars/token is consistent for Latin languages

**Trade-offs:**
- Estimation is approximate (not actual tokenizer)
- May under-utilize memory with short texts
- **Verdict:** Good balance of simplicity vs. accuracy

### 3. Why 3000 Max Tokens?

**Decision:** MAX_TOKENS = 3000 per chunk

**Rationale:**
- **384MB Translator:** CTranslate2 needs ~300MB for model
- **Safety Margin:** Leave ~84MB for runtime overhead
- **Practical:** ~300 typical product titles per chunk

**Trade-offs:**
- Conservative limit may cause more chunks
- **Verdict:** Better safe than OOM

### 4. Why ARM64?

**Decision:** Use ARM64 (Graviton2) instead of x86_64

**Rationale:**
- **Cost:** 20% cheaper than x86_64
- **Performance:** Comparable or better for Go workloads
- **Go Support:** Excellent ARM64 support in Go

**Trade-offs:**
- Cannot share binaries with x86 services
- **Verdict:** Easy win for Go Lambda

### 5. Why Single Lambda Invocation for All Chunks?

**Decision:** Send all chunks in one Lambda invocation, process sequentially in translator

**Rationale:**
- **No Lambda Explosion:** 1 invocation regardless of chunk count
- **No Cold Start Storm:** Single Lambda instance handles everything
- **Cost Efficient:** Pay for 1 invocation, not N
- **Simpler Orchestration:** No need to manage concurrent invocations

**Trade-offs:**
- Longer total latency (sequential vs. parallel)
- Single Lambda timeout must accommodate all chunks
- **Verdict:** Worth it for cost and reliability

---

## Quality Metrics

### Code Quality

| Metric | Target | Current |
|--------|--------|---------|
| **Test Coverage** | > 80% | TBD |
| **Cyclomatic Complexity** | < 10 | < 5 |
| **Code Duplication** | < 3% | 0% |

### Performance

| Operation | Target | Notes |
|-----------|--------|-------|
| **Handler Latency** | < 50ms | Excluding translator invocation |
| **Chunk Processing** | 1-2s | Per chunk, depends on translator |
| **Cold Start** | < 500ms | Go runtime + AWS SDK init |

### Memory

| Metric | Allocated | Expected Usage |
|--------|-----------|----------------|
| **Memory** | 128 MB | ~30-40 MB |

---

## Dependency Graph

```
cmd/lambda/
  └── depends on → internal/handler/

internal/handler/
  ├── depends on → internal/chunker/
  └── depends on → internal/router/

internal/chunker/
  └── no dependencies

internal/router/
  └── depends on → AWS SDK Lambda
```

---

## Testing Strategy

### Unit Tests

**Location:** `internal/*_test.go`

**Tests:**
- `chunker/chunker_test.go` - Token estimation, batch splitting
- `router/router_test.go` - Lambda invocation (mocked)
- `handler/handler_test.go` - Request validation, orchestration

**Coverage Target:** 80%+

### E2E Tests

**Location:** `test/e2e/`

**Tests:**
- Real Lambda invocation
- Full translation flow
- Chunking verification with large batches

---

## Security Considerations

### IAM Permissions

- **Lambda Invoke:** Only `pricofy-translator-*` functions
- **No SSM:** No secrets needed
- **No VPC:** Runs outside VPC

### Input Validation

- Validate sourceLang/targetLang are supported
- Validate texts is non-nil array
- No size limits (chunking handles large inputs)

---

## Future Enhancements

1. **English Pivot:** Route unsupported pairs through EN
2. **Parallel Chunks:** Process chunks concurrently
3. **Caching:** Cache common translations (DynamoDB)
4. **Metrics:** CloudWatch custom metrics for chunk counts

---

**Last Updated:** December 2024  
**Version:** 1.0.0  
**Status:** ✅ Ready for Deployment
