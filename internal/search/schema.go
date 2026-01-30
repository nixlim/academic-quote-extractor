package search

import (
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	weaviatemodels "github.com/weaviate/weaviate/entities/models"
)

const ChunkClassName = "Chunk"

// GetChunkClass returns the Weaviate class definition for chunks
func GetChunkClass(ollamaEndpoint string) *weaviatemodels.Class {
	return &weaviatemodels.Class{
		Class:       ChunkClassName,
		Description: "Text chunk from academic document for semantic search",
		Vectorizer:  "text2vec-ollama",
		ModuleConfig: map[string]interface{}{
			"text2vec-ollama": map[string]interface{}{
				"model":       "nomic-embed-text",
				"apiEndpoint": ollamaEndpoint,
			},
		},
		Properties: []*weaviatemodels.Property{
			{
				Name:        "text",
				DataType:    []string{"text"},
				Description: "Chunk text content (used for vectorization)",
				ModuleConfig: map[string]interface{}{
					"text2vec-ollama": map[string]interface{}{
						"skip":                  false,
						"vectorizePropertyName": false,
					},
				},
			},
			{
				Name:        "chunk_id",
				DataType:    []string{"string"},
				Description: "Docling self_ref - links to SQLite chunks.id",
				ModuleConfig: map[string]interface{}{
					"text2vec-ollama": map[string]interface{}{
						"skip": true,
					},
				},
			},
			{
				Name:        "document_id",
				DataType:    []string{"int"},
				Description: "SQLite document ID",
				ModuleConfig: map[string]interface{}{
					"text2vec-ollama": map[string]interface{}{
						"skip": true,
					},
				},
			},
			{
				Name:        "page_num",
				DataType:    []string{"int"},
				Description: "Page number in source document",
				ModuleConfig: map[string]interface{}{
					"text2vec-ollama": map[string]interface{}{
						"skip": true,
					},
				},
			},
			{
				Name:        "section_path",
				DataType:    []string{"string[]"},
				Description: "Section heading hierarchy",
				ModuleConfig: map[string]interface{}{
					"text2vec-ollama": map[string]interface{}{
						"skip": true,
					},
				},
			},
		},
	}
}

// ChunkFields returns the GraphQL fields for chunk queries
func ChunkFields() []graphql.Field {
	return []graphql.Field{
		{Name: "text"},
		{Name: "chunk_id"},
		{Name: "document_id"},
		{Name: "page_num"},
		{Name: "section_path"},
	}
}
