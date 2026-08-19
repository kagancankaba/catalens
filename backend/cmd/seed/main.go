package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/kagancankaba/catalens/internal/catalog"
	"github.com/kagancankaba/catalens/internal/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/genai"
)

func loadProducts(path string) ([]catalog.Product, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read products file: %w", err)
	}

	var products []catalog.Product
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, fmt.Errorf("parse product json: %w", err)
	}

	return products, nil
}

func main() {

	ctx := context.Background()

	if err := config.LoadEnv(".env"); err != nil {
		fmt.Println("uyari: .env yuklenemedi", err)
	}

	products, err := loadProducts("data/products.json")
	if err != nil {
		fmt.Println("hata:", err)
		return
	}

	fmt.Println("Yuklenen urun sayisi: ", len(products))

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		fmt.Println("Mongo baglanti hatasi: ", err)
		return
	}
	defer mongoClient.Disconnect(ctx)

	if err := mongoClient.Ping(ctx, nil); err != nil {
		fmt.Println("mongo ping hatai: ", err)
		return
	}
	fmt.Println("MongoDB baglantisi basarili.")

	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		fmt.Println("gemini client hatasi: ", err)
		return
	}
	fmt.Println("Gemini Client hazir")

	collection := mongoClient.Database("catalens").Collection("products")

	for _, p := range products {
		text := catalog.EmbeddingText(p)
		hashBytes := sha256.Sum256([]byte(text))
		hash := hex.EncodeToString(hashBytes[:])

		filter := bson.M{"brand": p.Brand, "name": p.Name}

		var existing struct {
			EmbeddingHash string `bson:"embeddingHash"`
		}

		findErr := collection.FindOne(ctx, filter).Decode(&existing)
		if findErr == nil && existing.EmbeddingHash == hash {
			fmt.Println("atlandi(degismedi): ", p.Brand, p.Name)
			continue
		}

		vec, err := catalog.Embed(ctx, geminiClient, text)
		if err != nil {
			fmt.Println("embed hatasi: ", err)
			continue
		}

		update := bson.M{
			"$set": bson.M{
				"brand":         p.Brand,
				"name":          p.Name,
				"category":      p.Category,
				"price":         p.Price,
				"inStock":       p.InStock,
				"attributes":    p.Attributes,
				"embedding":     vec,
				"embeddingHash": hash,
			},
		}

		opts := options.UpdateOne().SetUpsert(true)

		_, err = collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			fmt.Println("mongo yazma hatasi: ", err)
			continue
		}
		fmt.Println("yazildi: ", p.Brand, p.Name)
	}
}
