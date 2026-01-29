# Ollama API Contract

**Service**: Ollama
**Base URL**: http://localhost:11434
**Model**: nomic-embed-text
**Purpose**: Local embedding generation via Weaviate text2vec-ollama module

## Overview

Ollama provides local embedding generation. In this architecture, **Weaviate calls Ollama directly** via the text2vec-ollama module. The Go application does not call Ollama directly for embeddings.

However, we need contract tests to verify:
1. Ollama is running and healthy
2. The embedding model is available
3. Embeddings are generated correctly

## Endpoints

### GET /api/tags

List available models.

**Response (200 OK):**
```json
{
  "models": [
    {
      "name": "nomic-embed-text:latest",
      "model": "nomic-embed-text:latest",
      "modified_at": "2024-01-15T10:30:00Z",
      "size": 274000000,
      "digest": "sha256:abc123...",
      "details": {
        "parent_model": "",
        "format": "gguf",
        "family": "nomic-bert",
        "families": ["nomic-bert"],
        "parameter_size": "137M",
        "quantization_level": "F16"
      }
    }
  ]
}
```

---

### POST /api/embed

Generate embeddings for input text.

**Request:**
```json
{
  "model": "nomic-embed-text",
  "input": "The sky is blue because of Rayleigh scattering"
}
```

**Response (200 OK):**
```json
{
  "model": "nomic-embed-text",
  "embeddings": [
    [0.123, -0.456, 0.789, ...]
  ],
  "total_duration": 123456789,
  "load_duration": 12345678,
  "prompt_eval_count": 10
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `model` | string | Model used for embedding |
| `embeddings` | array[array[float]] | Array of embedding vectors (768 dimensions for nomic-embed-text) |
| `total_duration` | int | Total processing time in nanoseconds |

---

### GET /api/version

Get Ollama version.

**Response (200 OK):**
```json
{
  "version": "0.1.26"
}
```

---

## Model Requirements

### nomic-embed-text

| Property | Value |
|----------|-------|
| Dimensions | 768 |
| Context Window | 2048 tokens |
| Size | 274 MB |
| Min Ollama Version | 0.1.26 |

### Installation

```bash
# Pull the model
docker exec -it ollama ollama pull nomic-embed-text

# Verify installation
docker exec -it ollama ollama list
```

---

## Weaviate Integration

Weaviate's text2vec-ollama module calls Ollama's `/api/embed` endpoint automatically when data is inserted. Configuration in docker-compose.yml:

```yaml
weaviate:
  environment:
    DEFAULT_VECTORIZER_MODULE: 'text2vec-ollama'
    ENABLE_MODULES: 'text2vec-ollama'
    OLLAMA_API_ENDPOINT: 'http://ollama:11434'
```

The Go application inserts text into Weaviate; Weaviate handles embedding generation transparently.

---

## Go Client (for contract tests only)

```go
// tests/contract/ollama_test.go

type OllamaClient struct {
    baseURL    string
    httpClient *http.Client
}

type EmbedRequest struct {
    Model string `json:"model"`
    Input string `json:"input"`
}

type EmbedResponse struct {
    Model      string      `json:"model"`
    Embeddings [][]float64 `json:"embeddings"`
}

type TagsResponse struct {
    Models []ModelInfo `json:"models"`
}

type ModelInfo struct {
    Name string `json:"name"`
}

// GetTags lists available models
func (c *OllamaClient) GetTags() (*TagsResponse, error)

// Embed generates embeddings (for testing only - Weaviate calls this in production)
func (c *OllamaClient) Embed(text string) ([]float64, error)

// HasModel checks if a specific model is available
func (c *OllamaClient) HasModel(name string) (bool, error)
```

---

## Contract Tests

```go
// tests/contract/ollama_test.go

func TestOllamaHealthy(t *testing.T) {
    // Given: Ollama service is running
    // When: GET /api/version
    // Then: Response contains version string
}

func TestOllamaModelAvailable(t *testing.T) {
    // Given: Ollama service is running
    // When: GET /api/tags
    // Then: nomic-embed-text is in the model list
}

func TestOllamaEmbeddingDimensions(t *testing.T) {
    // Given: nomic-embed-text model is loaded
    // When: POST /api/embed with sample text
    // Then: Response contains 768-dimensional vector
}

func TestOllamaEmbeddingConsistency(t *testing.T) {
    // Given: Same input text
    // When: POST /api/embed twice
    // Then: Embeddings are identical (deterministic)
}
```

---

## Error Handling

| Error | Cause | Resolution |
|-------|-------|------------|
| Connection refused | Ollama not running | Start Ollama container |
| Model not found | Model not pulled | Run `ollama pull nomic-embed-text` |
| Timeout | Model loading or large input | Wait for model load; reduce input size |

---

## Performance Characteristics

| Metric | Value | Notes |
|--------|-------|-------|
| Cold start | ~5s | First request loads model to memory |
| Warm embedding | ~50ms | Subsequent requests |
| Max input | 2048 tokens | Longer inputs truncated |
| Memory usage | ~500MB | With nomic-embed-text loaded |
