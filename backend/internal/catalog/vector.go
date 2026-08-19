package catalog

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Match struct {
	ID    string  `bson:"id" json:"id"`
	Name  string  `bson:"name" json:"name"`
	Brand string  `bson:"brand" json:"brand"`
	Score float64 `bson:"score" json:"score"`
}

func VectorSearch(ctx context.Context, collection *mongo.Collection, vector []float64, category string, limit int) ([]Match, error) {
	VectorSearchStage := bson.M{
		"index":         "products_vec",
		"path":          "embedding",
		"queryVector":   vector,
		"numCandidates": limit * 20,
		"limit":         limit,
	}
	if category != "" {
		VectorSearchStage["filter"] = bson.M{"category": bson.M{"$eq": category}}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$vectorSearch", Value: VectorSearchStage}},
		{{Key: "$project", Value: bson.M{
			"_id":   0,
			"id":    bson.M{"$toString": "$_id"},
			"name":  1,
			"brand": 1,
			"score": bson.M{"$meta": "vectorSearchScore"},
		}}},
	}
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("Vector search aggregate: %w", err)
	}
	defer cursor.Close(ctx)

	var matches []Match
	if err := cursor.All(ctx, &matches); err != nil {
		return nil, fmt.Errorf("decode matches: %w", err)
	}

	return matches, nil
}
