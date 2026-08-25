package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"

	"github.com/kagancankaba/catalens/internal/catalog"
	"github.com/kagancankaba/catalens/internal/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/genai"
)

var categories = []string{"sneakers", "tea", "chairs"}

const confidenceThreshold = 0.87

type RecognizeResponse struct {
	Descriptor    *catalog.Descriptor `json:"descriptor"`
	FilterApplied any                 `json:"filterApplied"`
	Matches       []catalog.Match     `json:"matches"`
	NoMatch       bool                `json:"noMatch"`
	Substitutes   []catalog.Match     `json:"substitutes"`
}

type ItemResult struct {
	Descriptor    catalog.Descriptor `json:"descriptor"`
	FilterApplied any				 `json:"filterApplied"`
	Matches 	  []catalog.Match	 `json:"matches"`
	NoMatch 	  bool				 `json:"noMatch"`
}

type RecognizeMultiResponse struct {
	Items []ItemResult `json:"items"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func main() {
	ctx := context.Background()

	if err := config.LoadEnv(".env"); err != nil {
		fmt.Println("Uyari env yuklenemedi: ", err)
	}

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		fmt.Println("MongoDB baglanti hatasi : ", err)
		return
	}
	defer mongoClient.Disconnect(ctx)

	if err := mongoClient.Ping(ctx, nil); err != nil {
		fmt.Println("Mongo ping hatasi: ", err)
		return
	}
	fmt.Println("MongoDB baglantisi basarili")

	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		fmt.Println("gemini client hatasi: ", err)
		return
	}
	fmt.Println("Gemini client hazir")

	collection := mongoClient.Database("catalens").Collection("products")

	mux := http.NewServeMux()
	mux.HandleFunc("/recognize", func(w http.ResponseWriter, r *http.Request) {
		reqCtx := r.Context()

		file, header, err := r.FormFile("image")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Image Required")
			return
		}
		defer file.Close()

		imageBytes, err := io.ReadAll(file)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Image Required")
			return
		}

		mimeType := header.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "image/jpeg"
		}

		descriptor, err := catalog.VisionDescribe(reqCtx, geminiClient, imageBytes, mimeType, categories)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Vision Unavailable")
			return
		}

		text := catalog.EmbeddingTextFromDescriptor(*descriptor)
		vector, err := catalog.Embed(reqCtx, geminiClient, text)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Vision Unavailable")
			return
		}

		matches, err := catalog.VectorSearch(reqCtx, collection, vector, descriptor.Category, 5)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "Search Unavailable")
			return
		}

		var filterApplied any = descriptor.Category
		if len(matches) == 0 && descriptor.Category != "" {
			matches, err = catalog.VectorSearch(reqCtx, collection, vector, "", 5)
			if err != nil {
				writeError(w, http.StatusServiceUnavailable, "Search Unavailable")
				return
			}
			filterApplied = nil
		}

		textMatches, err := catalog.TextSearch(reqCtx, collection, text, 10)
		if err == nil {
			matches = catalog.BlendScores(matches, textMatches)
		}

		confident := []catalog.Match{}
		for _, m := range matches {
			if m.Score >= confidenceThreshold {
				confident = append(confident, m)
			}
		}

		substitutes := []catalog.Match{}
		if len(confident) > 0 && !confident[0].InStock {
			if embedding, err := catalog.ProductEmbedding(reqCtx, collection, confident[0].ID); err == nil {
				if subs, err := catalog.FindSubstitutes(reqCtx, collection, confident[0].ID, embedding, descriptor.Category, 5); err == nil {
					substitutes = subs
				}
			}
		}

		response := RecognizeResponse{
			Descriptor:    descriptor,
			FilterApplied: filterApplied,
			Matches:       confident,
			NoMatch:       len(confident) == 0,
			Substitutes:   substitutes,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/recognize-multi", func(w http.ResponseWriter, r *http.Request) {
		reqCtx := r.Context()

		file, header, err := r.FormFile("image")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Image Required")
			return
		}
		defer file.Close()

		imageBytes, err := io.ReadAll(file)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Image Required")
			return
		}

		mimeType := header.Header.Get("Content-Type")
		if mimeType ==  "" {
			mimeType = "image/jpeg"
		}

		descriptors, err := catalog.VisionDescribeMulti(reqCtx, geminiClient, imageBytes, mimeType, categories)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Vision Unavailable")
			return
		}

				sort.Slice(descriptors, func(i, j int) bool {
			areaI := 0.0
			if descriptors[i].BoundingBox != nil {
				b := descriptors[i].BoundingBox
				areaI = (b.XMax - b.XMin) * (b.YMax - b.YMin)
			}
			areaJ := 0.0
			if descriptors[j].BoundingBox != nil {
				b := descriptors[j].BoundingBox
				areaJ = (b.XMax - b.XMin) * (b.YMax - b.YMin)
			}
			return areaI > areaJ
		})

		if len(descriptors) > 5 {
			descriptors = descriptors[:5]
		}

		items := []ItemResult{}
		for _, descriptor := range descriptors {
			text := catalog.EmbeddingTextFromDescriptor(descriptor)
			vector, err := catalog.Embed(reqCtx, geminiClient, text)
			if err != nil {
				continue
			}
		
		matches, err := catalog.VectorSearch(reqCtx, collection, vector, descriptor.Category, 5)
		if err != nil {
			continue
		}

		var filterApplied any = descriptor.Category
		if len(matches) == 0 && descriptor.Category != "" {
			matches, err = catalog.VectorSearch(reqCtx, collection, vector, "", 5)
			if err != nil {
				continue
			}
			filterApplied = nil
		}

		textMatches, err := catalog.TextSearch(reqCtx, collection, text, 10)
			if err == nil {
				matches = catalog.BlendScores(matches, textMatches)
			}

		confident := []catalog.Match{}
		for _, m := range matches {
			if m.Score >= confidenceThreshold {
				confident = append(confident, m)
			}
		}

		items = append(items, ItemResult{
				Descriptor:    descriptor,
				FilterApplied: filterApplied,
				Matches: 	   confident,
				NoMatch: 	   len(confident) == 0,
			})
		}

		response := RecognizeMultiResponse{Items: items}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	fmt.Println("Sunucu 8080 portunda dinliyor...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("sunucu hatasi: ", err)
	}
}
