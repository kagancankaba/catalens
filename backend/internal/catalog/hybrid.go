package catalog

import (
	"context"
	"fmt"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TextSearch(ctx context.Context, collection *mongo.Collection, queryText string, limit int) ([]Match, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$search", Value: bson.M{
			"index": "products_search",
			"text": bson.M{
				"query": queryText,
				"path":  bson.M{"wildcard": "*"},
			},
		}}},
		{{Key: "$limit", Value: int64(limit)}},
		{{Key: "$project", Value: bson.M{
			"_id":   0,
			"id":    bson.M{"$toString": "$_id"},
			"name":  1,
			"brand": 1,
			"score": bson.M{"$meta": "searchScore"},
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("text search aggregate: %w", err)
	}
	defer cursor.Close(ctx)

	var matches []Match
	if err := cursor.All(ctx, &matches); err != nil {
		return nil, fmt.Errorf("decode text matches: %w", err)
	}

	return matches, nil
}

func BlendScores(vectorMatches []Match, textMatches []Match) []Match {
	textScoreById := make(map[string]float64, len(textMatches))
	var maxTextScore float64
	for _, m := range textMatches {
		textScoreById[m.ID] = m.Score
		if m.Score > maxTextScore {
			maxTextScore = m.Score
		}
	}

	type ranked struct {
		match    Match
		combined float64
	}

	rankedList := make([]ranked, len(vectorMatches))
	for i, m := range vectorMatches {
		textScore := 0.0
		if maxTextScore > 0 {
			textScore = textScoreById[m.ID] / maxTextScore
		}
		rankedList[i] = ranked{match: m, combined: 0.7*m.Score + 0.3*textScore}
	}

	sort.Slice(rankedList, func(i, j int) bool {
		return rankedList[i].combined > rankedList[j].combined
	})

	result := make([]Match, len(rankedList))
	for i, r := range rankedList {
		result[i] = r.match
	}
	return result
}
