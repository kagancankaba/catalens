package catalog

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type Product struct {
	Brand      string            `json:"brand"`
	Name       string            `json:"name"`
	Category   string            `json:"category"`
	Price      float64           `json:"price"`
	InStock    bool              `json:"inStock"`
	Attributes map[string]string `json:"attributes"`
}

func EmbeddingText(p Product) string {
	keys := make([]string, 0, len(p.Attributes))
	for k := range p.Attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := []string{p.Brand, p.Name, p.Category}
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, p.Attributes[k]))
	}

	return strings.Join(parts, ", ")
}

func l2normalize(vec []float64) []float64 {
	var sumSquares float64
	for _, v := range vec {
		sumSquares += v * v
	}
	norm := math.Sqrt(sumSquares)

	result := make([]float64, len(vec))
	for i, v := range vec {
		result[i] = v / norm
	}
	return result
}

func EmbeddingTextFromDescriptor(d Descriptor) string {
	sorted := make([]AttributeKV, len(d.Attributes))
	copy(sorted, d.Attributes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Key < sorted[j].Key
	})

	parts := []string{d.Brand, d.Category, d.Colour, d.Form}
	for _, kv := range sorted {
		parts = append(parts, fmt.Sprintf("%s: %s", kv.Key, kv.Value))
	}

	return strings.Join(parts, ", ")
}
