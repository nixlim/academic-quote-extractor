package search

import (
	"context"
	"fmt"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	weaviatemodels "github.com/weaviate/weaviate/entities/models"
)

// ChunkResult represents a search result from Weaviate
type ChunkResult struct {
	Text        string   `json:"text"`
	ChunkID     string   `json:"chunk_id"`
	DocumentID  int64    `json:"document_id"`
	PageNum     *int     `json:"page_num"`
	SectionPath []string `json:"section_path"`
	Score       float64  `json:"score"`
}

// WeaviateClient provides access to Weaviate operations
type WeaviateClient struct {
	client         *weaviate.Client
	ollamaEndpoint string
	debug          bool
}

// NewWeaviateClient creates a new Weaviate client
func NewWeaviateClient(host string, ollamaEndpoint string) (*WeaviateClient, error) {
	cfg := weaviate.Config{
		Host:   host,
		Scheme: "http",
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create weaviate client: %w", err)
	}

	return &WeaviateClient{
		client:         client,
		ollamaEndpoint: ollamaEndpoint,
		debug:          false,
	}, nil
}

// SetDebug enables or disables debug logging
func (w *WeaviateClient) SetDebug(debug bool) {
	w.debug = debug
}

// CreateSchema creates the Chunk class in Weaviate if it doesn't exist
func (w *WeaviateClient) CreateSchema(ctx context.Context) error {
	// Check if class exists
	exists, err := w.client.Schema().ClassExistenceChecker().
		WithClassName(ChunkClassName).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("check class existence: %w", err)
	}

	if exists {
		if w.debug {
			fmt.Printf("[DEBUG] Weaviate class %s already exists\n", ChunkClassName)
		}
		return nil
	}

	// Create the class
	chunkClass := GetChunkClass(w.ollamaEndpoint)
	err = w.client.Schema().ClassCreator().
		WithClass(chunkClass).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("create chunk class: %w", err)
	}

	if w.debug {
		fmt.Printf("[DEBUG] Weaviate class %s created\n", ChunkClassName)
	}

	return nil
}

// InsertChunk inserts a single chunk into Weaviate and returns its UUID
func (w *WeaviateClient) InsertChunk(ctx context.Context, chunkID string, documentID int64, text string, pageNum *int, sectionPath []string) (string, error) {
	properties := map[string]interface{}{
		"text":         text,
		"chunk_id":     chunkID,
		"document_id":  documentID,
		"section_path": sectionPath,
	}

	if pageNum != nil {
		properties["page_num"] = *pageNum
	}

	result, err := w.client.Data().Creator().
		WithClassName(ChunkClassName).
		WithProperties(properties).
		Do(ctx)
	if err != nil {
		return "", fmt.Errorf("insert chunk: %w", err)
	}

	if w.debug {
		fmt.Printf("[DEBUG] Weaviate inserted chunk %s with UUID %s\n", chunkID, result.Object.ID.String())
	}

	return result.Object.ID.String(), nil
}

// DeleteByDocumentID removes all chunks for a given document ID
func (w *WeaviateClient) DeleteByDocumentID(ctx context.Context, documentID int64) error {
	where := filters.Where().
		WithPath([]string{"document_id"}).
		WithOperator(filters.Equal).
		WithValueInt(documentID)

	result, err := w.client.Batch().ObjectsBatchDeleter().
		WithClassName(ChunkClassName).
		WithWhere(where).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("delete chunks: %w", err)
	}

	if w.debug {
		fmt.Printf("[DEBUG] Weaviate deleted %d chunks for document %d\n", result.Results.Successful, documentID)
	}

	return nil
}

// HybridSearch performs a hybrid BM25+vector search
func (w *WeaviateClient) HybridSearch(ctx context.Context, query string, limit int, alpha float32) ([]ChunkResult, error) {
	if w.debug {
		fmt.Printf("[DEBUG] Weaviate hybrid search: query=%q, limit=%d, alpha=%.2f\n", query, limit, alpha)
	}

	hybrid := w.client.GraphQL().HybridArgumentBuilder().
		WithQuery(query).
		WithAlpha(alpha)

	additional := graphql.Field{
		Name:   "_additional",
		Fields: []graphql.Field{{Name: "score"}},
	}

	fields := append(ChunkFields(), additional)

	result, err := w.client.GraphQL().Get().
		WithClassName(ChunkClassName).
		WithHybrid(hybrid).
		WithLimit(limit).
		WithFields(fields...).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("hybrid search: %w", err)
	}

	if result.Errors != nil && len(result.Errors) > 0 {
		return nil, fmt.Errorf("graphql errors: %v", result.Errors)
	}

	// Parse results
	chunks, err := parseChunkResults(result)
	if err != nil {
		return nil, fmt.Errorf("parse results: %w", err)
	}

	if w.debug {
		fmt.Printf("[DEBUG] Weaviate found %d chunks\n", len(chunks))
	}

	return chunks, nil
}

// parseChunkResults converts GraphQL response to ChunkResult slice
func parseChunkResults(result *weaviatemodels.GraphQLResponse) ([]ChunkResult, error) {
	if result.Data == nil {
		return nil, nil
	}

	getData, ok := result.Data["Get"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	chunks, ok := getData[ChunkClassName].([]interface{})
	if !ok {
		return nil, nil
	}

	var results []ChunkResult
	for _, item := range chunks {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		chunk := ChunkResult{}

		if text, ok := obj["text"].(string); ok {
			chunk.Text = text
		}
		if chunkID, ok := obj["chunk_id"].(string); ok {
			chunk.ChunkID = chunkID
		}
		if docID, ok := obj["document_id"].(float64); ok {
			chunk.DocumentID = int64(docID)
		}
		if pageNum, ok := obj["page_num"].(float64); ok {
			pn := int(pageNum)
			chunk.PageNum = &pn
		}
		if sp, ok := obj["section_path"].([]interface{}); ok {
			for _, s := range sp {
				if str, ok := s.(string); ok {
					chunk.SectionPath = append(chunk.SectionPath, str)
				}
			}
		}
		if additional, ok := obj["_additional"].(map[string]interface{}); ok {
			if score, ok := additional["score"].(float64); ok {
				chunk.Score = score
			}
		}

		results = append(results, chunk)
	}

	return results, nil
}
