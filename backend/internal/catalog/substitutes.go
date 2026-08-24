package catalog

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func ProductEmbedding(ctx context.Context, collection *mongo.Collection, id string) ([]float64, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid product id: %w", err)
	}

	var doc struct {
		Embedding []float64 `bson:"embedding"`
	}

	if err := collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&doc); err != nil {
		return nil, fmt.Errorf("find product: %w", err)
	}

	return doc.Embedding, nil
}

func FindSubstitutes(ctx context.Context, collection *mongo.Collection, productID string, vector []float64, category string, limit int) ([]Match, error) {
	objID, err := bson.ObjectIDFromHex(productID)
	if err != nil {
		return nil, fmt.Errorf("invalid product id: %w", err)
	}

	pipeline := mongo.Pipeline{
		{{Key: "$vectorSearch", Value: bson.M{
			"index":         "products_vec",
			"path":          "embedding",
			"queryVector":   vector,
			"numCandidates": (limit + 1) * 20,
			"limit":         limit + 1,
			"filter": bson.M{
				"inStock":  bson.M{"$eq": true},
				"category": bson.M{"$eq": category},
			},
		}}},
		{{Key: "$match", Value: bson.M{"_id": bson.M{"$ne": objID}}}},
		{{Key: "$limit", Value: int64(limit)}},
		{{Key: "$project", Value: bson.M{
			"_id":     0,
			"id":      bson.M{"$toString": "$_id"},
			"name":    1,
			"brand":   1,
			"inStock": 1,
			"score":   bson.M{"$meta": "vectorSearchScore"},
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("substitute search aggregate: %w", err)
	}
	defer cursor.Close(ctx)

	var matches []Match
	if err := cursor.All(ctx, &matches); err != nil {
		return nil, fmt.Errorf("decode substitute matches: %w", err)
	}

	return matches, nil
}
