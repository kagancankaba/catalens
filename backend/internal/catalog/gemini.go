package catalog

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

func Embed(ctx context.Context, client *genai.Client, text string) ([]float64, error) {
	dim := int32(768)
	contents := []*genai.Content{
		genai.NewContentFromText(text, genai.RoleUser),
	}

	response, err := client.Models.EmbedContent(ctx, "gemini-embedding-001", contents,
		&genai.EmbedContentConfig{
			OutputDimensionality: &dim,
		})

	if err != nil {
		return nil, fmt.Errorf("embed content: %w", err)
	}

	if len(response.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	values := response.Embeddings[0].Values
	vec := make([]float64, len(values))
	for i, v := range values {
		vec[i] = float64(v)
	}

	return l2normalize(vec), nil
}
