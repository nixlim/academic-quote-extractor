package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

const extractionPromptTemplate = `You are an academic research assistant helping extract relevant quotes from a corpus of academic documents.

TOPIC: {{.Topic}}

INSTRUCTIONS:
1. Review each chunk below and assess its relevance to the research topic
2. Select chunks that contain quotable passages relevant to the topic
3. Assign a relevance score (0-100) based on how directly the passage addresses the topic
4. Provide a brief explanation of why each selected passage is relevant

IMPORTANT RULES:
- Only select chunks that are genuinely relevant to the topic (score >= 60)
- The relevance score should reflect how directly useful the quote would be for academic writing
- Your explanation should be concise (1-2 sentences) and help the researcher understand the quote's value
- CONTEXT sections (marked [PREV CONTEXT] and [NEXT CONTEXT]) are provided for understanding only — do NOT select or quote from context sections
- Return ONLY the JSON response, no other text

CANDIDATE CHUNKS:
{{range .Chunks}}
---
ID: {{.ID}}
Document: {{.DocumentID}}{{if .PageNum}}
Page: {{.PageNum}}{{end}}{{if .SectionPath}}
Section: {{range $i, $s := .SectionPath}}{{if $i}} > {{end}}{{$s}}{{end}}{{end}}{{if .PrevContext}}
[PREV CONTEXT]: {{.PrevContext}}{{end}}
Text: {{.Text}}{{if .NextContext}}
[NEXT CONTEXT]: {{.NextContext}}{{end}}
---
{{end}}

Respond with a JSON object in exactly this format:
{
  "selected_chunks": [
    {
      "chunk_id": "#/chunks/0",
      "relevance": 85,
      "explanation": "Brief explanation of why this quote is relevant"
    }
  ]
}

Return an empty array if no chunks are relevant. Only include chunks with relevance >= 60.`

var extractionTemplate = template.Must(template.New("extraction").Parse(extractionPromptTemplate))

// buildPrompt creates the prompt for Claude
func (w *Wrapper) buildPrompt(task ExtractionTask) (string, error) {
	var buf bytes.Buffer

	data := struct {
		Topic  string
		Chunks []ChunkInput
	}{
		Topic:  task.Topic,
		Chunks: task.Chunks,
	}

	if err := extractionTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// BuildPromptJSON creates a prompt with chunks serialized as JSON
// This is useful when chunks contain special characters
func (w *Wrapper) BuildPromptJSON(task ExtractionTask) (string, error) {
	chunksJSON, err := json.MarshalIndent(task.Chunks, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal chunks: %w", err)
	}

	prompt := fmt.Sprintf(`You are an academic research assistant helping extract relevant quotes from a corpus of academic documents.

TOPIC: %s

INSTRUCTIONS:
1. Review each chunk below and assess its relevance to the research topic
2. Select chunks that contain quotable passages relevant to the topic
3. Assign a relevance score (0-100) based on how directly the passage addresses the topic
4. Provide a brief explanation of why each selected passage is relevant

IMPORTANT RULES:
- Only select chunks that are genuinely relevant to the topic (score >= 60)
- The relevance score should reflect how directly useful the quote would be for academic writing
- Your explanation should be concise (1-2 sentences) and help the researcher understand the quote's value
- Return ONLY the JSON response, no other text

CANDIDATE CHUNKS (JSON format):
%s

Respond with a JSON object in exactly this format:
{
  "selected_chunks": [
    {
      "chunk_id": "#/chunks/0",
      "relevance": 85,
      "explanation": "Brief explanation of why this quote is relevant"
    }
  ]
}

Return an empty array if no chunks are relevant. Only include chunks with relevance >= 60.`, task.Topic, string(chunksJSON))

	return prompt, nil
}
