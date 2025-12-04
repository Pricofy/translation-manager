# pricofy-translation-manager - Translation Orchestrator Service (Go)

**Repository:** https://github.com/cnebrera/pricofy  
**Purpose:** Orchestrates translation requests, routing to specialized translator Lambdas  
**Tech Stack:** Go 1.21, AWS Lambda (Custom Runtime ARM64), AWS Lambda SDK  
**Deployment:** Lambda function with direct invoke (Lambda-to-Lambda SDK)  
**Invocation:** AWS Lambda SDK (no HTTP/API Gateway)  
**Version:** 1.0.0

---

## What This Does

Routes translation requests to the appropriate `translator-{src}-{tgt}` Lambda:
1. **Chunking**: Splits large text batches into memory-safe chunks (MAX_TOKENS=3000)
2. **Routing**: Determines correct translator Lambda based on language pair
3. **Single Invocation**: Sends ALL chunks in one Lambda call (translator processes sequentially)

**Key:** One translator Lambda invocation per request, regardless of chunk count.

### Key Features

1. **Token-Based Chunking**
   - Estimates tokens (~4 chars/token for Latin languages)
   - Splits batches to fit within translator Lambda memory (384MB)
   - Preserves text order across chunks

2. **Language Pair Routing**
   - Supports 20 direct pairs: ES, IT, PT, FR, DE (all combinations)
   - Future: Pivot through EN for unsupported pairs

3. **Simple API**
   - Input: texts[], sourceLang, targetLang
   - Output: translations[] (same order)

---

## API

### Request

```json
{
  "texts": ["texto1", "texto2", ...],
  "sourceLang": "es",
  "targetLang": "fr"
}
```

### Response

```json
{
  "translations": ["traduction1", "traduction2", ...],
  "chunksProcessed": 3
}
```

### Error

```json
{
  "error": "no translator for xx→yy"
}
```

---

## Supported Languages

ES, IT, PT, FR, DE - all 20 combinations (no English):

- es↔it, es↔pt, es↔fr, es↔de
- it↔pt, it↔fr, it↔de
- pt↔fr, pt↔de
- fr↔de

---

## Integration

### From Other Lambda/Services

```go
import (
    "github.com/aws/aws-sdk-go-v2/service/lambda"
)

payload := `{"texts": ["Hola mundo"], "sourceLang": "es", "targetLang": "fr"}`
result, _ := lambdaClient.Invoke(ctx, &lambda.InvokeInput{
    FunctionName: aws.String("pricofy-translation-manager"),
    Payload:      []byte(payload),
})
```

---

## Deployment

```bash
cd translation-manager

# Build
make build

# Deploy
make deploy ENV=dev

# Test
make test-invoke ENV=dev
```

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| ENVIRONMENT | dev | Environment (dev/prod) |
| MAX_TOKENS | 3000 | Max tokens per chunk |

| Lambda | Memory | Timeout |
|--------|--------|---------|
| translation-manager | 128 MB | 60s |
| translator-{src}-{tgt} | 384 MB | 30s |

---

## Architecture

```
Client → translation-manager (Go, 128MB)
              |
              ├── Validate request
              ├── Chunk by tokens (MAX_TOKENS=3000)
              ├── Single invocation with all chunks
              └── Flatten results

          translator-{src}-{tgt} (Python, 384MB)
              |
              ├── Receives {"chunks": [[...], [...], ...]}
              ├── Processes each chunk sequentially
              └── Returns {"translations": [[...], [...], ...]}
```

**Why single invocation?**
- No Lambda explosion (1 invocation regardless of chunks)
- No cold start storm
- Cost efficient (pay for 1 Lambda, not N)

---

## Token Estimation

Uses simple heuristic for Latin languages:
- 1 token ≈ 4 characters
- 3000 tokens ≈ 12000 characters ≈ ~300 typical titles

Chunking ensures each batch fits within translator Lambda memory.

---

## Related Documentation

- Translator Lambdas: `translation-services/README.md`
- Architecture: `docs/ARCHITECTURE.md`
