# Translation Manager

Go Lambda that orchestrates translation requests across 4 single-direction translator Lambdas.

## Architecture

```
Client → translation-manager (Go, 128MB)
              │
              ├── Chunks input (max 50 texts/chunk)
              ├── Routes to correct Lambda(s) by language pair
              └── Chains 2 Lambdas when pivoting through EN
                     │
                     ├── translator-romance-en (40+ Romance → EN)
                     ├── translator-en-romance (EN → 40+ Romance)
                     ├── translator-de-en (DE → EN)
                     └── translator-en-de (EN → DE)
```

## Supported Languages (40+)

### Core Languages

| Code | Language   |
|------|------------|
| `es` | Spanish    |
| `fr` | French     |
| `it` | Italian    |
| `pt` | Portuguese |
| `de` | German     |
| `en` | English    |

### Regional Variants

- **Spanish**: `es_AR`, `es_CL`, `es_CO`, `es_MX`, `es_ES`, `es_PE`, etc.
- **French**: `fr_BE`, `fr_CA`, `fr_FR`, `wa` (Walloon), `oc` (Occitan)
- **Italian**: `co` (Corsican), `nap` (Neapolitan), `scn` (Sicilian), `vec` (Venetian)
- **Portuguese**: `pt_BR`, `pt_PT`, `gl` (Galician), `mwl` (Mirandese)

### Extended Romance

`ca` (Catalan), `an` (Aragonese), `ro` (Romanian), `la` (Latin), `rm` (Romansh), `lld` (Ladin), `fur` (Friulian), `lij` (Ligurian), `lmo` (Lombard), `sc` (Sardinian)

## API

See [api/asyncapi.yaml](api/asyncapi.yaml) for full specification.

### Request

```json
{
  "texts": ["Hola mundo", "iPhone en buen estado"],
  "sourceLang": "es",
  "targetLang": "en"
}
```

### Response

```json
{
  "translations": ["Hello world", "iPhone in good condition"],
  "chunksProcessed": 1
}
```

### Error Response

```json
{
  "error": "unsupported language pair: zh→en"
}
```

## Routing Logic

| Source → Target     | Lambda Call(s)                           |
|---------------------|------------------------------------------|
| Romance → EN        | `romance-en` (1 call)                    |
| EN → Romance        | `en-romance` (1 call)                    |
| DE → EN             | `de-en` (1 call)                         |
| EN → DE             | `en-de` (1 call)                         |
| Romance ↔ Romance   | `romance-en` → `en-romance` (2 calls)    |
| Romance ↔ DE        | Pivot through EN (2 calls)               |

## Chunking

Input is automatically split into chunks of **50 texts** each. This ensures:

- No Lambda OOM errors (512MB memory per translator)
- Optimal batch processing performance
- ~6s per 50 texts for direct translations

## Development

### Prerequisites

- Go 1.21+
- AWS CLI configured
- Node.js 18+ (for CDK)

### Commands

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Deploy
make deploy ENV=dev

# Test deployed Lambda
make test-invoke ENV=dev
```

### Project Structure

```
translation-manager/
├── api/                    # AsyncAPI specification
├── cmd/lambda/             # Lambda entrypoint
├── internal/
│   ├── chunker/            # Text chunking logic
│   ├── domain/             # Domain models
│   ├── handler/            # Lambda handler
│   └── router/             # Language routing
├── infrastructure/         # CDK stack
├── test/e2e/               # E2E tests (TypeScript)
└── Makefile
```

## Configuration

| Variable    | Default | Description           |
|-------------|---------|----------------------|
| ENVIRONMENT | dev     | Environment (dev/prod) |

## Performance

| Batch Size  | Direct (ES→EN) | Pivot (ES→FR) |
|-------------|----------------|---------------|
| 1 text      | ~2.5s          | ~3s           |
| 50 texts    | ~6s            | ~8s           |
| 150 texts   | ~18s           | ~24s          |

*Times include cold start. Warm invocations are ~30% faster.*
