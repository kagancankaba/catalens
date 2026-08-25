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

type BoundingBox struct {
	XMin float64 `json:"xMin"`
	YMin float64 `json:"yMin"`
	XMax float64 `json:"xMax"`
	YMax float64 `json:"yMax"`
}

type Descriptor struct {
	Brand       string        `json:"brand"`
	Category    string        `json:"category"`
	Colour      string        `json:"colour"`
	Form        string        `json:"form"`
	VisibleText string        `json:"visibleText"`
	Attributes  []AttributeKV `json:"attributes"`
	BoundingBox *BoundingBox  `json:"boundingBox,omitempty"`
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
		genai.NewPartFromText("Describe this product photo for a catalog search system. Be precise and consistent: use only the primary brand name printed on the product (no sub-labels or model lines), and list all clearly visible colours as a comma-separated list ordered by prominence (e.g. 'black, red, white'), not a vague summary word like 'multicolor'. Fill every field of the schema as accurately as possible."),
		genai.NewPartFromBytes(imageBytes, mimeType),
	}
	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	temperature := float32(0.1)
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   VisionResponseSchema(categories),
		Temperature:      &temperature,
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


func VisionResponseSchemaMulti(categories []string) *genai.Schema {
	itemSchema := &genai.Schema{
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
			"boundingBox": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"xMin": {Type: genai.TypeNumber},
					"yMin": {Type: genai.TypeNumber},
					"xMax": {Type: genai.TypeNumber},
					"yMax": {Type: genai.TypeNumber},
				},
				Required: []string{"xMin", "yMin", "xMax", "yMax"},
			},
		},
		Required: []string{"brand", "category", "colour", "form", "attributes", "boundingBox"},
	}

	return &genai.Schema{
		Type:  genai.TypeArray,
		Items: itemSchema,
	}
}

func VisionDescribeMulti(ctx context.Context, client *genai.Client, imageBytes []byte, mimeType string, categories []string) ([]Descriptor, error) {
  parts := []*genai.Part{
		genai.NewPartFromText("This photo may show multiple distinct products on a shelf. Identify each distinct product separately and describe each one for a catalog search system. Be precise and consistent: use only the primary brand name printed on the product (no sub-labels or model lines), and list all clearly visible colours as a comma-separated list ordered by prominence (e.g. 'black, red, white'), not a vague summary word like 'multicolor'. Fill every field of the schema as accurately as possible for each product. Return one entry per distinct product, not per visible unit of an identical product (e.g. a shelf with five identical bottles is one entry). For each product also return a boundingBox: xMin, yMin, xMax, yMax as fractions between 0 and 1 of the image width/height, where (xMin, yMin) is the top-left corner and (xMax, yMax) is the bottom-right corner of the region containing that product."),
		genai.NewPartFromBytes(imageBytes, mimeType),
	}
  contents := []*genai.Content{
	genai.NewContentFromParts(parts, genai.RoleUser),
  }

  temperature := float32(0.1)
  config := &genai.GenerateContentConfig{
	ResponseMIMEType: "application/json",
	ResponseSchema: VisionResponseSchemaMulti(categories),
	Temperature: &temperature,
  }

  response, err := client.Models.GenerateContent(ctx, "gemini-flash-lite-latest", contents, config)
  if err != nil {
	return nil, fmt.Errorf("generate content: %w", err)
  }

  var descriptors []Descriptor
  if err := json.Unmarshal([]byte(response.Text()), &descriptors); err != nil {
	return nil, fmt.Errorf("parse descriptors: %w", err)
  }

  return descriptors, nil
}