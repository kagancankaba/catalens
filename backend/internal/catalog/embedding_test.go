package catalog

import (
	"fmt"
	"testing"
)

func TestEmbeddingText(t *testing.T) {
	p := Product{
		Brand:    "Nike",
		Name:     "Air Jordan 1 Low",
		Category: "sneakers",
		Attributes: map[string]string{
			"colour":   "black/red/white",
			"material": "leather",
			"form":     "low-top",
		},
	}

	result := EmbeddingText(p)
	fmt.Println(result)
}
