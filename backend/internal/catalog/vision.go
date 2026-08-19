package catalog

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
)

type AttributeKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Descriptor struct {
	Brand       string        `json:"brand"`
	Category    string        `json:"category"`
	Colour      string        `json:"colour"`
	Form        string        `json:"form"`
	VisibleText string        `json:"visibleText"`
	Attributes  []AttributeKV `json:"attributes"`
}

func VisionResponseSchema(categories []string) *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"brand": {Type: genai.TypeString},
			"category": {
				Type: genai.TypeString,
				Enum: categories,
			},
			"colour":      {Type: genai.TypeString},
			"form":        {Type: genai.TypeString},
			"visibleText": {Type: genai.TypeString},
			"attributes": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"key":   {Type: genai.TypeString},
						"value": {Type: genai.TypeString},
					},
					Required: []string{"key", "value"},
				},
			},
		},
		Required: []string{"brand", "category", "colour", "form", "attributes"},
	}
}

func VisionDescribe(ctx context.Context, client *genai.Client, imageBytes []byte, mimeType string, categories []string) (*Descriptor, error) {
	parts := []*genai.Part{
		genai.NewPartFromText("Describe this product photo. Fill every field of the schema as accurately as possible."),
		genai.NewPartFromBytes(imageBytes, mimeType),
	}
	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   VisionResponseSchema(categories),
	}

	response, err := client.Models.GenerateContent(ctx, "gemini-flash-lite-latest", contents, config)
	if err != nil {
		return nil, fmt.Errorf("generate content: %w", err)
	}

	var descriptor Descriptor
	if err := json.Unmarshal([]byte(response.Text()), &descriptor); err != nil {
		return nil, fmt.Errorf("parse descriptor: %w", err)
	}

	return &descriptor, nil
}
