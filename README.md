# Translation Manager

Go Lambda that orchestrates translation requests. Routes to `translator-{src}-{tgt}` Lambdas.

## Architecture

```
Client → translation-manager (Go, 128MB) → translator-{src}-{tgt} (Python, 384MB)
                |
                ├── Chunking by tokens (MAX_TOKENS=3000)
                ├── Routes to correct Lambda by language pair
                └── Future: pivot through EN if no direct pair
```

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

## Supported Languages

ES, IT, PT, FR, DE (all pairs, 20 total, no English).

## Usage

```bash
# Build
make build

# Deploy
make deploy ENV=dev

# Test
make test-invoke ENV=dev
```

## Development

```bash
# Install deps
make install

# Run tests
make test

# Lint
make lint
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| ENVIRONMENT | dev | Environment (dev/prod) |
| MAX_TOKENS | 3000 | Max tokens per chunk |
