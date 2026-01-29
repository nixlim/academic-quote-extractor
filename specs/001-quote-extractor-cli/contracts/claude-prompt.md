# Claude Code CLI Contract

**Integration**: Claude Code CLI wrapper
**Purpose**: Relevance scoring and explanation generation for retrieved chunks

## Critical Constraint

> **CONSTITUTION PRINCIPLE I**: Claude MUST return chunk IDs only. The system retrieves verbatim text from SQLite. This eliminates citation hallucination.

## Invocation

```bash
claude --print --output-format json -p "$(cat prompt.txt)"
```

**Flags:**
- `--print`: Output to stdout (no interactive mode)
- `--output-format json`: Structured JSON response
- `-p`: Pass prompt text (MUST be the LAST flag - no flags after it)

**IMPORTANT**: The `-p` flag must be the last flag on the command line. The prompt text follows immediately after `-p`. No other flags should come after `-p`.

**Go Example:**
```go
cmd := exec.CommandContext(ctx, "claude",
    "--print",
    "--output-format", "json",
    "-p", prompt,  // -p MUST be last, prompt follows immediately
)
```

## Prompt Structure

### System Context

```
You are an academic research assistant. Your task is to identify quotes relevant to an essay topic.

CRITICAL RULES:
1. You MUST only reference chunks by their ID (e.g., "#/texts/42")
2. You MUST NOT generate or paraphrase quote text
3. You MUST NOT include any text from the chunks in your output
4. You MUST return valid JSON in the exact format specified

If you include any quote text in your response, it will be rejected.
```

### User Prompt Template

```
ESSAY TOPIC: {{.Topic}}

AVAILABLE CHUNKS:
{{range .Chunks}}
---
ID: {{.ChunkID}}
Page: {{.PageNum}}
Section: {{.SectionPath | join " > "}}
Text: "{{.Text}}"
---
{{end}}

TASK:
1. Review each chunk for relevance to the essay topic
2. Select chunks that would make good quotes (direct evidence, key arguments, statistics)
3. For each selected chunk, provide:
   - The exact chunk ID
   - A relevance score (0-100)
   - A 2-3 sentence explanation of why this quote supports the topic

OUTPUT FORMAT (JSON only):
{
  "selected_chunks": [
    {
      "chunk_id": "#/texts/42",
      "relevance": 85,
      "explanation": "This quote provides statistical evidence showing..."
    }
  ]
}

SCORING GUIDE:
- 80-100: Essential quote - directly supports thesis
- 60-79: Useful quote - provides relevant context or evidence
- 40-59: Marginal quote - tangentially related
- 0-39: Not relevant - do not include

Return empty array if no relevant quotes found.
```

## Expected Response

### Success Response

```json
{
  "selected_chunks": [
    {
      "chunk_id": "#/texts/15",
      "relevance": 92,
      "explanation": "Provides quantitative evidence that social media algorithms increase exposure to ideologically aligned content by 34%, directly supporting the polarization thesis."
    },
    {
      "chunk_id": "#/texts/23",
      "relevance": 78,
      "explanation": "Offers a theoretical framework for understanding echo chambers, useful for contextualizing the empirical findings."
    },
    {
      "chunk_id": "#/texts/41",
      "relevance": 65,
      "explanation": "Presents counterargument about filter bubble effects being overstated, valuable for balanced discussion."
    }
  ]
}
```

### No Matches Response

```json
{
  "selected_chunks": []
}
```

## Response Validation

The Go wrapper MUST validate responses before processing:

```go
// internal/claude/wrapper.go

type ExtractionResponse struct {
    SelectedChunks []SelectedChunk `json:"selected_chunks"`
}

type SelectedChunk struct {
    ChunkID     string `json:"chunk_id"`
    Relevance   int    `json:"relevance"`
    Explanation string `json:"explanation"`
}

func ValidateResponse(resp ExtractionResponse, validChunkIDs map[string]bool) error {
    for _, chunk := range resp.SelectedChunks {
        // 1. Verify chunk_id format
        if !strings.HasPrefix(chunk.ChunkID, "#/texts/") {
            return fmt.Errorf("invalid chunk_id format: %s", chunk.ChunkID)
        }
        
        // 2. Verify chunk_id exists in our corpus
        if !validChunkIDs[chunk.ChunkID] {
            return fmt.Errorf("unknown chunk_id: %s", chunk.ChunkID)
        }
        
        // 3. Verify relevance score bounds
        if chunk.Relevance < 0 || chunk.Relevance > 100 {
            return fmt.Errorf("relevance score out of bounds: %d", chunk.Relevance)
        }
        
        // 4. Verify explanation is not empty
        if strings.TrimSpace(chunk.Explanation) == "" {
            return fmt.Errorf("empty explanation for chunk: %s", chunk.ChunkID)
        }
    }
    return nil
}
```

## Error Handling

| Error Type | Handling |
|------------|----------|
| Invalid JSON | Retry once with simplified prompt |
| Unknown chunk_id | Log warning, skip that chunk |
| Score out of bounds | Clamp to 0-100 |
| Empty response | Return empty results (valid case) |
| CLI timeout | Return error to user |
| CLI not found | Return error with installation instructions |

## Go Wrapper Interface

```go
// internal/claude/wrapper.go

type Wrapper struct {
    cliPath string
    timeout time.Duration
}

type ExtractionTask struct {
    Topic  string       `json:"topic"`
    Chunks []ChunkInput `json:"chunks"`
}

type ChunkInput struct {
    ChunkID     string   `json:"chunk_id"`
    PageNum     int      `json:"page_num"`
    SectionPath []string `json:"section_path"`
    Text        string   `json:"text"`
}

// ExtractQuotes sends chunks to Claude for relevance scoring
// Returns only chunk IDs - verbatim text retrieved separately from SQLite
func (w *Wrapper) ExtractQuotes(ctx context.Context, task ExtractionTask) (*ExtractionResponse, error)

// buildPrompt constructs the full prompt from template
func (w *Wrapper) buildPrompt(task ExtractionTask) string
```

## Contract Tests

```go
// tests/contract/claude_test.go

func TestClaudeOutputFormat(t *testing.T) {
    // Given: A valid prompt with chunks
    // When: Claude processes the request
    // Then: Response is valid JSON with expected structure
}

func TestClaudeChunkIDOnly(t *testing.T) {
    // Given: A response from Claude
    // When: Validating the response
    // Then: No chunk text appears in explanation field
}

func TestClaudeRelevanceScoring(t *testing.T) {
    // Given: Chunks with varying relevance
    // When: Claude scores them
    // Then: Scores are within 0-100 range
}

func TestClaudeEmptyCorpus(t *testing.T) {
    // Given: No relevant chunks
    // When: Claude processes the request
    // Then: Returns empty selected_chunks array
}
```

## Security Considerations

1. **Input Sanitization**: Escape special characters in chunk text before including in prompt
2. **Output Validation**: Never trust chunk_id from response - always verify against known IDs
3. **Timeout**: Set reasonable timeout (30s default) to prevent hanging
4. **Rate Limiting**: Implement backoff if CLI returns rate limit errors
